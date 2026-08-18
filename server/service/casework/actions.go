package casework

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"gorm.io/gorm"
)

func (s *CaseWorkService) Acknowledge(ctx context.Context, id uint, key string, req caseworkreq.AcknowledgeCase) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 || strings.TrimSpace(req.Result) == "" {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "事项标识、expectedVersion 和确认结果必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleCareSteward && decision.RoleType != caremodel.AuthorityRoleClinician {
		return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前角色不能确认关注事项")
	}
	operation := fmt.Sprintf("ACKNOWLEDGE_CASE:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req, func(tx *gorm.DB, keyDigest string) (caseworkres.ActionResult, error) {
		attentionCase, err := s.loadActionableCase(tx, decision, id)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		if attentionCase.Version != req.ExpectedVersion {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if attentionCase.Status != caseworkmodel.CaseStatusPendingAck {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseTransitionDenied, "关注事项当前状态不能确认")
		}
		if !isCurrentAssignee(attentionCase, decision.Identity.UserID) {
			return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前人员不是事项责任人")
		}
		assignment, err := currentActorAssignment(tx, attentionCase.CareClientID, decision.Identity.UserID, decision.RoleType, s.now())
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		now := s.now()
		action := caseworkmodel.CaseAction{
			AttentionCaseID: attentionCase.ID, ActionType: caseworkmodel.CaseActionAcknowledge,
			ActorID: decision.Identity.UserID, ActorRole: decision.RoleType,
			OrganizationID: assignment.OrganizationID, TeamID: assignment.TeamID,
			Source: caseworkmodel.ActionSourceStaff, Result: strings.TrimSpace(req.Result),
			FromStatus: attentionCase.Status, ToStatus: caseworkmodel.CaseStatusAcknowledged,
			OccurredAt: now, CommandKeyDigest: keyDigest, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&action).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		update := tx.Model(&caseworkmodel.AttentionCase{}).
			Where("id = ? AND version = ?", attentionCase.ID, req.ExpectedVersion).
			Updates(map[string]any{"status": caseworkmodel.CaseStatusAcknowledged, "version": gorm.Expr("version + 1")})
		if update.Error != nil {
			return caseworkres.ActionResult{}, update.Error
		}
		if update.RowsAffected != 1 {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if err = appendCaseEvent(tx, attentionCase, action, caseworkmodel.EventAttentionCaseAcknowledged); err != nil {
			return caseworkres.ActionResult{}, err
		}
		return caseworkres.ActionResult{
			ResourceID: attentionCase.ID, ActionID: action.ID, Status: caseworkmodel.CaseStatusAcknowledged,
			Version: req.ExpectedVersion + 1, OccurredAt: now,
		}, nil
	})
}

func (s *CaseWorkService) RecordHandling(ctx context.Context, id uint, key string, req caseworkreq.HandlingRecord) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 || strings.TrimSpace(req.Result) == "" || !validHandlingAction(req.ActionType) {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "事项标识、expectedVersion、动作类型和结果必填")
	}
	if req.NextStatus == "" {
		req.NextStatus = caseworkmodel.CaseStatusHandling
	}
	if !validHandlingTarget(req.NextStatus) {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "处理后的事项状态无效")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if err = authorizeHandling(decision.RoleType, req.ActionType, req.NextStatus); err != nil {
		return caseworkres.ActionResult{}, err
	}
	operation := fmt.Sprintf("RECORD_CASE_HANDLING:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req, func(tx *gorm.DB, keyDigest string) (caseworkres.ActionResult, error) {
		attentionCase, err := s.loadActionableCase(tx, decision, id)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		if attentionCase.Version != req.ExpectedVersion {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if !canRecordHandlingFrom(attentionCase.Status) {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseTransitionDenied, "关注事项当前状态不能追加处理记录")
		}
		if !isCurrentAssignee(attentionCase, decision.Identity.UserID) {
			return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前人员不是事项责任人")
		}
		assignment, err := currentActorAssignment(tx, attentionCase.CareClientID, decision.Identity.UserID, decision.RoleType, s.now())
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		now := s.now()
		action := caseworkmodel.CaseAction{
			AttentionCaseID: attentionCase.ID, ActionType: req.ActionType,
			ActorID: decision.Identity.UserID, ActorRole: decision.RoleType,
			OrganizationID: assignment.OrganizationID, TeamID: assignment.TeamID,
			Source: caseworkmodel.ActionSourceStaff, Result: strings.TrimSpace(req.Result), Reason: strings.TrimSpace(req.NextAction),
			FromStatus: attentionCase.Status, ToStatus: req.NextStatus,
			OccurredAt: now, CommandKeyDigest: keyDigest, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&action).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		updates := map[string]any{"status": req.NextStatus, "version": gorm.Expr("version + 1")}
		if req.ActionType == caseworkmodel.CaseActionHandling {
			updates["handling_result"] = strings.TrimSpace(req.Result)
		}
		if req.NextStatus == caseworkmodel.CaseStatusResolved {
			updates["resolved_at"] = now
		}
		update := tx.Model(&caseworkmodel.AttentionCase{}).
			Where("id = ? AND version = ?", attentionCase.ID, req.ExpectedVersion).Updates(updates)
		if update.Error != nil {
			return caseworkres.ActionResult{}, update.Error
		}
		if update.RowsAffected != 1 {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if err = appendCaseEvent(tx, attentionCase, action, caseworkmodel.EventAttentionHandlingRecorded); err != nil {
			return caseworkres.ActionResult{}, err
		}
		if req.NextStatus == caseworkmodel.CaseStatusResolved {
			resolveAction := action
			resolveAction.ID = 0
			resolveAction.ActionType = caseworkmodel.CaseActionResolve
			resolveAction.FromStatus = attentionCase.Status
			resolveAction.ToStatus = caseworkmodel.CaseStatusResolved
			if err = tx.Create(&resolveAction).Error; err != nil {
				return caseworkres.ActionResult{}, err
			}
			if err = appendCaseEvent(tx, attentionCase, resolveAction, caseworkmodel.EventAttentionCaseResolved); err != nil {
				return caseworkres.ActionResult{}, err
			}
		}
		return caseworkres.ActionResult{
			ResourceID: attentionCase.ID, ActionID: action.ID, Status: req.NextStatus,
			Version: req.ExpectedVersion + 1, OccurredAt: now,
		}, nil
	})
}

func (s *CaseWorkService) Escalate(ctx context.Context, id uint, key string, req caseworkreq.EscalateCase) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 || req.TargetAssigneeID == 0 || strings.TrimSpace(req.Reason) == "" {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "事项标识、expectedVersion、目标责任人和升级原因必填")
	}
	if req.DueAt != nil && !req.DueAt.After(s.now()) {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "目标时间必须晚于当前时间")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleCareSteward && decision.RoleType != caremodel.AuthorityRoleClinician {
		return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前角色不能升级关注事项")
	}
	operation := fmt.Sprintf("ESCALATE_CASE:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req, func(tx *gorm.DB, keyDigest string) (caseworkres.ActionResult, error) {
		attentionCase, err := s.loadActionableCase(tx, decision, id)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		if attentionCase.Version != req.ExpectedVersion {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if !canEscalateFrom(attentionCase.Status) {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseTransitionDenied, "关注事项当前状态不能升级")
		}
		if !isCurrentAssignee(attentionCase, decision.Identity.UserID) {
			return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前人员不是事项责任人")
		}
		actorAssignment, err := currentActorAssignment(tx, attentionCase.CareClientID, decision.Identity.UserID, decision.RoleType, s.now())
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		target, err := activeAssignmentForTarget(tx, attentionCase.CareClientID, req.TargetAssigneeID, caremodel.AssignmentRoleClinician, s.now())
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		now := s.now()
		action := caseworkmodel.CaseAction{
			AttentionCaseID: attentionCase.ID, ActionType: caseworkmodel.CaseActionEscalate,
			ActorID: decision.Identity.UserID, ActorRole: decision.RoleType,
			OrganizationID: actorAssignment.OrganizationID, TeamID: actorAssignment.TeamID,
			Source: caseworkmodel.ActionSourceStaff, Result: "已升级至责任医护", Reason: strings.TrimSpace(req.Reason),
			FromStatus: attentionCase.Status, ToStatus: caseworkmodel.CaseStatusWaitingCollaboration,
			TargetAssigneeID: &target.AssigneeID, TargetRole: target.RoleType, DueAt: req.DueAt,
			OccurredAt: now, CommandKeyDigest: keyDigest, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&action).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		if err = supersedeActiveTodo(tx, attentionCase.ID, now, "事项已升级并转交"); err != nil {
			return caseworkres.ActionResult{}, err
		}
		active := caseworkmodel.TodoActiveSlot
		todo := caseworkmodel.TodoItem{
			Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase,
			SourceID: attentionCase.ID, ActiveSlot: &active, CareClientID: attentionCase.CareClientID,
			AssigneeID: target.AssigneeID, AssigneeRole: target.RoleType, Status: caseworkmodel.TodoStatusOpen,
			OpenedAt: now, DueAt: req.DueAt, Version: 1, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&todo).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		update := tx.Model(&caseworkmodel.AttentionCase{}).
			Where("id = ? AND version = ?", attentionCase.ID, req.ExpectedVersion).
			Updates(map[string]any{
				"status":      caseworkmodel.CaseStatusWaitingCollaboration,
				"assignee_id": target.AssigneeID, "assignee_role": target.RoleType,
				"due_at": req.DueAt, "version": gorm.Expr("version + 1"),
			})
		if update.Error != nil {
			return caseworkres.ActionResult{}, update.Error
		}
		if update.RowsAffected != 1 {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if err = appendCaseEvent(tx, attentionCase, action, caseworkmodel.EventAttentionCaseEscalated); err != nil {
			return caseworkres.ActionResult{}, err
		}
		return caseworkres.ActionResult{
			ResourceID: attentionCase.ID, ActionID: action.ID, Status: caseworkmodel.CaseStatusWaitingCollaboration,
			Version: req.ExpectedVersion + 1, OccurredAt: now,
		}, nil
	})
}

func canEscalateFrom(status string) bool {
	switch status {
	case caseworkmodel.CaseStatusAcknowledged,
		caseworkmodel.CaseStatusHandling,
		caseworkmodel.CaseStatusWaitingClient,
		caseworkmodel.CaseStatusWaitingCollaboration,
		caseworkmodel.CaseStatusWaitingSupervisor:
		return true
	default:
		return false
	}
}

func activeAssignmentForTarget(db *gorm.DB, careClientID, assigneeID uint, role string, now time.Time) (caremodel.CareAssignment, error) {
	var assignment caremodel.CareAssignment
	err := db.Where("care_client_id = ? AND assignee_id = ? AND role_type = ? AND cancelled_at IS NULL AND valid_from <= ?", careClientID, assigneeID, role, now).
		Where("valid_until IS NULL OR valid_until > ?", now).
		Order("valid_from DESC, id DESC").First(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return assignment, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseResponsibilityRequired, "目标责任人不在当前有效责任链中")
	}
	return assignment, err
}

func validHandlingAction(action string) bool {
	return action == caseworkmodel.CaseActionContact || action == caseworkmodel.CaseActionHandling
}

func validHandlingTarget(status string) bool {
	switch status {
	case caseworkmodel.CaseStatusHandling,
		caseworkmodel.CaseStatusWaitingClient,
		caseworkmodel.CaseStatusWaitingCollaboration,
		caseworkmodel.CaseStatusWaitingSupervisor,
		caseworkmodel.CaseStatusResolved:
		return true
	default:
		return false
	}
}

func authorizeHandling(role, action, nextStatus string) error {
	switch role {
	case caremodel.AuthorityRoleCareSteward:
		if action != caseworkmodel.CaseActionContact ||
			(nextStatus != caseworkmodel.CaseStatusHandling && nextStatus != caseworkmodel.CaseStatusWaitingClient && nextStatus != caseworkmodel.CaseStatusWaitingCollaboration) {
			return caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "健康管家只能记录非专业联系")
		}
	case caremodel.AuthorityRoleClinician:
		if action != caseworkmodel.CaseActionHandling {
			return caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "一线医护应使用处置记录")
		}
	default:
		return caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前角色不能追加事项处理记录")
	}
	return nil
}

func canRecordHandlingFrom(status string) bool {
	switch status {
	case caseworkmodel.CaseStatusAcknowledged,
		caseworkmodel.CaseStatusHandling,
		caseworkmodel.CaseStatusWaitingClient,
		caseworkmodel.CaseStatusWaitingCollaboration,
		caseworkmodel.CaseStatusWaitingSupervisor:
		return true
	default:
		return false
	}
}

func (s *CaseWorkService) Close(ctx context.Context, id uint, key string, req caseworkreq.CloseCase) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "事项标识和 expectedVersion 必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleClinician && decision.RoleType != caremodel.AuthorityRoleSupervisor {
		return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前角色不能关闭关注事项")
	}
	operation := fmt.Sprintf("CLOSE_CASE:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req, func(tx *gorm.DB, keyDigest string) (caseworkres.ActionResult, error) {
		attentionCase, err := s.loadActionableCase(tx, decision, id)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		if attentionCase.Version != req.ExpectedVersion {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if strings.TrimSpace(attentionCase.HandlingResult) == "" {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeHandlingResultRequired, "关闭前必须记录处理结果")
		}
		if strings.TrimSpace(req.CloseReason) == "" {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCloseReasonRequired, "关闭理由必填")
		}
		if attentionCase.Status != caseworkmodel.CaseStatusResolved {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseTransitionDenied, "关注事项当前状态不能关闭")
		}
		if decision.RoleType == caremodel.AuthorityRoleClinician && !isCurrentAssignee(attentionCase, decision.Identity.UserID) {
			return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前人员不是事项责任人")
		}
		now := s.now()
		organizationID := decision.Identity.DeptID
		teamID := decision.Identity.DeptID
		if decision.RoleType == caremodel.AuthorityRoleClinician {
			assignment, assignmentErr := currentActorAssignment(tx, attentionCase.CareClientID, decision.Identity.UserID, decision.RoleType, s.now())
			if assignmentErr != nil {
				return caseworkres.ActionResult{}, assignmentErr
			}
			organizationID = assignment.OrganizationID
			teamID = assignment.TeamID
		}
		action := caseworkmodel.CaseAction{
			AttentionCaseID: attentionCase.ID, ActionType: caseworkmodel.CaseActionClose,
			ActorID: decision.Identity.UserID, ActorRole: decision.RoleType, OrganizationID: organizationID, TeamID: teamID,
			Source: caseworkmodel.ActionSourceStaff, Result: strings.TrimSpace(req.CloseReason),
			FromStatus: attentionCase.Status, ToStatus: caseworkmodel.CaseStatusClosed,
			OccurredAt: now, CommandKeyDigest: keyDigest, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&action).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		update := tx.Model(&caseworkmodel.AttentionCase{}).
			Where("id = ? AND version = ?", attentionCase.ID, req.ExpectedVersion).
			Updates(map[string]any{
				"status": caseworkmodel.CaseStatusClosed, "closed_at": now,
				"close_reason": strings.TrimSpace(req.CloseReason), "version": gorm.Expr("version + 1"),
			})
		if update.Error != nil {
			return caseworkres.ActionResult{}, update.Error
		}
		if update.RowsAffected != 1 {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if err = completeActiveTodo(tx, attentionCase.ID, now, "事项已关闭"); err != nil {
			return caseworkres.ActionResult{}, err
		}
		if err = appendCaseEvent(tx, attentionCase, action, caseworkmodel.EventAttentionCaseClosed); err != nil {
			return caseworkres.ActionResult{}, err
		}
		return caseworkres.ActionResult{
			ResourceID: attentionCase.ID, ActionID: action.ID, Status: caseworkmodel.CaseStatusClosed,
			Version: req.ExpectedVersion + 1, OccurredAt: now,
		}, nil
	})
}

func (s *CaseWorkService) Reopen(ctx context.Context, id uint, key string, req caseworkreq.ReopenCase) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 || strings.TrimSpace(req.Reason) == "" {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "事项标识、expectedVersion 和重开理由必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleSupervisor {
		return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "仅主管可以重开关注事项")
	}
	operation := fmt.Sprintf("REOPEN_CASE:%d", id)
	return runIdempotent(s, ctx, operation, decision.Identity.UserID, key, req, func(tx *gorm.DB, keyDigest string) (caseworkres.ActionResult, error) {
		attentionCase, err := s.loadActionableCase(tx, decision, id)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		if attentionCase.Version != req.ExpectedVersion {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		if attentionCase.Status != caseworkmodel.CaseStatusClosed {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseTransitionDenied, "只有已关闭的关注事项可以重开")
		}
		if attentionCase.AssigneeID == nil || *attentionCase.AssigneeID == 0 || attentionCase.AssigneeRole == "" {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseResponsibilityRequired, "关注事项缺少可恢复的责任人")
		}
		now := s.now()
		assignee, err := activeAssignmentForTarget(tx, attentionCase.CareClientID, *attentionCase.AssigneeID, attentionCase.AssigneeRole, now)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
		action := caseworkmodel.CaseAction{
			AttentionCaseID: attentionCase.ID, ActionType: caseworkmodel.CaseActionReopen,
			ActorID: decision.Identity.UserID, ActorRole: decision.RoleType,
			OrganizationID: decision.Identity.DeptID, TeamID: decision.Identity.DeptID,
			Source: caseworkmodel.ActionSourceStaff,
			Result: "已重新打开关注事项", Reason: strings.TrimSpace(req.Reason),
			FromStatus: attentionCase.Status, ToStatus: caseworkmodel.CaseStatusHandling,
			OccurredAt: now, CommandKeyDigest: keyDigest, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&action).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		update := tx.Model(&caseworkmodel.AttentionCase{}).
			Where("id = ? AND version = ?", attentionCase.ID, req.ExpectedVersion).
			Updates(map[string]any{
				"status": caseworkmodel.CaseStatusHandling, "handling_result": "", "close_reason": "",
				"resolved_at": nil, "closed_at": nil, "version": gorm.Expr("version + 1"),
			})
		if update.Error != nil {
			return caseworkres.ActionResult{}, update.Error
		}
		if update.RowsAffected != 1 {
			return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
		}
		active := caseworkmodel.TodoActiveSlot
		todo := caseworkmodel.TodoItem{
			Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase,
			SourceID: attentionCase.ID, ActiveSlot: &active, CareClientID: attentionCase.CareClientID,
			AssigneeID: assignee.AssigneeID, AssigneeRole: assignee.RoleType,
			Status: caseworkmodel.TodoStatusOpen, OpenedAt: now, Version: 1, Synthetic: attentionCase.Synthetic,
		}
		if err = tx.Create(&todo).Error; err != nil {
			return caseworkres.ActionResult{}, err
		}
		if err = appendCaseEvent(tx, attentionCase, action, caseworkmodel.EventAttentionCaseReopened); err != nil {
			return caseworkres.ActionResult{}, err
		}
		return caseworkres.ActionResult{
			ResourceID: attentionCase.ID, ActionID: action.ID, Status: caseworkmodel.CaseStatusHandling,
			Version: req.ExpectedVersion + 1, OccurredAt: now,
		}, nil
	})
}

func (s *CaseWorkService) loadActionableCase(db *gorm.DB, decision *accesspolicy.CareClientDecision, id uint) (caseworkmodel.AttentionCase, error) {
	var attentionCase caseworkmodel.AttentionCase
	query := decision.ScopeAttentionCases(locking(db).Model(&caseworkmodel.AttentionCase{}), s.now())
	err := query.Where("attention_cases.id = ?", id).First(&attentionCase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return attentionCase, caseworkmodel.NewForbiddenError(caseworkmodel.CodeAccessScopeDenied, "关注事项不存在或不在当前访问范围")
	}
	return attentionCase, err
}

func isCurrentAssignee(attentionCase caseworkmodel.AttentionCase, actorID uint) bool {
	return attentionCase.AssigneeID != nil && *attentionCase.AssigneeID == actorID
}

func currentActorAssignment(db *gorm.DB, careClientID, actorID uint, role string, now time.Time) (caremodel.CareAssignment, error) {
	assignmentRole := caremodel.AssignmentRoleCareSteward
	if role == caremodel.AuthorityRoleClinician {
		assignmentRole = caremodel.AssignmentRoleClinician
	}
	var assignment caremodel.CareAssignment
	err := db.Where("care_client_id = ? AND assignee_id = ? AND role_type = ? AND cancelled_at IS NULL AND valid_from <= ?", careClientID, actorID, assignmentRole, now).
		Where("valid_until IS NULL OR valid_until > ?", now).
		Order("valid_from DESC, id DESC").First(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return assignment, caseworkmodel.NewForbiddenError(caseworkmodel.CodeCaseResponsibilityRequired, "当前人员缺少有效责任关系")
	}
	return assignment, err
}

func completeActiveTodo(db *gorm.DB, caseID uint, now time.Time, note string) error {
	return db.Model(&caseworkmodel.TodoItem{}).
		Where("source_type = ? AND source_id = ? AND active_slot = ?", caseworkmodel.TodoSourceAttentionCase, caseID, caseworkmodel.TodoActiveSlot).
		Updates(map[string]any{
			"status": caseworkmodel.TodoStatusCompleted, "active_slot": nil,
			"completed_at": now, "completion_note": note, "version": gorm.Expr("version + 1"),
		}).Error
}

func supersedeActiveTodo(db *gorm.DB, caseID uint, now time.Time, note string) error {
	return db.Model(&caseworkmodel.TodoItem{}).
		Where("source_type = ? AND source_id = ? AND active_slot = ?", caseworkmodel.TodoSourceAttentionCase, caseID, caseworkmodel.TodoActiveSlot).
		Updates(map[string]any{
			"status": caseworkmodel.TodoStatusSuperseded, "active_slot": nil,
			"completed_at": now, "completion_note": note, "version": gorm.Expr("version + 1"),
		}).Error
}

func appendCaseEvent(db *gorm.DB, attentionCase caseworkmodel.AttentionCase, action caseworkmodel.CaseAction, eventType string) error {
	return platformoutbox.Append(db, platformoutbox.AppendInput{
		EventType: eventType, AggregateType: "AttentionCase", AggregateID: attentionCase.ID,
		Payload: map[string]any{
			"attentionCaseId": attentionCase.ID, "careClientId": attentionCase.CareClientID,
			"taskId": attentionCase.TaskID, "actionId": action.ID, "actorId": action.ActorID,
			"actorRole": action.ActorRole, "fromStatus": action.FromStatus, "toStatus": action.ToStatus,
			"synthetic": attentionCase.Synthetic,
		},
		OccurredAt: action.OccurredAt, CausationID: fmt.Sprintf("case-action:%d", action.ID),
		Synthetic: attentionCase.Synthetic,
	})
}

func normalizeAccessError(err error) error {
	var domainErr *caremodel.DomainError
	if !errors.As(err, &domainErr) {
		return err
	}
	return &caseworkmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
}
