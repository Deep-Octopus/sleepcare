package casework

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OpenFromRuleHitsCommand struct {
	CareClientID  uint
	TaskID        uint
	SubmissionID  uint
	RuleHitIDs    []uint
	CorrelationID string
}

type OpenFromRuleHitsResult struct {
	AttentionCaseIDs []uint `json:"attentionCaseIds"`
}

func (s *CaseWorkService) OpenFromRuleHits(ctx context.Context, command OpenFromRuleHitsCommand) (OpenFromRuleHitsResult, error) {
	if len(command.RuleHitIDs) == 0 {
		return OpenFromRuleHitsResult{AttentionCaseIDs: []uint{}}, nil
	}
	if command.CareClientID == 0 || command.TaskID == 0 || command.SubmissionID == 0 {
		return OpenFromRuleHitsResult{}, caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "创建关注事项缺少必要标识")
	}
	result := OpenFromRuleHitsResult{AttentionCaseIDs: []uint{}}
	err := s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var hits []qmodel.QuestionnaireRuleHit
		if err := tx.Where("id IN ? AND submission_id = ?", command.RuleHitIDs, command.SubmissionID).
			Order("id ASC").Find(&hits).Error; err != nil {
			return err
		}
		if len(hits) != len(command.RuleHitIDs) {
			return caseworkmodel.NewDomainError(caseworkmodel.CodeResourceNotFound, "规则命中记录不存在或不属于当前答卷")
		}
		for _, hit := range hits {
			caseID, err := s.openOne(tx, command, hit)
			if err != nil {
				return err
			}
			result.AttentionCaseIDs = append(result.AttentionCaseIDs, caseID)
		}
		return nil
	})
	sort.Slice(result.AttentionCaseIDs, func(i, j int) bool { return result.AttentionCaseIDs[i] < result.AttentionCaseIDs[j] })
	return result, err
}

func (s *CaseWorkService) openOne(tx *gorm.DB, command OpenFromRuleHitsCommand, hit qmodel.QuestionnaireRuleHit) (uint, error) {
	var existing caseworkmodel.AttentionCase
	err := tx.Where("source_type = ? AND source_rule_hit_id = ? AND dedup_key = ?", caseworkmodel.CaseSourceRuleHit, hit.ID, hit.DedupKey).
		First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	assignee, err := activeSteward(tx, command.CareClientID, s.now())
	if err != nil {
		return 0, err
	}
	now := s.now()
	assigneeID := assignee.AssigneeID
	attentionCase := caseworkmodel.AttentionCase{
		CareClientID: command.CareClientID, TaskID: command.TaskID, SubmissionID: command.SubmissionID,
		SourceType: caseworkmodel.CaseSourceRuleHit, SourceRuleHitID: hit.ID, DedupKey: hit.DedupKey,
		Status: caseworkmodel.CaseStatusPendingAck, AttentionLevel: hit.AttentionLevel,
		ReasonSummary: hit.ReasonSnapshot, AssigneeID: &assigneeID, AssigneeRole: caremodel.AssignmentRoleCareSteward,
		OpenedAt: now, Version: 1, Synthetic: hit.Synthetic,
	}
	created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&attentionCase)
	if created.Error != nil {
		return 0, created.Error
	}
	if created.RowsAffected == 0 {
		if err = tx.Where("source_type = ? AND source_rule_hit_id = ? AND dedup_key = ?", caseworkmodel.CaseSourceRuleHit, hit.ID, hit.DedupKey).
			First(&existing).Error; err != nil {
			return 0, err
		}
		return existing.ID, nil
	}
	active := caseworkmodel.TodoActiveSlot
	todo := caseworkmodel.TodoItem{
		Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase,
		SourceID: attentionCase.ID, ActiveSlot: &active, CareClientID: command.CareClientID,
		AssigneeID: assignee.AssigneeID, AssigneeRole: assignee.RoleType, Status: caseworkmodel.TodoStatusOpen,
		OpenedAt: now, Version: 1, Synthetic: hit.Synthetic,
	}
	if err = tx.Create(&todo).Error; err != nil {
		return 0, err
	}
	if err = platformoutbox.Append(tx, platformoutbox.AppendInput{
		EventType: caseworkmodel.EventAttentionCaseOpened, AggregateType: "AttentionCase", AggregateID: attentionCase.ID,
		Payload: map[string]any{
			"attentionCaseId": attentionCase.ID, "careClientId": command.CareClientID, "taskId": command.TaskID,
			"submissionId": command.SubmissionID, "ruleHitId": hit.ID, "status": attentionCase.Status,
			"assigneeId": assignee.AssigneeID, "assigneeRole": assignee.RoleType, "synthetic": hit.Synthetic,
		},
		OccurredAt: now, CorrelationID: command.CorrelationID, CausationID: strconv.FormatUint(uint64(hit.ID), 10),
		Synthetic: hit.Synthetic,
	}); err != nil {
		return 0, err
	}
	return attentionCase.ID, nil
}

func activeSteward(db *gorm.DB, careClientID uint, now time.Time) (caremodel.CareAssignment, error) {
	var assignment caremodel.CareAssignment
	err := db.Where("care_client_id = ? AND role_type = ? AND cancelled_at IS NULL AND valid_from <= ?", careClientID, caremodel.AssignmentRoleCareSteward, now).
		Where("valid_until IS NULL OR valid_until > ?", now).
		Order("valid_from DESC, id DESC").First(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return assignment, caseworkmodel.NewDomainError(caseworkmodel.CodeCareAssignmentRequired, "当前康养用户缺少有效责任管家")
	}
	return assignment, err
}

func duplicateError(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique")
}
