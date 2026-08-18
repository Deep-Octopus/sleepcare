package casework

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"gorm.io/gorm"
)

type SupervisorActionCommand struct {
	AttentionCaseID       uint
	ExpectedVersion       uint
	ActionType            string
	Result                string
	ResponsibleAssigneeID uint
	DueAt                 time.Time
	CommandKeyDigest      string
}

// ApplySupervisorAction is the transaction-aware caseWork seam used by the
// supervision application service. The caller injects its transaction through
// CaseWorkService.DB so the guidance record, case action, todo and outbox fact
// commit or roll back together.
func (s *CaseWorkService) ApplySupervisorAction(ctx context.Context, command SupervisorActionCommand) (caseworkres.ActionResult, error) {
	if command.AttentionCaseID == 0 || command.ExpectedVersion == 0 ||
		strings.TrimSpace(command.Result) == "" || command.ResponsibleAssigneeID == 0 ||
		command.DueAt.IsZero() || !command.DueAt.After(s.now()) || strings.TrimSpace(command.CommandKeyDigest) == "" {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "督导动作参数不完整")
	}
	if command.ActionType != caseworkmodel.CaseActionGuidance && command.ActionType != caseworkmodel.CaseActionIntervene {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "督导动作类型无效")
	}

	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleSupervisor {
		return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeAccessScopeDenied, "当前角色不能执行上级督导动作")
	}

	attentionCase, err := s.loadActionableCase(s.db().WithContext(ctx), decision, command.AttentionCaseID)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	if attentionCase.Version != command.ExpectedVersion {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
	}
	if !attentionCase.Synthetic {
		return caseworkres.ActionResult{}, caseworkmodel.NewForbiddenError(caseworkmodel.CodeAccessScopeDenied, "当前阶段只允许处理固定测试事项")
	}
	if attentionCase.Status != caseworkmodel.CaseStatusWaitingSupervisor {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseTransitionDenied, "关注事项当前不在待上级复核状态")
	}
	target, err := activeAssignmentForTarget(
		s.db().WithContext(ctx), attentionCase.CareClientID, command.ResponsibleAssigneeID,
		caremodel.AssignmentRoleClinician, s.now(),
	)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}

	now := s.now()
	nextStatus := caseworkmodel.CaseStatusWaitingSupervisor
	if command.ActionType == caseworkmodel.CaseActionIntervene {
		nextStatus = caseworkmodel.CaseStatusHandling
	}
	action := caseworkmodel.CaseAction{
		AttentionCaseID:  attentionCase.ID,
		ActionType:       command.ActionType,
		ActorID:          decision.Identity.UserID,
		ActorRole:        decision.RoleType,
		OrganizationID:   target.OrganizationID,
		TeamID:           target.TeamID,
		Source:           caseworkmodel.ActionSourceStaff,
		Result:           strings.TrimSpace(command.Result),
		FromStatus:       attentionCase.Status,
		ToStatus:         nextStatus,
		TargetAssigneeID: &target.AssigneeID,
		TargetRole:       target.RoleType,
		DueAt:            &command.DueAt,
		OccurredAt:       now,
		CommandKeyDigest: strings.TrimSpace(command.CommandKeyDigest),
		Synthetic:        attentionCase.Synthetic,
	}
	if err = s.db().WithContext(ctx).Create(&action).Error; err != nil {
		return caseworkres.ActionResult{}, err
	}

	todoUpdate := s.db().WithContext(ctx).Model(&caseworkmodel.TodoItem{}).
		Where("source_type = ? AND source_id = ? AND active_slot = ?", caseworkmodel.TodoSourceAttentionCase, attentionCase.ID, caseworkmodel.TodoActiveSlot).
		Updates(map[string]any{
			"assignee_id":   target.AssigneeID,
			"assignee_role": target.RoleType,
			"due_at":        command.DueAt,
			"version":       gorm.Expr("version + 1"),
		})
	if todoUpdate.Error != nil {
		return caseworkres.ActionResult{}, todoUpdate.Error
	}
	if todoUpdate.RowsAffected != 1 {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeCaseResponsibilityRequired, "关注事项缺少唯一活动待办")
	}

	caseUpdate := s.db().WithContext(ctx).Model(&caseworkmodel.AttentionCase{}).
		Where("id = ? AND version = ?", attentionCase.ID, command.ExpectedVersion).
		Updates(map[string]any{
			"status":        nextStatus,
			"assignee_id":   target.AssigneeID,
			"assignee_role": target.RoleType,
			"due_at":        command.DueAt,
			"version":       gorm.Expr("version + 1"),
		})
	if caseUpdate.Error != nil {
		return caseworkres.ActionResult{}, caseUpdate.Error
	}
	if caseUpdate.RowsAffected != 1 {
		return caseworkres.ActionResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeVersionConflict, "关注事项版本已变化")
	}
	if err = appendCaseEvent(s.db().WithContext(ctx), attentionCase, action, caseworkmodel.EventSupervisorGuidanceAdded); err != nil {
		return caseworkres.ActionResult{}, err
	}

	return caseworkres.ActionResult{
		ResourceID: attentionCase.ID,
		ActionID:   action.ID,
		Status:     nextStatus,
		Version:    command.ExpectedVersion + 1,
		OccurredAt: now,
	}, nil
}
