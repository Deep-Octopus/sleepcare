package casework

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
)

func (s *CaseWorkService) AssignConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.AssignConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Reason = trimmed(req.Reason)
	if id == 0 || req.ExpectedVersion == 0 || req.TargetAssigneeID == 0 || req.Reason == "" {
		return caseworkres.ConsultationActionResult{}, invalidConsultationAction("咨询标识、版本、目标人员和分配原因必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	if !decision.CanManage() {
		return caseworkres.ConsultationActionResult{}, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeConsultationAssigneeRequired,
			"仅上级可以分配待分配咨询",
		)
	}
	operation := fmt.Sprintf("consultation:assign:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if consultation.Status != caseworkmodel.ConsultationStatusWaitingAssignment {
				return caseworkres.ConsultationActionResult{}, transitionDenied("只有待分配咨询可以执行分配")
			}
			if _, err = activeCareAssignment(
				tx,
				consultation.CareClientID,
				req.TargetAssigneeID,
				req.TargetRole,
				s.now(),
			); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return s.applyConsultationAssignment(
				tx,
				consultation,
				keyDigest,
				decision.Identity.UserID,
				decision.RoleType,
				req.TargetAssigneeID,
				req.TargetRole,
				caseworkmodel.ConsultationActionAssign,
				req.Reason,
				caseworkmodel.ConsultationStatusAssigned,
				caseworkmodel.EventConsultationAssigned,
			)
		})
}

func (s *CaseWorkService) ReplyConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.ReplyConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Message = trimmed(req.Message)
	if id == 0 || req.ExpectedVersion == 0 || req.Message == "" ||
		(req.NextStatus != caseworkmodel.ConsultationStatusHandling && req.NextStatus != caseworkmodel.ConsultationStatusWaitingClient) {
		return caseworkres.ConsultationActionResult{}, invalidConsultationAction("咨询标识、版本、回复和后续状态无效")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	operation := fmt.Sprintf("consultation:reply:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if !isCurrentConsultationAssignee(consultation, decision.Identity.UserID) {
				return caseworkres.ConsultationActionResult{}, assigneeRequired("只有当前责任人可以回复咨询")
			}
			if !consultationCanBeHandled(consultation.Status) {
				return caseworkres.ConsultationActionResult{}, transitionDenied("当前状态不能回复咨询")
			}
			now := s.now()
			updates := map[string]any{
				"status":  req.NextStatus,
				"version": gorm.Expr("version + 1"),
			}
			if consultation.FirstRespondedAt == nil {
				updates["first_responded_at"] = now
				consultation.FirstRespondedAt = &now
			}
			interaction := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionReply,
				ActorType:        caseworkmodel.ConsultationActorStaff,
				ActorID:          decision.Identity.UserID,
				ActorRole:        decision.RoleType,
				Content:          req.Message,
				FromStatus:       consultation.Status,
				ToStatus:         req.NextStatus,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        consultation.Synthetic,
			}
			if err = updateConsultation(tx, &consultation, updates); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = tx.Create(&interaction).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = appendConsultationEvent(tx, consultation, interaction, caseworkmodel.EventConsultationReplied); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return consultationActionResult(consultation, interaction), nil
		})
}

func (s *CaseWorkService) TransferConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.TransferConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Reason = trimmed(req.Reason)
	if id == 0 || req.ExpectedVersion == 0 || req.TargetAssigneeID == 0 || req.Reason == "" {
		return caseworkres.ConsultationActionResult{}, invalidConsultationAction("咨询标识、版本、目标人员和转交原因必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	operation := fmt.Sprintf("consultation:transfer:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if !isCurrentConsultationAssignee(consultation, decision.Identity.UserID) {
				return caseworkres.ConsultationActionResult{}, assigneeRequired("只有当前责任人可以转交咨询")
			}
			if !consultationCanBeHandled(consultation.Status) {
				return caseworkres.ConsultationActionResult{}, transitionDenied("当前状态不能转交咨询")
			}
			if req.TargetAssigneeID == decision.Identity.UserID {
				return caseworkres.ConsultationActionResult{}, transitionDenied("不能转交给当前责任人")
			}
			if _, err = activeCareAssignment(
				tx,
				consultation.CareClientID,
				req.TargetAssigneeID,
				req.TargetRole,
				s.now(),
			); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return s.applyConsultationAssignment(
				tx,
				consultation,
				keyDigest,
				decision.Identity.UserID,
				decision.RoleType,
				req.TargetAssigneeID,
				req.TargetRole,
				caseworkmodel.ConsultationActionTransfer,
				req.Reason,
				caseworkmodel.ConsultationStatusWaitingCollaboration,
				caseworkmodel.EventConsultationTransferred,
			)
		})
}

func (s *CaseWorkService) EscalateConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.EscalateConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Reason = trimmed(req.Reason)
	if id == 0 || req.ExpectedVersion == 0 || req.TargetAssigneeID == 0 || req.Reason == "" {
		return caseworkres.ConsultationActionResult{}, invalidConsultationAction("咨询标识、版本、目标人员和升级原因必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType == caremodel.AuthorityRoleSupervisor {
		return caseworkres.ConsultationActionResult{}, transitionDenied("上级不能继续升级咨询")
	}
	operation := fmt.Sprintf("consultation:escalate:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if !isCurrentConsultationAssignee(consultation, decision.Identity.UserID) {
				return caseworkres.ConsultationActionResult{}, assigneeRequired("只有当前责任人可以升级咨询")
			}
			if !consultationCanBeHandled(consultation.Status) {
				return caseworkres.ConsultationActionResult{}, transitionDenied("当前状态不能升级咨询")
			}
			if req.TargetAssigneeID == decision.Identity.UserID {
				return caseworkres.ConsultationActionResult{}, transitionDenied("不能升级给当前责任人")
			}
			targetRole := caremodel.AssignmentRoleClinician
			if decision.RoleType == caremodel.AuthorityRoleCareSteward {
				_, err = activeCareAssignment(
					tx,
					consultation.CareClientID,
					req.TargetAssigneeID,
					targetRole,
					s.now(),
				)
			} else {
				targetRole = caremodel.AuthorityRoleSupervisor
				err = validateConsultationSupervisor(tx, consultation, req.TargetAssigneeID)
			}
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return s.applyConsultationAssignment(
				tx,
				consultation,
				keyDigest,
				decision.Identity.UserID,
				decision.RoleType,
				req.TargetAssigneeID,
				targetRole,
				caseworkmodel.ConsultationActionEscalate,
				req.Reason,
				caseworkmodel.ConsultationStatusWaitingCollaboration,
				caseworkmodel.EventConsultationEscalated,
			)
		})
}

func (s *CaseWorkService) ResolveConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.ResolveConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Resolution = trimmed(req.Resolution)
	req.FollowUpPlan = trimmed(req.FollowUpPlan)
	if id == 0 || req.ExpectedVersion == 0 || req.Resolution == "" {
		return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
			caseworkmodel.CodeConsultationResolutionRequired,
			"咨询标识、版本和解决结果必填",
		)
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	operation := fmt.Sprintf("consultation:resolve:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if !isCurrentConsultationAssignee(consultation, decision.Identity.UserID) {
				return caseworkres.ConsultationActionResult{}, assigneeRequired("只有当前责任人可以记录解决结果")
			}
			if !consultationCanBeHandled(consultation.Status) {
				return caseworkres.ConsultationActionResult{}, transitionDenied("当前状态不能标记已解决")
			}
			now := s.now()
			interaction := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionResolve,
				ActorType:        caseworkmodel.ConsultationActorStaff,
				ActorID:          decision.Identity.UserID,
				ActorRole:        decision.RoleType,
				Content:          req.Resolution,
				Reason:           req.FollowUpPlan,
				FromStatus:       consultation.Status,
				ToStatus:         caseworkmodel.ConsultationStatusResolved,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        consultation.Synthetic,
			}
			updates := map[string]any{
				"status":         caseworkmodel.ConsultationStatusResolved,
				"resolution":     req.Resolution,
				"follow_up_plan": req.FollowUpPlan,
				"resolved_at":    now,
				"version":        gorm.Expr("version + 1"),
			}
			if err = updateConsultation(tx, &consultation, updates); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = tx.Create(&interaction).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = appendConsultationEvent(tx, consultation, interaction, caseworkmodel.EventConsultationResolved); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return consultationActionResult(consultation, interaction), nil
		})
}

func (s *CaseWorkService) CloseConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.CloseConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.CloseReason = trimmed(req.CloseReason)
	if id == 0 || req.ExpectedVersion == 0 || req.CloseReason == "" {
		return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
			caseworkmodel.CodeConsultationCloseReasonRequired,
			"咨询标识、版本和关闭理由必填",
		)
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	operation := fmt.Sprintf("consultation:close:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if !decision.CanManage() && !isCurrentConsultationAssignee(consultation, decision.Identity.UserID) {
				return caseworkres.ConsultationActionResult{}, assigneeRequired("只有当前责任人或上级可以关闭咨询")
			}
			if consultation.Status != caseworkmodel.ConsultationStatusResolved || strings.TrimSpace(consultation.Resolution) == "" {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
					caseworkmodel.CodeConsultationResolutionRequired,
					"记录解决结果后才能关闭咨询",
				)
			}
			now := s.now()
			interaction := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionClose,
				ActorType:        caseworkmodel.ConsultationActorStaff,
				ActorID:          decision.Identity.UserID,
				ActorRole:        decision.RoleType,
				Content:          "本次咨询已关闭",
				Reason:           req.CloseReason,
				FromStatus:       consultation.Status,
				ToStatus:         caseworkmodel.ConsultationStatusClosed,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        consultation.Synthetic,
			}
			updates := map[string]any{
				"status":       caseworkmodel.ConsultationStatusClosed,
				"close_reason": req.CloseReason,
				"closed_at":    now,
				"version":      gorm.Expr("version + 1"),
			}
			if err = updateConsultation(tx, &consultation, updates); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = tx.Create(&interaction).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = completeConsultationTodo(tx, consultation.ID, now, req.CloseReason); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = appendConsultationEvent(tx, consultation, interaction, caseworkmodel.EventConsultationClosed); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return consultationActionResult(consultation, interaction), nil
		})
}

func (s *CaseWorkService) ReopenConsultation(
	ctx context.Context,
	id uint,
	key string,
	req caseworkreq.ReopenConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Reason = trimmed(req.Reason)
	if id == 0 || req.ExpectedVersion == 0 || req.Reason == "" {
		return caseworkres.ConsultationActionResult{}, invalidConsultationAction("咨询标识、版本和重开原因必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ConsultationActionResult{}, normalizeAccessError(err)
	}
	if !decision.CanManage() {
		return caseworkres.ConsultationActionResult{}, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeConsultationAssigneeRequired,
			"仅上级可以重开咨询",
		)
	}
	operation := fmt.Sprintf("consultation:reopen:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			consultation, err := s.loadActionableConsultation(tx, decision, id)
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = ensureConsultationVersion(consultation, req.ExpectedVersion); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if consultation.Status != caseworkmodel.ConsultationStatusClosed {
				return caseworkres.ConsultationActionResult{}, transitionDenied("只有已关闭咨询可以重开")
			}
			if consultation.AssigneeID == nil || consultation.AssigneeRole == "" {
				return caseworkres.ConsultationActionResult{}, assigneeRequired("咨询缺少可恢复的责任人")
			}
			now := s.now()
			if consultation.AssigneeRole == caremodel.AuthorityRoleSupervisor {
				err = validateConsultationSupervisor(tx, consultation, *consultation.AssigneeID)
			} else {
				_, err = activeCareAssignment(
					tx,
					consultation.CareClientID,
					*consultation.AssigneeID,
					consultation.AssigneeRole,
					now,
				)
			}
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			interaction := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionReopen,
				ActorType:        caseworkmodel.ConsultationActorStaff,
				ActorID:          decision.Identity.UserID,
				ActorRole:        decision.RoleType,
				Content:          "服务团队已重新开始处理",
				Reason:           req.Reason,
				FromStatus:       consultation.Status,
				ToStatus:         caseworkmodel.ConsultationStatusAssigned,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        consultation.Synthetic,
			}
			updates := map[string]any{
				"status":         caseworkmodel.ConsultationStatusAssigned,
				"resolution":     "",
				"follow_up_plan": "",
				"close_reason":   "",
				"resolved_at":    nil,
				"closed_at":      nil,
				"version":        gorm.Expr("version + 1"),
			}
			if err = updateConsultation(tx, &consultation, updates); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = tx.Create(&interaction).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = createConsultationTodo(tx, consultation, now); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = appendConsultationEvent(tx, consultation, interaction, caseworkmodel.EventConsultationReopened); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return consultationActionResult(consultation, interaction), nil
		})
}

func (s *CaseWorkService) applyConsultationAssignment(
	tx *gorm.DB,
	consultation caseworkmodel.Consultation,
	keyDigest string,
	actorID uint,
	actorRole string,
	targetID uint,
	targetRole string,
	actionType string,
	reason string,
	toStatus string,
	eventType string,
) (caseworkres.ConsultationActionResult, error) {
	now := s.now()
	fromStatus := consultation.Status
	if consultation.AssigneeID != nil {
		if err := supersedeConsultationTodo(tx, consultation.ID, now, reason); err != nil {
			return caseworkres.ConsultationActionResult{}, err
		}
	}
	updates := map[string]any{
		"status":        toStatus,
		"assignee_id":   targetID,
		"assignee_role": targetRole,
		"version":       gorm.Expr("version + 1"),
	}
	if err := updateConsultation(tx, &consultation, updates); err != nil {
		return caseworkres.ConsultationActionResult{}, err
	}
	interaction := caseworkmodel.ConsultationInteraction{
		ConsultationID:   consultation.ID,
		ActionType:       actionType,
		ActorType:        caseworkmodel.ConsultationActorStaff,
		ActorID:          actorID,
		ActorRole:        actorRole,
		Reason:           reason,
		FromStatus:       fromStatus,
		ToStatus:         toStatus,
		TargetAssigneeID: &targetID,
		TargetRole:       targetRole,
		ClientVisible:    false,
		OccurredAt:       now,
		CommandKeyDigest: keyDigest,
		Synthetic:        consultation.Synthetic,
	}
	if err := tx.Create(&interaction).Error; err != nil {
		return caseworkres.ConsultationActionResult{}, err
	}
	if err := createConsultationTodo(tx, consultation, now); err != nil {
		return caseworkres.ConsultationActionResult{}, err
	}
	if err := appendConsultationEvent(tx, consultation, interaction, eventType); err != nil {
		return caseworkres.ConsultationActionResult{}, err
	}
	return consultationActionResult(consultation, interaction), nil
}

func (s *CaseWorkService) loadActionableConsultation(
	db *gorm.DB,
	decision *accesspolicy.CareClientDecision,
	id uint,
) (caseworkmodel.Consultation, error) {
	var consultation caseworkmodel.Consultation
	query := decision.ScopeConsultations(locking(db).Model(&caseworkmodel.Consultation{}), s.now()).
		Where("consultations.synthetic = ?", true)
	err := query.Where("consultations.id = ?", id).First(&consultation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return consultation, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeAccessScopeDenied,
			"咨询不存在或不在当前访问范围",
		)
	}
	return consultation, err
}

func updateConsultation(db *gorm.DB, consultation *caseworkmodel.Consultation, updates map[string]any) error {
	previousVersion := consultation.Version
	update := db.Model(&caseworkmodel.Consultation{}).
		Where("id = ? AND version = ?", consultation.ID, previousVersion).
		Updates(updates)
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "咨询版本已变化")
	}
	if status, ok := updates["status"].(string); ok {
		consultation.Status = status
	}
	if assigneeID, ok := updates["assignee_id"].(uint); ok {
		consultation.AssigneeID = &assigneeID
	}
	if assigneeRole, ok := updates["assignee_role"].(string); ok {
		consultation.AssigneeRole = assigneeRole
	}
	consultation.Version++
	return nil
}

func validateConsultationSupervisor(
	db *gorm.DB,
	consultation caseworkmodel.Consultation,
	targetID uint,
) error {
	var client caremodel.CareClient
	if err := db.Set("data_scope:skip", true).Where("id = ?", consultation.CareClientID).First(&client).Error; err != nil {
		return err
	}
	var user system.SysUser
	if err := db.Set("data_scope:skip", true).
		Where("id = ? AND enable = ?", targetID, 1).First(&user).Error; err != nil {
		return assigneeRequired("目标上级账号不可用")
	}
	var profile caremodel.CareAuthorityProfile
	err := db.Set("data_scope:skip", true).
		Where("authority_id = ? AND role_type = ? AND active = ?", user.AuthorityId, caremodel.AuthorityRoleSupervisor, true).
		First(&profile).Error
	if err != nil || user.DeptId != client.OrganizationID {
		return assigneeRequired("目标人员不是当前机构的有效上级")
	}
	return nil
}

func consultationCanBeHandled(status string) bool {
	switch status {
	case caseworkmodel.ConsultationStatusAssigned,
		caseworkmodel.ConsultationStatusHandling,
		caseworkmodel.ConsultationStatusWaitingClient,
		caseworkmodel.ConsultationStatusWaitingCollaboration:
		return true
	default:
		return false
	}
}

func isCurrentConsultationAssignee(consultation caseworkmodel.Consultation, actorID uint) bool {
	return consultation.AssigneeID != nil && *consultation.AssigneeID == actorID
}

func ensureConsultationVersion(consultation caseworkmodel.Consultation, expected uint) error {
	if consultation.Version != expected {
		return caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "咨询版本已变化")
	}
	return nil
}

func invalidConsultationAction(message string) error {
	return caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, message)
}

func transitionDenied(message string) error {
	return caseworkmodel.NewDomainError(caseworkmodel.CodeConsultationTransitionDenied, message)
}

func assigneeRequired(message string) error {
	return caseworkmodel.NewForbiddenError(caseworkmodel.CodeConsultationAssigneeRequired, message)
}
