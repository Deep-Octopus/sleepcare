package casework

import (
	"context"
	"fmt"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

// ReconcileRuleHits opens missing cases only for retained test records and only
// while the fixed-fixture gate is enabled. The unique source key makes retries
// safe and existing case history is never rewritten.
func (s *CaseWorkService) ReconcileRuleHits(ctx context.Context) (int, error) {
	if !s.syntheticFixturesEnabled() {
		return 0, nil
	}
	created := 0
	err := s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var hits []qmodel.QuestionnaireRuleHit
		if err := tx.Where("questionnaire_rule_hits.synthetic = ?", true).
			Where(`NOT EXISTS (
				SELECT 1 FROM attention_cases ac
				WHERE ac.source_type = ?
				  AND ac.source_rule_hit_id = questionnaire_rule_hits.id
				  AND ac.dedup_key = questionnaire_rule_hits.dedup_key
				  AND ac.deleted_at IS NULL
			)`, caseworkmodel.CaseSourceRuleHit).
			Order("questionnaire_rule_hits.id ASC").Find(&hits).Error; err != nil {
			return err
		}
		for _, hit := range hits {
			var submission qmodel.QuestionnaireSubmission
			if err := tx.Where("id = ? AND synthetic = ?", hit.SubmissionID, true).First(&submission).Error; err != nil {
				return fmt.Errorf("load retained submission %d for rule hit %d: %w", hit.SubmissionID, hit.ID, err)
			}
			if submission.DeptId == 0 || hit.DeptId != submission.DeptId || uint64(uint(submission.TaskID)) != submission.TaskID {
				return caseworkmodel.NewDomainError(caseworkmodel.CodeInvalidArgument, "历史规则命中记录的归属或任务标识无效")
			}
			actorID := submission.CreatedBy
			if actorID == 0 {
				actorID = hit.CreatedBy
			}
			if actorID == 0 {
				actorID = submission.CareClientID
			}
			ownerCtx := datascope.WithIdentity(ctx, &datascope.Identity{
				UserID: actorID, AuthorityID: 1, DeptID: submission.DeptId,
				DeptIDs: []uint{submission.DeptId}, VisibleDeptIDs: []uint{submission.DeptId}, Scope: datascope.ScopeDept,
			})
			worker := &CaseWorkService{DB: tx, Now: s.Now, SyntheticFixturesEnabled: s.SyntheticFixturesEnabled}
			result, err := worker.OpenFromRuleHits(ownerCtx, OpenFromRuleHitsCommand{
				CareClientID: submission.CareClientID, TaskID: uint(submission.TaskID), SubmissionID: submission.ID,
				RuleHitIDs: []uint{hit.ID}, CorrelationID: fmt.Sprintf("rule-hit-reconcile:%d", hit.ID),
			})
			if err != nil {
				return err
			}
			if len(result.AttentionCaseIDs) == 1 {
				created++
			}
		}
		return nil
	})
	return created, err
}
