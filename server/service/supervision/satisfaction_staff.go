package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
)

func (s *SupervisionService) ListSatisfactionResponses(
	ctx context.Context,
	req supervisionreq.SatisfactionResponseSearch,
) ([]supervisionres.SatisfactionResponseItem, int64, error) {
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return nil, 0, err
	}
	if req.Rating > 5 || (req.FollowUpStatus != "" && req.FollowUpStatus != "NONE" && !validSatisfactionFollowUpStatus(req.FollowUpStatus)) {
		return nil, 0, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"评价筛选条件无效",
		)
	}
	if err = s.reconcileOrganizationSatisfactionRequests(ctx, organizationID); err != nil {
		return nil, 0, err
	}
	query := s.db().WithContext(ctx).Model(&supervisionmodel.SatisfactionResponse{}).
		Joins("JOIN satisfaction_requests ON satisfaction_requests.id = satisfaction_responses.request_id AND satisfaction_requests.deleted_at IS NULL").
		Joins("LEFT JOIN satisfaction_follow_ups ON satisfaction_follow_ups.request_id = satisfaction_requests.id AND satisfaction_follow_ups.deleted_at IS NULL").
		Where(
			"satisfaction_requests.organization_id = ? AND satisfaction_responses.synthetic = ?",
			organizationID,
			true,
		)
	if req.Rating != 0 {
		query = query.Where("satisfaction_responses.rating = ?", req.Rating)
	}
	if req.FollowUpStatus == "NONE" {
		query = query.Where("satisfaction_follow_ups.id IS NULL")
	} else if req.FollowUpStatus != "" {
		query = query.Where("satisfaction_follow_ups.status = ?", req.FollowUpStatus)
	}
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()
	var rows []supervisionmodel.SatisfactionResponse
	if err = query.Order("satisfaction_responses.submitted_at DESC, satisfaction_responses.id DESC").
		Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := s.satisfactionResponseItems(ctx, rows)
	return items, total, err
}

func (s *SupervisionService) ListSatisfactionFollowUps(
	ctx context.Context,
	req supervisionreq.SatisfactionFollowUpSearch,
) ([]supervisionres.SatisfactionFollowUpSummary, int64, error) {
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return nil, 0, err
	}
	if req.Status != "" && !validSatisfactionFollowUpStatus(req.Status) {
		return nil, 0, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"质量跟进状态无效",
		)
	}
	query := s.db().WithContext(ctx).Model(&supervisionmodel.SatisfactionFollowUp{}).
		Where("organization_id = ? AND synthetic = ?", organizationID, true)
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()
	var rows []supervisionmodel.SatisfactionFollowUp
	if err = query.Order("opened_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := s.satisfactionFollowUpSummaries(ctx, rows)
	return items, total, err
}

func (s *SupervisionService) GetSatisfactionFollowUp(
	ctx context.Context,
	id uint,
) (supervisionres.SatisfactionFollowUpDetail, error) {
	_, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return supervisionres.SatisfactionFollowUpDetail{}, err
	}
	followUp, err := s.loadSatisfactionFollowUp(ctx, id, organizationID)
	if err != nil {
		return supervisionres.SatisfactionFollowUpDetail{}, err
	}
	return s.satisfactionFollowUpDetail(ctx, followUp)
}

func (s *SupervisionService) AcknowledgeSatisfactionFollowUp(
	ctx context.Context,
	id uint,
	key string,
	req supervisionreq.AcknowledgeSatisfactionFollowUp,
) (supervisionres.SatisfactionFollowUpActionResult, error) {
	req.Note = strings.TrimSpace(req.Note)
	if id == 0 || req.ExpectedVersion == 0 || req.Note == "" {
		return supervisionres.SatisfactionFollowUpActionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"质量跟进、版本和接收说明必填",
		)
	}
	decision, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return supervisionres.SatisfactionFollowUpActionResult{}, err
	}
	operation := fmt.Sprintf("satisfaction-followup:acknowledge:%d", id)
	return s.runSatisfactionFollowUpCommand(ctx, id, organizationID, decision.Identity.UserID, decision.RoleType, operation, key, req,
		func(tx *gorm.DB, followUp *supervisionmodel.SatisfactionFollowUp, keyDigest string, requestHash string) (supervisionmodel.SatisfactionFollowUpAction, error) {
			if followUp.Status != supervisionmodel.SatisfactionFollowUpOpen {
				return supervisionmodel.SatisfactionFollowUpAction{}, satisfactionTransitionDenied("只有待接收跟进可以接收")
			}
			if followUp.AssigneeID != nil && *followUp.AssigneeID != decision.Identity.UserID {
				return supervisionmodel.SatisfactionFollowUpAction{}, satisfactionScopeDenied("当前质量跟进已分配给其他上级")
			}
			now := s.now()
			fromStatus := followUp.Status
			update := tx.Model(&supervisionmodel.SatisfactionFollowUp{}).
				Where("id = ? AND version = ?", followUp.ID, followUp.Version).
				Updates(map[string]any{
					"assignee_id":     decision.Identity.UserID,
					"status":          supervisionmodel.SatisfactionFollowUpInReview,
					"acknowledged_at": now,
					"version":         gorm.Expr("version + 1"),
				})
			if update.Error != nil {
				return supervisionmodel.SatisfactionFollowUpAction{}, update.Error
			}
			if update.RowsAffected != 1 {
				return supervisionmodel.SatisfactionFollowUpAction{}, supervisionmodel.NewDomainError(
					supervisionmodel.CodeVersionConflict,
					"质量跟进版本已变化",
				)
			}
			actorID := decision.Identity.UserID
			followUp.AssigneeID = &actorID
			followUp.Status = supervisionmodel.SatisfactionFollowUpInReview
			followUp.AcknowledgedAt = &now
			versionBefore := followUp.Version
			followUp.Version++
			return supervisionmodel.SatisfactionFollowUpAction{
				FollowUpID:       followUp.ID,
				ActionType:       supervisionmodel.SatisfactionFollowUpActionAcknowledge,
				ActorID:          actorID,
				ActorRole:        decision.RoleType,
				Content:          req.Note,
				FromStatus:       fromStatus,
				ToStatus:         followUp.Status,
				VersionBefore:    versionBefore,
				VersionAfter:     followUp.Version,
				OccurredAt:       now,
				Operation:        operation,
				CommandKeyDigest: keyDigest,
				RequestHash:      requestHash,
				Synthetic:        followUp.Synthetic,
			}, nil
		})
}

func (s *SupervisionService) ResolveSatisfactionFollowUp(
	ctx context.Context,
	id uint,
	key string,
	req supervisionreq.ResolveSatisfactionFollowUp,
) (supervisionres.SatisfactionFollowUpActionResult, error) {
	req.Resolution = strings.TrimSpace(req.Resolution)
	req.ImprovementAction = strings.TrimSpace(req.ImprovementAction)
	if id == 0 || req.ExpectedVersion == 0 || req.Resolution == "" {
		return supervisionres.SatisfactionFollowUpActionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"质量跟进、版本和核查结果必填",
		)
	}
	if !req.UsageBoundaryConfirmed {
		return supervisionres.SatisfactionFollowUpActionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeSatisfactionUsageBoundaryRequired,
			"必须确认单条评价不能直接形成对人员的结论",
		)
	}
	decision, organizationID, err := s.supervisorScope(ctx)
	if err != nil {
		return supervisionres.SatisfactionFollowUpActionResult{}, err
	}
	operation := fmt.Sprintf("satisfaction-followup:resolve:%d", id)
	return s.runSatisfactionFollowUpCommand(ctx, id, organizationID, decision.Identity.UserID, decision.RoleType, operation, key, req,
		func(tx *gorm.DB, followUp *supervisionmodel.SatisfactionFollowUp, keyDigest string, requestHash string) (supervisionmodel.SatisfactionFollowUpAction, error) {
			if followUp.Status != supervisionmodel.SatisfactionFollowUpInReview {
				return supervisionmodel.SatisfactionFollowUpAction{}, satisfactionTransitionDenied("接收后才能解决质量跟进")
			}
			if followUp.AssigneeID == nil || *followUp.AssigneeID != decision.Identity.UserID {
				return supervisionmodel.SatisfactionFollowUpAction{}, satisfactionScopeDenied("只有当前跟进责任上级可以解决")
			}
			now := s.now()
			fromStatus := followUp.Status
			update := tx.Model(&supervisionmodel.SatisfactionFollowUp{}).
				Where("id = ? AND version = ?", followUp.ID, followUp.Version).
				Updates(map[string]any{
					"status":             supervisionmodel.SatisfactionFollowUpResolved,
					"resolution":         req.Resolution,
					"improvement_action": req.ImprovementAction,
					"resolved_at":        now,
					"version":            gorm.Expr("version + 1"),
				})
			if update.Error != nil {
				return supervisionmodel.SatisfactionFollowUpAction{}, update.Error
			}
			if update.RowsAffected != 1 {
				return supervisionmodel.SatisfactionFollowUpAction{}, supervisionmodel.NewDomainError(
					supervisionmodel.CodeVersionConflict,
					"质量跟进版本已变化",
				)
			}
			if err := tx.Model(&caseworkmodel.TodoItem{}).
				Where(
					"source_type = ? AND source_id = ? AND active_slot = ?",
					caseworkmodel.TodoSourceSatisfactionFollowUp,
					followUp.ID,
					caseworkmodel.TodoActiveSlot,
				).Updates(map[string]any{
				"status":          caseworkmodel.TodoStatusCompleted,
				"active_slot":     nil,
				"completed_at":    now,
				"completion_note": req.Resolution,
				"version":         gorm.Expr("version + 1"),
			}).Error; err != nil {
				return supervisionmodel.SatisfactionFollowUpAction{}, err
			}
			versionBefore := followUp.Version
			followUp.Status = supervisionmodel.SatisfactionFollowUpResolved
			followUp.Resolution = req.Resolution
			followUp.ImprovementAction = req.ImprovementAction
			followUp.ResolvedAt = &now
			followUp.Version++
			return supervisionmodel.SatisfactionFollowUpAction{
				FollowUpID:             followUp.ID,
				ActionType:             supervisionmodel.SatisfactionFollowUpActionResolve,
				ActorID:                decision.Identity.UserID,
				ActorRole:              decision.RoleType,
				Content:                req.Resolution,
				ImprovementAction:      req.ImprovementAction,
				UsageBoundaryConfirmed: true,
				FromStatus:             fromStatus,
				ToStatus:               followUp.Status,
				VersionBefore:          versionBefore,
				VersionAfter:           followUp.Version,
				OccurredAt:             now,
				Operation:              operation,
				CommandKeyDigest:       keyDigest,
				RequestHash:            requestHash,
				Synthetic:              followUp.Synthetic,
			}, nil
		})
}

func (s *SupervisionService) runSatisfactionFollowUpCommand(
	ctx context.Context,
	id uint,
	organizationID uint,
	actorID uint,
	actorRole string,
	operation string,
	key string,
	request any,
	apply func(*gorm.DB, *supervisionmodel.SatisfactionFollowUp, string, string) (supervisionmodel.SatisfactionFollowUpAction, error),
) (supervisionres.SatisfactionFollowUpActionResult, error) {
	key = strings.TrimSpace(key)
	if key == "" || actorID == 0 {
		return supervisionres.SatisfactionFollowUpActionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"Idempotency-Key 和操作者必填",
		)
	}
	keyDigest := digest(key)
	requestHash, err := hashRequest(request)
	if err != nil {
		return supervisionres.SatisfactionFollowUpActionResult{}, err
	}
	var result supervisionres.SatisfactionFollowUpActionResult
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing supervisionmodel.SatisfactionFollowUpAction
		existingErr := tx.Where(
			"actor_id = ? AND operation = ? AND command_key_digest = ?",
			actorID,
			operation,
			keyDigest,
		).First(&existing).Error
		if existingErr == nil {
			if existing.RequestHash != requestHash {
				return supervisionmodel.NewDomainError(
					supervisionmodel.CodeIdempotencyConflict,
					"幂等键已用于不同质量跟进请求",
				)
			}
			result = satisfactionFollowUpActionResult(existing)
			return nil
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		var followUp supervisionmodel.SatisfactionFollowUp
		loadErr := summaryLocking(tx).Where(
			"id = ? AND organization_id = ? AND synthetic = ?",
			id,
			organizationID,
			true,
		).First(&followUp).Error
		if errors.Is(loadErr, gorm.ErrRecordNotFound) {
			return satisfactionScopeDenied("质量跟进不存在或不在当前管理范围")
		}
		if loadErr != nil {
			return loadErr
		}
		expectedVersion := satisfactionExpectedVersion(request)
		if followUp.Version != expectedVersion {
			return supervisionmodel.NewDomainError(supervisionmodel.CodeVersionConflict, "质量跟进版本已变化")
		}
		action, applyErr := apply(tx, &followUp, keyDigest, requestHash)
		if applyErr != nil {
			return applyErr
		}
		action.ActorRole = actorRole
		if err := tx.Create(&action).Error; err != nil {
			if duplicateError(err) {
				return supervisionmodel.NewDomainError(
					supervisionmodel.CodeIdempotencyConflict,
					"质量跟进命令发生并发冲突，请重试",
				)
			}
			return err
		}
		eventType := supervisionmodel.EventSatisfactionFollowUpAcknowledged
		if action.ActionType == supervisionmodel.SatisfactionFollowUpActionResolve {
			eventType = supervisionmodel.EventSatisfactionFollowUpResolved
		}
		if err := appendFollowUpEvent(tx, followUp, eventType, action.OccurredAt, map[string]any{
			"actionId":               action.ID,
			"actionType":             action.ActionType,
			"actorId":                action.ActorID,
			"usageBoundaryConfirmed": action.UsageBoundaryConfirmed,
		}); err != nil {
			return err
		}
		result = satisfactionFollowUpActionResult(action)
		return nil
	})
	return result, err
}

func (s *SupervisionService) satisfactionResponseItems(
	ctx context.Context,
	rows []supervisionmodel.SatisfactionResponse,
) ([]supervisionres.SatisfactionResponseItem, error) {
	requestIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		requestIDs = append(requestIDs, row.RequestID)
	}
	requests := make(map[uint]supervisionmodel.SatisfactionRequest)
	followUps := make(map[uint]supervisionmodel.SatisfactionFollowUp)
	if len(requestIDs) > 0 {
		var requestRows []supervisionmodel.SatisfactionRequest
		if err := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("id IN ?", requestIDs).Find(&requestRows).Error; err != nil {
			return nil, err
		}
		for _, request := range requestRows {
			requests[request.ID] = request
		}
		var followUpRows []supervisionmodel.SatisfactionFollowUp
		if err := s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where("request_id IN ?", requestIDs).Find(&followUpRows).Error; err != nil {
			return nil, err
		}
		for _, followUp := range followUpRows {
			followUps[followUp.RequestID] = followUp
		}
	}
	items := make([]supervisionres.SatisfactionResponseItem, 0, len(rows))
	for _, row := range rows {
		request := requests[row.RequestID]
		item := supervisionres.SatisfactionResponseItem{
			ID:            row.ID,
			PublicCode:    satisfactionPublicCode(request.ID),
			Rating:        row.Rating,
			Comment:       row.Comment,
			AnonymityMode: request.AnonymityMode,
			SubmittedAt:   row.SubmittedAt,
		}
		if followUp, ok := followUps[row.RequestID]; ok {
			item.FollowUpID = &followUp.ID
			item.FollowUpStatus = followUp.Status
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SupervisionService) satisfactionFollowUpSummaries(
	ctx context.Context,
	rows []supervisionmodel.SatisfactionFollowUp,
) ([]supervisionres.SatisfactionFollowUpSummary, error) {
	requestIDs := make([]uint, 0, len(rows))
	responseIDs := make([]uint, 0, len(rows))
	assigneeIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		requestIDs = append(requestIDs, row.RequestID)
		responseIDs = append(responseIDs, row.ResponseID)
		if row.AssigneeID != nil {
			assigneeIDs = append(assigneeIDs, *row.AssigneeID)
		}
	}
	requests := make(map[uint]supervisionmodel.SatisfactionRequest)
	responses := make(map[uint]supervisionmodel.SatisfactionResponse)
	users := make(map[uint]system.SysUser)
	db := func() *gorm.DB {
		return s.db().Session(&gorm.Session{NewDB: true}).
			WithContext(ctx).
			Set("data_scope:skip", true)
	}
	if len(requestIDs) > 0 {
		var values []supervisionmodel.SatisfactionRequest
		if err := db().Where("id IN ?", requestIDs).Find(&values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			requests[value.ID] = value
		}
	}
	if len(responseIDs) > 0 {
		var values []supervisionmodel.SatisfactionResponse
		if err := db().Where("id IN ?", responseIDs).Find(&values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			responses[value.ID] = value
		}
	}
	if len(assigneeIDs) > 0 {
		var values []system.SysUser
		if err := db().Where("id IN ?", assigneeIDs).Find(&values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			users[value.ID] = value
		}
	}
	items := make([]supervisionres.SatisfactionFollowUpSummary, 0, len(rows))
	for _, row := range rows {
		request := requests[row.RequestID]
		response := responses[row.ResponseID]
		assigneeName := ""
		if row.AssigneeID != nil {
			assigneeName = users[*row.AssigneeID].NickName
		}
		items = append(items, supervisionres.SatisfactionFollowUpSummary{
			ID:             row.ID,
			PublicCode:     satisfactionPublicCode(request.ID),
			Rating:         response.Rating,
			Status:         row.Status,
			AssigneeID:     row.AssigneeID,
			AssigneeName:   assigneeName,
			OpenedAt:       row.OpenedAt,
			AcknowledgedAt: row.AcknowledgedAt,
			ResolvedAt:     row.ResolvedAt,
			Version:        row.Version,
		})
	}
	return items, nil
}

func (s *SupervisionService) satisfactionFollowUpDetail(
	ctx context.Context,
	followUp supervisionmodel.SatisfactionFollowUp,
) (supervisionres.SatisfactionFollowUpDetail, error) {
	summaries, err := s.satisfactionFollowUpSummaries(ctx, []supervisionmodel.SatisfactionFollowUp{followUp})
	if err != nil {
		return supervisionres.SatisfactionFollowUpDetail{}, err
	}
	db := func() *gorm.DB {
		return s.db().Session(&gorm.Session{NewDB: true}).
			WithContext(ctx).
			Set("data_scope:skip", true)
	}
	var response supervisionmodel.SatisfactionResponse
	if err = db().Where("id = ?", followUp.ResponseID).First(&response).Error; err != nil {
		return supervisionres.SatisfactionFollowUpDetail{}, err
	}
	var request supervisionmodel.SatisfactionRequest
	if err = db().Where("id = ?", followUp.RequestID).First(&request).Error; err != nil {
		return supervisionres.SatisfactionFollowUpDetail{}, err
	}
	var actions []supervisionmodel.SatisfactionFollowUpAction
	if err = db().Where("follow_up_id = ?", followUp.ID).
		Order("occurred_at ASC, id ASC").Find(&actions).Error; err != nil {
		return supervisionres.SatisfactionFollowUpDetail{}, err
	}
	actorIDs := make([]uint, 0, len(actions))
	for _, action := range actions {
		actorIDs = append(actorIDs, action.ActorID)
	}
	actorNames := make(map[uint]string)
	if len(actorIDs) > 0 {
		var users []system.SysUser
		if err = db().Where("id IN ?", actorIDs).Find(&users).Error; err != nil {
			return supervisionres.SatisfactionFollowUpDetail{}, err
		}
		for _, user := range users {
			actorNames[user.ID] = user.NickName
		}
	}
	detail := supervisionres.SatisfactionFollowUpDetail{
		SatisfactionFollowUpSummary: summaries[0],
		Comment:                     response.Comment,
		AnonymityMode:               request.AnonymityMode,
		SubmittedAt:                 response.SubmittedAt,
		Resolution:                  followUp.Resolution,
		ImprovementAction:           followUp.ImprovementAction,
		Actions:                     make([]supervisionres.SatisfactionFollowUpAction, 0, len(actions)),
	}
	for _, action := range actions {
		detail.Actions = append(detail.Actions, supervisionres.SatisfactionFollowUpAction{
			ID:                     action.ID,
			ActionType:             action.ActionType,
			ActorName:              actorNames[action.ActorID],
			Content:                action.Content,
			ImprovementAction:      action.ImprovementAction,
			UsageBoundaryConfirmed: action.UsageBoundaryConfirmed,
			FromStatus:             action.FromStatus,
			ToStatus:               action.ToStatus,
			OccurredAt:             action.OccurredAt,
		})
	}
	return detail, nil
}

func (s *SupervisionService) loadSatisfactionFollowUp(
	ctx context.Context,
	id uint,
	organizationID uint,
) (supervisionmodel.SatisfactionFollowUp, error) {
	if id == 0 {
		return supervisionmodel.SatisfactionFollowUp{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"质量跟进标识必填",
		)
	}
	var followUp supervisionmodel.SatisfactionFollowUp
	err := s.db().WithContext(ctx).Where(
		"id = ? AND organization_id = ? AND synthetic = ?",
		id,
		organizationID,
		true,
	).First(&followUp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return followUp, satisfactionScopeDenied("质量跟进不存在或不在当前管理范围")
	}
	return followUp, err
}

func satisfactionExpectedVersion(request any) uint {
	switch value := request.(type) {
	case supervisionreq.AcknowledgeSatisfactionFollowUp:
		return value.ExpectedVersion
	case supervisionreq.ResolveSatisfactionFollowUp:
		return value.ExpectedVersion
	default:
		return 0
	}
}

func satisfactionFollowUpActionResult(
	action supervisionmodel.SatisfactionFollowUpAction,
) supervisionres.SatisfactionFollowUpActionResult {
	return supervisionres.SatisfactionFollowUpActionResult{
		FollowUpID: action.FollowUpID,
		ActionID:   action.ID,
		Status:     action.ToStatus,
		Version:    action.VersionAfter,
		OccurredAt: action.OccurredAt,
	}
}

func validSatisfactionFollowUpStatus(value string) bool {
	switch value {
	case supervisionmodel.SatisfactionFollowUpOpen,
		supervisionmodel.SatisfactionFollowUpInReview,
		supervisionmodel.SatisfactionFollowUpResolved:
		return true
	default:
		return false
	}
}

func satisfactionScopeDenied(message string) error {
	return supervisionmodel.NewForbiddenError(supervisionmodel.CodeSatisfactionScopeDenied, message)
}

func satisfactionTransitionDenied(message string) error {
	return supervisionmodel.NewDomainError(supervisionmodel.CodeSatisfactionTransitionDenied, message)
}
