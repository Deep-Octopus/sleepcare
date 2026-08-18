package supervision

import (
	"context"
	"errors"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"gorm.io/gorm"
)

func (s *SupervisionService) ListReviews(ctx context.Context, req supervisionreq.ReviewSearch) ([]supervisionres.ReviewItem, int64, error) {
	decision, _, err := s.supervisorScope(ctx)
	if err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	limit, offset := req.LimitOffset()

	query := decision.ScopeAttentionCases(
		s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}), s.now(),
	).Where("attention_cases.synthetic = ?", true).
		Where(`EXISTS (
			SELECT 1 FROM case_actions request_actions
			WHERE request_actions.attention_case_id = attention_cases.id
			  AND request_actions.action_type = ?
			  AND request_actions.actor_role = ?
			  AND request_actions.to_status = ?
			  AND request_actions.deleted_at IS NULL
		)`, caseworkmodel.CaseActionHandling, caremodel.AuthorityRoleClinician, caseworkmodel.CaseStatusWaitingSupervisor)
	var total int64
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var cases []caseworkmodel.AttentionCase
	if err = query.Order("attention_cases.opened_at DESC, attention_cases.id DESC").
		Limit(limit).Offset(offset).Find(&cases).Error; err != nil {
		return nil, 0, err
	}

	items := make([]supervisionres.ReviewItem, 0, len(cases))
	for i := range cases {
		var requestAction caseworkmodel.CaseAction
		err = s.db().WithContext(ctx).Set("data_scope:skip", true).
			Where(
				"attention_case_id = ? AND action_type = ? AND actor_role = ? AND to_status = ?",
				cases[i].ID, caseworkmodel.CaseActionHandling, caremodel.AuthorityRoleClinician, caseworkmodel.CaseStatusWaitingSupervisor,
			).
			Order("occurred_at DESC, id DESC").First(&requestAction).Error
		if err != nil {
			return nil, 0, err
		}
		status, statusErr := s.reviewStatus(ctx, cases[i], requestAction)
		if statusErr != nil {
			return nil, 0, statusErr
		}
		items = append(items, supervisionres.ReviewItem{
			ID:              cases[i].ID,
			AttentionCaseID: cases[i].ID,
			Status:          status,
			RequestedAt:     requestAction.OccurredAt,
			RequestedBy:     requestAction.ActorID,
		})
	}
	return items, total, nil
}

func (s *SupervisionService) reviewStatus(
	ctx context.Context,
	attentionCase caseworkmodel.AttentionCase,
	requestAction caseworkmodel.CaseAction,
) (string, error) {
	var latestGuidance supervisionmodel.SupervisorGuidance
	err := s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("attention_case_id = ?", attentionCase.ID).
		Order("occurred_at DESC, id DESC").First(&latestGuidance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if attentionCase.Status == caseworkmodel.CaseStatusWaitingSupervisor && requestAction.ID > 0 {
			return supervisionmodel.ReviewStatusPending, nil
		}
		return supervisionmodel.ReviewStatusCompleted, nil
	}
	if err != nil {
		return "", err
	}
	if requestAction.ID > latestGuidance.CaseActionID {
		return supervisionmodel.ReviewStatusPending, nil
	}
	var latestAction caseworkmodel.CaseAction
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("attention_case_id = ?", attentionCase.ID).
		Order("occurred_at DESC, id DESC").First(&latestAction).Error; err != nil {
		return "", err
	}
	if latestAction.ID == requestAction.ID {
		return supervisionmodel.ReviewStatusPending, nil
	}
	if latestAction.ID != latestGuidance.CaseActionID {
		return supervisionmodel.ReviewStatusCompleted, nil
	}
	if latestGuidance.ActionType == supervisionmodel.GuidanceActionIntervene {
		return supervisionmodel.ReviewStatusIntervened, nil
	}
	return supervisionmodel.ReviewStatusGuided, nil
}
