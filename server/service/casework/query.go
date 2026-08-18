package casework

import (
	"context"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"gorm.io/gorm"
)

func (s *CaseWorkService) List(ctx context.Context, req caseworkreq.AttentionCaseSearch) ([]caseworkres.AttentionCaseSummary, int64, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, 0, normalizeAccessError(err)
	}
	if req.Status != "" && !validCaseStatus(req.Status) {
		return nil, 0, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "关注事项状态无效")
	}
	query := decision.ScopeAttentionCases(s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}), s.now())
	if req.Status != "" {
		query = query.Where("attention_cases.status = ?", req.Status)
	}
	if req.AssigneeID != 0 {
		query = query.Where("attention_cases.assignee_id = ?", req.AssigneeID)
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
	var cases []caseworkmodel.AttentionCase
	if err = query.Order("attention_cases.opened_at DESC, attention_cases.id DESC").Limit(limit).Offset(offset).Find(&cases).Error; err != nil {
		return nil, 0, err
	}
	items := make([]caseworkres.AttentionCaseSummary, 0, len(cases))
	for _, attentionCase := range cases {
		items = append(items, summarizeAttentionCase(attentionCase))
	}
	return items, total, nil
}

func (s *CaseWorkService) Get(ctx context.Context, id uint) (caseworkres.AttentionCaseDetail, error) {
	if id == 0 {
		return caseworkres.AttentionCaseDetail{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "事项标识必填")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.AttentionCaseDetail{}, normalizeAccessError(err)
	}
	var attentionCase caseworkmodel.AttentionCase
	query := decision.ScopeAttentionCases(s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}), s.now())
	err = query.Where("attention_cases.id = ?", id).First(&attentionCase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return caseworkres.AttentionCaseDetail{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeAccessScopeDenied, "关注事项不存在或不在当前访问范围")
	}
	if err != nil {
		return caseworkres.AttentionCaseDetail{}, err
	}

	var hits []qmodel.QuestionnaireRuleHit
	// Child facts inherit access from the already-authorized aggregate. Explicitly
	// bypass their actor-owned department stamps so a responsible clinician can
	// see guidance recorded by a supervisor in an ancestor department.
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("id = ? AND submission_id = ?", attentionCase.SourceRuleHitID, attentionCase.SubmissionID).
		Order("occurred_at ASC, id ASC").Find(&hits).Error; err != nil {
		return caseworkres.AttentionCaseDetail{}, err
	}
	var actions []caseworkmodel.CaseAction
	if err = s.db().WithContext(ctx).Set("data_scope:skip", true).
		Where("attention_case_id = ?", attentionCase.ID).
		Order("occurred_at ASC, id ASC").Find(&actions).Error; err != nil {
		return caseworkres.AttentionCaseDetail{}, err
	}
	detail := caseworkres.AttentionCaseDetail{
		AttentionCaseSummary: summarizeAttentionCase(attentionCase),
		RuleHits:             make([]caseworkres.RuleHitSummary, 0, len(hits)),
		Actions:              make([]caseworkres.CaseAction, 0, len(actions)),
		HandlingResult:       optionalString(attentionCase.HandlingResult),
		CloseReason:          optionalString(attentionCase.CloseReason),
	}
	for _, hit := range hits {
		detail.RuleHits = append(detail.RuleHits, caseworkres.RuleHitSummary{
			ID: hit.ID, RuleVersionID: hit.RuleVersionID,
			ReasonSnapshot: hit.ReasonSnapshot, OccurredAt: hit.OccurredAt,
		})
	}
	for _, action := range actions {
		detail.Actions = append(detail.Actions, caseworkres.CaseAction{
			ID: action.ID, ActionType: action.ActionType, ActorRole: action.ActorRole,
			Result: action.Result, Reason: optionalString(action.Reason), OccurredAt: action.OccurredAt,
		})
	}
	return detail, nil
}

func summarizeAttentionCase(attentionCase caseworkmodel.AttentionCase) caseworkres.AttentionCaseSummary {
	return caseworkres.AttentionCaseSummary{
		ID: attentionCase.ID, CareClientID: attentionCase.CareClientID, TaskID: attentionCase.TaskID,
		Status: attentionCase.Status, AttentionLevel: attentionCase.AttentionLevel,
		ReasonSummary: attentionCase.ReasonSummary, AssigneeID: attentionCase.AssigneeID,
		OpenedAt: attentionCase.OpenedAt, DueAt: attentionCase.DueAt, Version: attentionCase.Version,
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func validCaseStatus(status string) bool {
	switch status {
	case caseworkmodel.CaseStatusPendingAck,
		caseworkmodel.CaseStatusAcknowledged,
		caseworkmodel.CaseStatusHandling,
		caseworkmodel.CaseStatusWaitingClient,
		caseworkmodel.CaseStatusWaitingCollaboration,
		caseworkmodel.CaseStatusWaitingSupervisor,
		caseworkmodel.CaseStatusResolved,
		caseworkmodel.CaseStatusClosed:
		return true
	default:
		return false
	}
}
