package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
)

func (s *SupervisionService) ListClientSatisfactionRequests(
	ctx context.Context,
	identity ClientSatisfactionIdentity,
	req supervisionreq.ClientSatisfactionSearch,
) ([]supervisionres.ClientSatisfactionSummary, int64, error) {
	if err := s.ReconcileClientSatisfactionRequests(ctx, identity); err != nil {
		return nil, 0, err
	}
	if req.Status != "" && !validSatisfactionRequestStatus(req.Status) {
		return nil, 0, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"评价邀请状态无效",
		)
	}
	if err := s.expireClientSatisfactionRequests(ctx, identity.CareClientID); err != nil {
		return nil, 0, err
	}
	query := s.db().WithContext(ctx).Model(&supervisionmodel.SatisfactionRequest{}).
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
	var rows []supervisionmodel.SatisfactionRequest
	if err := query.Order("invited_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]supervisionres.ClientSatisfactionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, clientSatisfactionSummary(row))
	}
	return items, total, nil
}

func (s *SupervisionService) GetClientSatisfactionRequest(
	ctx context.Context,
	identity ClientSatisfactionIdentity,
	id uint,
) (supervisionres.ClientSatisfactionDetail, error) {
	if id == 0 {
		return supervisionres.ClientSatisfactionDetail{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"评价邀请标识必填",
		)
	}
	if err := s.ReconcileClientSatisfactionRequests(ctx, identity); err != nil {
		return supervisionres.ClientSatisfactionDetail{}, err
	}
	if err := s.expireClientSatisfactionRequests(ctx, identity.CareClientID); err != nil {
		return supervisionres.ClientSatisfactionDetail{}, err
	}
	request, err := s.clientSatisfactionRequest(ctx, identity, id)
	if err != nil {
		return supervisionres.ClientSatisfactionDetail{}, err
	}
	detail := supervisionres.ClientSatisfactionDetail{
		ClientSatisfactionSummary: clientSatisfactionSummary(request),
	}
	var response supervisionmodel.SatisfactionResponse
	err = s.db().WithContext(ctx).Where("request_id = ?", request.ID).First(&response).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return detail, nil
	}
	if err != nil {
		return supervisionres.ClientSatisfactionDetail{}, err
	}
	detail.Response = &supervisionres.ClientSatisfactionResponse{
		Rating:      response.Rating,
		Comment:     response.Comment,
		SubmittedAt: response.SubmittedAt,
	}
	return detail, nil
}

func (s *SupervisionService) SubmitClientSatisfactionResponse(
	ctx context.Context,
	identity ClientSatisfactionIdentity,
	id uint,
	key string,
	req supervisionreq.SubmitSatisfactionResponse,
) (supervisionres.SubmitSatisfactionResult, error) {
	req.Comment = strings.TrimSpace(req.Comment)
	key = strings.TrimSpace(key)
	if id == 0 || req.ExpectedVersion == 0 || req.Rating < 1 || req.Rating > 5 || key == "" {
		return supervisionres.SubmitSatisfactionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeInvalidArgument,
			"评价邀请、版本、星级和 Idempotency-Key 必填",
		)
	}
	if err := s.ReconcileClientSatisfactionRequests(ctx, identity); err != nil {
		return supervisionres.SubmitSatisfactionResult{}, err
	}
	if err := s.expireClientSatisfactionRequests(ctx, identity.CareClientID); err != nil {
		return supervisionres.SubmitSatisfactionResult{}, err
	}
	keyDigest := digest(key)
	requestHash, err := hashRequest(req)
	if err != nil {
		return supervisionres.SubmitSatisfactionResult{}, err
	}
	var result supervisionres.SubmitSatisfactionResult
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var request supervisionmodel.SatisfactionRequest
		loadErr := summaryLocking(tx).Where(
			"id = ? AND care_client_id = ? AND synthetic = ?",
			id,
			identity.CareClientID,
			true,
		).First(&request).Error
		if errors.Is(loadErr, gorm.ErrRecordNotFound) {
			return supervisionmodel.NewForbiddenError(
				supervisionmodel.CodeSatisfactionScopeDenied,
				"评价邀请不存在或不在当前访问范围",
			)
		}
		if loadErr != nil {
			return loadErr
		}
		var existing supervisionmodel.SatisfactionResponse
		existingErr := tx.Where("request_id = ?", request.ID).First(&existing).Error
		if existingErr == nil {
			if existing.CommandKeyDigest == keyDigest && existing.RequestHash != requestHash {
				return supervisionmodel.NewDomainError(
					supervisionmodel.CodeIdempotencyConflict,
					"幂等键已用于不同评价内容",
				)
			}
			if existing.CommandKeyDigest != keyDigest {
				return supervisionmodel.NewDomainError(
					supervisionmodel.CodeSatisfactionAlreadySubmitted,
					"本次服务评价已经提交",
				)
			}
			result, loadErr = s.submittedSatisfactionResult(tx, request, existing)
			return loadErr
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		if request.Version != req.ExpectedVersion {
			return supervisionmodel.NewDomainError(supervisionmodel.CodeVersionConflict, "评价邀请版本已变化")
		}
		if request.Status == supervisionmodel.SatisfactionRequestExpired || !s.now().Before(request.ExpiresAt) {
			return supervisionmodel.NewDomainError(
				supervisionmodel.CodeSatisfactionTransitionDenied,
				"本次评价邀请已过期",
			)
		}
		if request.Status != supervisionmodel.SatisfactionRequestPending {
			return supervisionmodel.NewDomainError(
				supervisionmodel.CodeSatisfactionTransitionDenied,
				"当前评价邀请不能提交",
			)
		}
		now := s.now()
		response := supervisionmodel.SatisfactionResponse{
			RequestID:        request.ID,
			Rating:           req.Rating,
			Comment:          req.Comment,
			SubmittedAt:      now,
			CommandKeyDigest: keyDigest,
			RequestHash:      requestHash,
			Synthetic:        request.Synthetic,
		}
		if err := tx.Create(&response).Error; err != nil {
			return err
		}
		update := tx.Model(&supervisionmodel.SatisfactionRequest{}).
			Where("id = ? AND version = ?", request.ID, request.Version).
			Updates(map[string]any{
				"status":       supervisionmodel.SatisfactionRequestSubmitted,
				"submitted_at": now,
				"version":      gorm.Expr("version + 1"),
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return supervisionmodel.NewDomainError(supervisionmodel.CodeVersionConflict, "评价邀请版本已变化")
		}
		request.Status = supervisionmodel.SatisfactionRequestSubmitted
		request.SubmittedAt = &now
		request.Version++
		followUpCreated := false
		if response.Rating <= request.LowScoreThreshold {
			if _, err := s.createSatisfactionFollowUp(tx, request, response, now); err != nil {
				return err
			}
			followUpCreated = true
		}
		if err := appendSatisfactionEvent(tx, request, supervisionmodel.EventSatisfactionSubmitted, now, map[string]any{
			"requestId":       request.ID,
			"responseId":      response.ID,
			"rating":          response.Rating,
			"followUpCreated": followUpCreated,
		}); err != nil {
			return err
		}
		result = supervisionres.SubmitSatisfactionResult{
			RequestID:       request.ID,
			ResponseID:      response.ID,
			Status:          request.Status,
			Version:         request.Version,
			FollowUpCreated: followUpCreated,
			SubmittedAt:     response.SubmittedAt,
		}
		return nil
	})
	return result, err
}

func (s *SupervisionService) createSatisfactionFollowUp(
	tx *gorm.DB,
	request supervisionmodel.SatisfactionRequest,
	response supervisionmodel.SatisfactionResponse,
	now time.Time,
) (supervisionmodel.SatisfactionFollowUp, error) {
	var supervisor system.SysUser
	supervisorErr := tx.Set("data_scope:skip", true).
		Model(&system.SysUser{}).
		Joins("JOIN care_authority_profiles ON care_authority_profiles.authority_id = sys_users.authority_id AND care_authority_profiles.deleted_at IS NULL").
		Where(
			"sys_users.dept_id = ? AND sys_users.enable = ? AND care_authority_profiles.role_type = ? AND care_authority_profiles.active = ?",
			request.OrganizationID,
			1,
			caremodel.AuthorityRoleSupervisor,
			true,
		).Order("sys_users.id ASC").First(&supervisor).Error
	if supervisorErr != nil && !errors.Is(supervisorErr, gorm.ErrRecordNotFound) {
		return supervisionmodel.SatisfactionFollowUp{}, supervisorErr
	}
	var assigneeID *uint
	if supervisorErr == nil {
		value := supervisor.ID
		assigneeID = &value
	}
	followUp := supervisionmodel.SatisfactionFollowUp{
		RequestID:      request.ID,
		ResponseID:     response.ID,
		OrganizationID: request.OrganizationID,
		AssigneeID:     assigneeID,
		Status:         supervisionmodel.SatisfactionFollowUpOpen,
		OpenedAt:       now,
		Version:        1,
		Synthetic:      request.Synthetic,
	}
	if err := tx.Create(&followUp).Error; err != nil {
		return supervisionmodel.SatisfactionFollowUp{}, err
	}
	if assigneeID != nil {
		active := caseworkmodel.TodoActiveSlot
		todo := caseworkmodel.TodoItem{
			Category:     caseworkmodel.TodoCategorySatisfactionFollowUp,
			SourceType:   caseworkmodel.TodoSourceSatisfactionFollowUp,
			SourceID:     followUp.ID,
			ActiveSlot:   &active,
			CareClientID: request.CareClientID,
			AssigneeID:   *assigneeID,
			AssigneeRole: caremodel.AuthorityRoleSupervisor,
			Status:       caseworkmodel.TodoStatusOpen,
			OpenedAt:     now,
			Version:      1,
			Synthetic:    request.Synthetic,
		}
		if err := tx.Create(&todo).Error; err != nil {
			return supervisionmodel.SatisfactionFollowUp{}, err
		}
	}
	if err := appendFollowUpEvent(tx, followUp, supervisionmodel.EventSatisfactionFollowUpOpened, now, map[string]any{
		"requestId":  request.ID,
		"responseId": response.ID,
		"rating":     response.Rating,
	}); err != nil {
		return supervisionmodel.SatisfactionFollowUp{}, err
	}
	return followUp, nil
}

func (s *SupervisionService) submittedSatisfactionResult(
	db *gorm.DB,
	request supervisionmodel.SatisfactionRequest,
	response supervisionmodel.SatisfactionResponse,
) (supervisionres.SubmitSatisfactionResult, error) {
	var followUp supervisionmodel.SatisfactionFollowUp
	err := db.Where("request_id = ?", request.ID).First(&followUp).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return supervisionres.SubmitSatisfactionResult{}, err
	}
	return supervisionres.SubmitSatisfactionResult{
		RequestID:       request.ID,
		ResponseID:      response.ID,
		Status:          supervisionmodel.SatisfactionRequestSubmitted,
		Version:         request.Version,
		FollowUpCreated: err == nil,
		SubmittedAt:     response.SubmittedAt,
	}, nil
}

func (s *SupervisionService) expireClientSatisfactionRequests(ctx context.Context, careClientID uint) error {
	var requests []supervisionmodel.SatisfactionRequest
	if err := s.db().WithContext(ctx).Where(
		"care_client_id = ? AND status = ? AND expires_at <= ? AND synthetic = ?",
		careClientID,
		supervisionmodel.SatisfactionRequestPending,
		s.now(),
		true,
	).Find(&requests).Error; err != nil {
		return err
	}
	if len(requests) == 0 {
		return nil
	}
	return s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, request := range requests {
			update := tx.Model(&supervisionmodel.SatisfactionRequest{}).
				Where("id = ? AND version = ? AND status = ?", request.ID, request.Version, supervisionmodel.SatisfactionRequestPending).
				Updates(map[string]any{
					"status":  supervisionmodel.SatisfactionRequestExpired,
					"version": gorm.Expr("version + 1"),
				})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected == 0 {
				continue
			}
			request.Status = supervisionmodel.SatisfactionRequestExpired
			request.Version++
			if err := appendSatisfactionEvent(tx, request, supervisionmodel.EventSatisfactionExpired, s.now(), map[string]any{
				"requestId": request.ID,
				"status":    request.Status,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SupervisionService) clientSatisfactionRequest(
	ctx context.Context,
	identity ClientSatisfactionIdentity,
	id uint,
) (supervisionmodel.SatisfactionRequest, error) {
	var request supervisionmodel.SatisfactionRequest
	err := s.db().WithContext(ctx).Where(
		"id = ? AND care_client_id = ? AND synthetic = ?",
		id,
		identity.CareClientID,
		true,
	).First(&request).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return request, supervisionmodel.NewForbiddenError(
			supervisionmodel.CodeSatisfactionScopeDenied,
			"评价邀请不存在或不在当前访问范围",
		)
	}
	return request, err
}

func clientSatisfactionSummary(request supervisionmodel.SatisfactionRequest) supervisionres.ClientSatisfactionSummary {
	return supervisionres.ClientSatisfactionSummary{
		ID:            request.ID,
		PublicCode:    satisfactionPublicCode(request.ID),
		SourceType:    request.SourceType,
		Status:        request.Status,
		AnonymityMode: request.AnonymityMode,
		InvitedAt:     request.InvitedAt,
		ExpiresAt:     request.ExpiresAt,
		SubmittedAt:   request.SubmittedAt,
		Version:       request.Version,
	}
}

func satisfactionPublicCode(id uint) string {
	return fmt.Sprintf("EV-%06d", id)
}

func validSatisfactionRequestStatus(value string) bool {
	switch value {
	case supervisionmodel.SatisfactionRequestPending,
		supervisionmodel.SatisfactionRequestSubmitted,
		supervisionmodel.SatisfactionRequestExpired:
		return true
	default:
		return false
	}
}

func appendSatisfactionEvent(
	db *gorm.DB,
	request supervisionmodel.SatisfactionRequest,
	eventType string,
	occurredAt time.Time,
	payload map[string]any,
) error {
	payload["synthetic"] = request.Synthetic
	return platformoutbox.Append(db, platformoutbox.AppendInput{
		EventType:     eventType,
		AggregateType: "SatisfactionRequest",
		AggregateID:   request.ID,
		Payload:       payload,
		OccurredAt:    occurredAt,
		CausationID:   fmt.Sprintf("satisfaction-request:%d:v%d", request.ID, request.Version),
		Synthetic:     request.Synthetic,
	})
}

func appendFollowUpEvent(
	db *gorm.DB,
	followUp supervisionmodel.SatisfactionFollowUp,
	eventType string,
	occurredAt time.Time,
	payload map[string]any,
) error {
	payload["followUpId"] = followUp.ID
	payload["status"] = followUp.Status
	payload["synthetic"] = followUp.Synthetic
	return platformoutbox.Append(db, platformoutbox.AppendInput{
		EventType:     eventType,
		AggregateType: "SatisfactionFollowUp",
		AggregateID:   followUp.ID,
		Payload:       payload,
		OccurredAt:    occurredAt,
		CausationID:   fmt.Sprintf("satisfaction-follow-up:%d:v%d", followUp.ID, followUp.Version),
		Synthetic:     followUp.Synthetic,
	})
}
