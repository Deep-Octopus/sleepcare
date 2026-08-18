package questionnaire

import (
	"context"
	"errors"

	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"gorm.io/gorm"
)

// PrepareTaskDraft validates a partial answer object against the immutable
// task binding and returns its canonical JSON representation. Required-answer
// completeness remains a final-submission concern.
func (s *QuestionnaireService) PrepareTaskDraft(ctx context.Context, binding FrozenTaskBinding, answers map[string]any) (map[string]any, []byte, error) {
	if err := s.ValidateFrozenBinding(ctx, binding.QuestionnaireVersionID, binding.RuleVersionIDs, binding.Synthetic); err != nil {
		return nil, nil, err
	}
	_, questions, options, _, err := loadDefinition(ctx, s.db().WithContext(ctx), binding.QuestionnaireVersionID)
	if err != nil {
		return nil, nil, err
	}
	normalized, canonical, err := canonicalAnswers(answers)
	if err != nil {
		return nil, nil, err
	}
	if err = validatePartialAnswers(questions, options, normalized); err != nil {
		return nil, nil, err
	}
	return normalized, canonical, nil
}

// ValidateFrozenBinding is the module boundary used by plan creation. It
// validates published content gates without exposing questionnaire internals or
// creating a submission.
func (s *QuestionnaireService) ValidateFrozenBinding(ctx context.Context, questionnaireVersionID uint, ruleVersionIDs []uint, synthetic bool) error {
	if questionnaireVersionID == 0 {
		if len(ruleVersionIDs) != 0 {
			return qmodel.NewDomainError(qmodel.CodeInvalidArgument, "未绑定问卷时不能绑定关注规则")
		}
		return nil
	}
	version, questions, options, rules, err := loadDefinition(ctx, s.db().WithContext(ctx), questionnaireVersionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return qmodel.NewDomainError(qmodel.CodeResourceNotFound, "计划绑定的问卷版本不存在")
	}
	if err != nil {
		return err
	}
	binding := FrozenTaskBinding{QuestionnaireVersionID: questionnaireVersionID, Synthetic: synthetic}
	if err = validateQuestionnaireGate(version, binding, s.syntheticFixturesEnabled()); err != nil {
		return err
	}
	if err = verifyDefinitionHash(version, questions, options); err != nil {
		return err
	}
	selected, err := selectBoundRules(rules, ruleVersionIDs)
	if err != nil {
		return err
	}
	for _, rule := range selected {
		executable, gateErr := ruleExecutable(rule, s.syntheticFixturesEnabled(), synthetic)
		if gateErr != nil {
			return gateErr
		}
		if !executable {
			return qmodel.NewDomainError(qmodel.CodeContentNotPublished, "计划绑定的关注规则版本未发布或当前环境不可执行")
		}
		if err = verifyRuleHash(rule); err != nil {
			return err
		}
	}
	return nil
}
