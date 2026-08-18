package supervision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	caseworkservice "github.com/flipped-aurora/gin-vue-admin/server/service/casework"
	"gorm.io/gorm"
)

func (s *SupervisionService) AddGuidance(ctx context.Context, id uint, key string, req supervisionreq.Guidance) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 || strings.TrimSpace(req.Guidance) == "" ||
		req.ResponsibleAssigneeID == 0 || req.DueAt.IsZero() || !req.DueAt.After(s.now()) {
		return caseworkres.ActionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeGuidanceResultRequired, "指导内容、责任医护、未来截止时间和 expectedVersion 必填",
		)
	}
	return s.runSupervisorAction(
		ctx, id, key, fmt.Sprintf("GUIDANCE_REVIEW:%d", id), supervisionmodel.GuidanceActionGuidance,
		req.ExpectedVersion, req.Guidance, req.ResponsibleAssigneeID, req.DueAt, req,
	)
}

func (s *SupervisionService) Intervene(ctx context.Context, id uint, key string, req supervisionreq.Intervene) (caseworkres.ActionResult, error) {
	if id == 0 || req.ExpectedVersion == 0 || strings.TrimSpace(req.Result) == "" ||
		req.ResponsibleAssigneeID == 0 || req.DueAt.IsZero() || !req.DueAt.After(s.now()) {
		return caseworkres.ActionResult{}, supervisionmodel.NewDomainError(
			supervisionmodel.CodeGuidanceResultRequired, "介入结果、责任医护、未来截止时间和 expectedVersion 必填",
		)
	}
	return s.runSupervisorAction(
		ctx, id, key, fmt.Sprintf("INTERVENE_REVIEW:%d", id), supervisionmodel.GuidanceActionIntervene,
		req.ExpectedVersion, req.Result, req.ResponsibleAssigneeID, req.DueAt, req,
	)
}

func (s *SupervisionService) runSupervisorAction(
	ctx context.Context,
	id uint,
	key string,
	operation string,
	actionType string,
	expectedVersion uint,
	resultText string,
	responsibleAssigneeID uint,
	dueAt time.Time,
	request any,
) (caseworkres.ActionResult, error) {
	decision, _, err := s.supervisorScope(ctx)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return caseworkres.ActionResult{}, supervisionmodel.NewDomainError(supervisionmodel.CodeInvalidArgument, "Idempotency-Key 必填")
	}
	keyDigest := digest(key)
	requestHash, err := hashRequest(request)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}

	var response caseworkres.ActionResult
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing supervisionmodel.SupervisorGuidance
		findErr := tx.Where("actor_id = ? AND operation = ? AND command_key_digest = ?", decision.Identity.UserID, operation, keyDigest).
			First(&existing).Error
		if findErr == nil {
			if existing.RequestHash != requestHash {
				return supervisionmodel.NewDomainError(supervisionmodel.CodeIdempotencyConflict, "幂等键已用于不同请求")
			}
			response = guidanceResult(existing)
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		worker := &caseworkservice.CaseWorkService{DB: tx, Now: s.Now}
		caseResult, actionErr := worker.ApplySupervisorAction(ctx, caseworkservice.SupervisorActionCommand{
			AttentionCaseID:       id,
			ExpectedVersion:       expectedVersion,
			ActionType:            actionType,
			Result:                resultText,
			ResponsibleAssigneeID: responsibleAssigneeID,
			DueAt:                 dueAt,
			CommandKeyDigest:      keyDigest,
		})
		if actionErr != nil {
			return normalizeCaseWorkError(actionErr)
		}
		var attentionCase caseworkmodel.AttentionCase
		if loadErr := tx.Where("id = ?", id).First(&attentionCase).Error; loadErr != nil {
			return loadErr
		}
		guidance := supervisionmodel.SupervisorGuidance{
			AttentionCaseID:       id,
			CaseActionID:          caseResult.ActionID,
			ActionType:            actionType,
			Guidance:              strings.TrimSpace(resultText),
			ActorID:               decision.Identity.UserID,
			ResponsibleAssigneeID: responsibleAssigneeID,
			DueAt:                 dueAt,
			CaseVersionBefore:     expectedVersion,
			CaseVersionAfter:      caseResult.Version,
			OccurredAt:            caseResult.OccurredAt,
			Operation:             operation,
			CommandKeyDigest:      keyDigest,
			RequestHash:           requestHash,
			Synthetic:             attentionCase.Synthetic,
		}
		if createErr := tx.Create(&guidance).Error; createErr != nil {
			if duplicateError(createErr) {
				return supervisionmodel.NewDomainError(supervisionmodel.CodeIdempotencyConflict, "幂等请求发生并发冲突，请重试")
			}
			return createErr
		}
		response = caseResult
		return nil
	})
	return response, err
}

func guidanceResult(guidance supervisionmodel.SupervisorGuidance) caseworkres.ActionResult {
	status := caseworkmodel.CaseStatusWaitingSupervisor
	if guidance.ActionType == supervisionmodel.GuidanceActionIntervene {
		status = caseworkmodel.CaseStatusHandling
	}
	return caseworkres.ActionResult{
		ResourceID: guidance.AttentionCaseID,
		ActionID:   guidance.CaseActionID,
		Status:     status,
		Version:    guidance.CaseVersionAfter,
		OccurredAt: guidance.OccurredAt,
	}
}
