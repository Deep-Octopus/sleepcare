package casework

import (
	"context"
	"errors"
	"fmt"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"gorm.io/gorm"
)

func (s *CaseWorkService) CreateClientConsultation(
	ctx context.Context,
	identity ClientConsultationIdentity,
	key string,
	req caseworkreq.CreateConsultation,
) (caseworkres.ConsultationActionResult, error) {
	req.Subject = trimmed(req.Subject)
	req.Message = trimmed(req.Message)
	if identity.CareClientID == 0 || identity.DeptID == 0 || !identity.Synthetic || !s.syntheticFixturesEnabled() ||
		req.Subject == "" || req.Message == "" || !validConsultationUrgency(req.Urgency) {
		return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
			caseworkmodel.CodeInvalidArgument,
			"咨询主题、问题和联系优先级必填",
		)
	}
	return runIdempotent(s, ctx, "client-consultation:create", identity.CareClientID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			var client caremodel.CareClient
			if err := tx.Where(
				"id = ? AND dept_id = ? AND status = ? AND synthetic = ?",
				identity.CareClientID,
				identity.DeptID,
				caremodel.ClientStatusActive,
				true,
			).First(&client).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewForbiddenError(
					caseworkmodel.CodeAccessScopeDenied,
					"当前会话不能创建咨询",
				)
			}
			now := s.now()
			consultation := caseworkmodel.Consultation{
				CareClientID:    identity.CareClientID,
				Source:          caseworkmodel.ConsultationSourceOnline,
				Subject:         req.Subject,
				InitialQuestion: req.Message,
				Urgency:         req.Urgency,
				Status:          caseworkmodel.ConsultationStatusWaitingAssignment,
				OpenedAt:        now,
				Version:         1,
				Synthetic:       true,
			}
			if err := tx.Create(&consultation).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			created := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionCreate,
				ActorType:        caseworkmodel.ConsultationActorClient,
				ActorID:          identity.CareClientID,
				Content:          req.Message,
				FromStatus:       caseworkmodel.ConsultationStatusNew,
				ToStatus:         caseworkmodel.ConsultationStatusWaitingAssignment,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        true,
			}
			if err := tx.Create(&created).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err := appendConsultationEvent(tx, consultation, created, caseworkmodel.EventConsultationCreated); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}

			assignment, err := currentCareAssignment(
				tx,
				identity.CareClientID,
				caremodel.AssignmentRoleCareSteward,
				now,
			)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return consultationActionResult(consultation, created), nil
			}
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			consultation.Status = caseworkmodel.ConsultationStatusAssigned
			consultation.AssigneeID = &assignment.AssigneeID
			consultation.AssigneeRole = assignment.RoleType
			consultation.Version++
			update := tx.Model(&caseworkmodel.Consultation{}).
				Where("id = ? AND version = ?", consultation.ID, consultation.Version-1).
				Updates(map[string]any{
					"status":        consultation.Status,
					"assignee_id":   assignment.AssigneeID,
					"assignee_role": assignment.RoleType,
					"version":       consultation.Version,
				})
			if update.Error != nil {
				return caseworkres.ConsultationActionResult{}, update.Error
			}
			if update.RowsAffected != 1 {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
					caseworkmodel.CodeVersionConflict,
					"咨询版本已变化",
				)
			}
			assigned := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionAssign,
				ActorType:        caseworkmodel.ConsultationActorSystem,
				ActorID:          identity.CareClientID,
				Content:          "已进入服务团队待办",
				FromStatus:       caseworkmodel.ConsultationStatusWaitingAssignment,
				ToStatus:         consultation.Status,
				TargetAssigneeID: &assignment.AssigneeID,
				TargetRole:       assignment.RoleType,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        true,
			}
			if err = tx.Create(&assigned).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = createConsultationTodo(tx, consultation, now); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = appendConsultationEvent(tx, consultation, assigned, caseworkmodel.EventConsultationAssigned); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return consultationActionResult(consultation, assigned), nil
		})
}

func (s *CaseWorkService) ListClientConsultations(
	ctx context.Context,
	identity ClientConsultationIdentity,
	req caseworkreq.ClientConsultationSearch,
) ([]caseworkres.ClientConsultationSummary, int64, error) {
	if identity.CareClientID == 0 || identity.DeptID == 0 || !identity.Synthetic || !s.syntheticFixturesEnabled() {
		return nil, 0, caseworkmodel.NewForbiddenError(caseworkmodel.CodeAccessScopeDenied, "当前会话不能查看咨询")
	}
	if req.Status != "" && !validConsultationStatus(req.Status) {
		return nil, 0, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "咨询状态无效")
	}
	query := s.db().WithContext(ctx).Model(&caseworkmodel.Consultation{}).
		Where("care_client_id = ? AND synthetic = ?", identity.CareClientID, true)
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()
	var rows []caseworkmodel.Consultation
	if err := query.Order("opened_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]caseworkres.ClientConsultationSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, summarizeClientConsultation(row))
	}
	return items, total, nil
}

func (s *CaseWorkService) GetClientConsultation(
	ctx context.Context,
	identity ClientConsultationIdentity,
	id uint,
) (caseworkres.ClientConsultationDetail, error) {
	if id == 0 || identity.CareClientID == 0 || identity.DeptID == 0 || !identity.Synthetic || !s.syntheticFixturesEnabled() {
		return caseworkres.ClientConsultationDetail{}, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeAccessScopeDenied,
			"咨询不存在或不在当前访问范围",
		)
	}
	var consultation caseworkmodel.Consultation
	err := s.db().WithContext(ctx).
		Where("id = ? AND care_client_id = ? AND synthetic = ?", id, identity.CareClientID, true).
		First(&consultation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return caseworkres.ClientConsultationDetail{}, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeAccessScopeDenied,
			"咨询不存在或不在当前访问范围",
		)
	}
	if err != nil {
		return caseworkres.ClientConsultationDetail{}, err
	}
	var interactions []caseworkmodel.ConsultationInteraction
	if err = s.db().WithContext(ctx).
		Where("consultation_id = ? AND client_visible = ?", consultation.ID, true).
		Order("occurred_at ASC, id ASC").Find(&interactions).Error; err != nil {
		return caseworkres.ClientConsultationDetail{}, err
	}
	detail := caseworkres.ClientConsultationDetail{
		ClientConsultationSummary: summarizeClientConsultation(consultation),
		InitialQuestion:           consultation.InitialQuestion,
		Resolution:                optionalString(consultation.Resolution),
		FollowUpPlan:              optionalString(consultation.FollowUpPlan),
		Interactions:              make([]caseworkres.ClientConsultationInteraction, 0, len(interactions)),
	}
	for _, interaction := range interactions {
		senderType := "SERVICE_TEAM"
		if interaction.ActorType == caseworkmodel.ConsultationActorClient {
			senderType = "CLIENT"
		} else if interaction.ActorType == caseworkmodel.ConsultationActorSystem {
			senderType = "SYSTEM"
		}
		detail.Interactions = append(detail.Interactions, caseworkres.ClientConsultationInteraction{
			ID:         interaction.ID,
			SenderType: senderType,
			Content:    interaction.Content,
			OccurredAt: interaction.OccurredAt,
		})
	}
	return detail, nil
}

func (s *CaseWorkService) AddClientConsultationMessage(
	ctx context.Context,
	identity ClientConsultationIdentity,
	id uint,
	key string,
	req caseworkreq.AddClientConsultationMessage,
) (caseworkres.ConsultationActionResult, error) {
	req.Message = trimmed(req.Message)
	if id == 0 || req.ExpectedVersion == 0 || req.Message == "" || identity.CareClientID == 0 ||
		identity.DeptID == 0 || !identity.Synthetic || !s.syntheticFixturesEnabled() {
		return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
			caseworkmodel.CodeInvalidArgument,
			"咨询标识、版本和补充内容必填",
		)
	}
	operation := fmt.Sprintf("client-consultation:message:%d", id)
	return runIdempotent(s, ctx, operation, identity.CareClientID, key, req,
		func(tx *gorm.DB, keyDigest string) (caseworkres.ConsultationActionResult, error) {
			var consultation caseworkmodel.Consultation
			err := locking(tx).Where(
				"id = ? AND care_client_id = ? AND synthetic = ?",
				id,
				identity.CareClientID,
				true,
			).First(&consultation).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewForbiddenError(
					caseworkmodel.CodeAccessScopeDenied,
					"咨询不存在或不在当前访问范围",
				)
			}
			if err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if consultation.Version != req.ExpectedVersion {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
					caseworkmodel.CodeVersionConflict,
					"咨询版本已变化",
				)
			}
			if consultation.Status == caseworkmodel.ConsultationStatusClosed {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
					caseworkmodel.CodeConsultationTransitionDenied,
					"已关闭咨询不能继续补充，请重新发起咨询",
				)
			}
			fromStatus := consultation.Status
			toStatus := fromStatus
			updates := map[string]any{"version": gorm.Expr("version + 1")}
			if fromStatus == caseworkmodel.ConsultationStatusWaitingClient || fromStatus == caseworkmodel.ConsultationStatusResolved {
				toStatus = caseworkmodel.ConsultationStatusHandling
				updates["status"] = toStatus
			}
			if fromStatus == caseworkmodel.ConsultationStatusResolved {
				updates["resolution"] = ""
				updates["follow_up_plan"] = ""
				updates["resolved_at"] = nil
			}
			update := tx.Model(&caseworkmodel.Consultation{}).
				Where("id = ? AND version = ?", consultation.ID, consultation.Version).
				Updates(updates)
			if update.Error != nil {
				return caseworkres.ConsultationActionResult{}, update.Error
			}
			if update.RowsAffected != 1 {
				return caseworkres.ConsultationActionResult{}, caseworkmodel.NewDomainError(
					caseworkmodel.CodeVersionConflict,
					"咨询版本已变化",
				)
			}
			consultation.Status = toStatus
			consultation.Version++
			now := s.now()
			interaction := caseworkmodel.ConsultationInteraction{
				ConsultationID:   consultation.ID,
				ActionType:       caseworkmodel.ConsultationActionClientMessage,
				ActorType:        caseworkmodel.ConsultationActorClient,
				ActorID:          identity.CareClientID,
				Content:          req.Message,
				FromStatus:       fromStatus,
				ToStatus:         toStatus,
				ClientVisible:    true,
				OccurredAt:       now,
				CommandKeyDigest: keyDigest,
				Synthetic:        true,
			}
			if err = tx.Create(&interaction).Error; err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			if err = appendConsultationEvent(tx, consultation, interaction, caseworkmodel.EventConsultationMessageAdded); err != nil {
				return caseworkres.ConsultationActionResult{}, err
			}
			return consultationActionResult(consultation, interaction), nil
		})
}

func summarizeClientConsultation(consultation caseworkmodel.Consultation) caseworkres.ClientConsultationSummary {
	return caseworkres.ClientConsultationSummary{
		ID:               consultation.ID,
		Subject:          consultation.Subject,
		Urgency:          consultation.Urgency,
		Status:           consultation.Status,
		OpenedAt:         consultation.OpenedAt,
		FirstRespondedAt: consultation.FirstRespondedAt,
		ResolvedAt:       consultation.ResolvedAt,
		ClosedAt:         consultation.ClosedAt,
		Version:          consultation.Version,
	}
}
