package questionnaire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	qreq "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire/request"
	qres "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire/response"
	"gorm.io/gorm"
)

func (s *QuestionnaireService) ListVersions(ctx context.Context, req qreq.QuestionnaireVersionSearch) ([]qres.QuestionnaireVersionSummary, int64, error) {
	db := s.db()
	if _, err := accesspolicy.ResolveQuestionnaire(ctx, db); err != nil {
		return nil, 0, err
	}
	query := db.WithContext(ctx).Model(&qmodel.QuestionnaireVersion{})
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("code LIKE ? OR title LIKE ? OR version LIKE ?", like, like, like)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.UsageScope != "" {
		query = query.Where("usage_scope = ?", req.UsageScope)
	}
	if req.Synthetic != nil {
		query = query.Where("synthetic = ?", *req.Synthetic)
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
	var versions []qmodel.QuestionnaireVersion
	if err := query.Order("published_at DESC, id DESC").Limit(limit).Offset(offset).Find(&versions).Error; err != nil {
		return nil, 0, err
	}
	questionCounts, err := countByVersion(ctx, db, &qmodel.QuestionnaireQuestion{}, "questionnaire_version_id", versionIDs(versions))
	if err != nil {
		return nil, 0, err
	}
	ruleCounts, err := countByVersion(ctx, db, &qmodel.QuestionnaireRuleVersion{}, "questionnaire_version_id", versionIDs(versions))
	if err != nil {
		return nil, 0, err
	}
	items := make([]qres.QuestionnaireVersionSummary, 0, len(versions))
	for _, version := range versions {
		items = append(items, summary(version, questionCounts[version.ID], ruleCounts[version.ID]))
	}
	return items, total, nil
}

func (s *QuestionnaireService) GetVersion(ctx context.Context, id uint) (qres.QuestionnaireVersionDetail, error) {
	db := s.db()
	if _, err := accesspolicy.ResolveQuestionnaire(ctx, db); err != nil {
		return qres.QuestionnaireVersionDetail{}, err
	}
	version, questions, options, rules, err := loadDefinition(ctx, db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return qres.QuestionnaireVersionDetail{}, qmodel.NewDomainError(qmodel.CodeResourceNotFound, "问卷版本不存在")
	}
	if err != nil {
		return qres.QuestionnaireVersionDetail{}, err
	}
	if err = verifyDefinitionHash(version, questions, options); err != nil {
		return qres.QuestionnaireVersionDetail{}, err
	}
	for _, rule := range rules {
		if err = verifyRuleHash(rule); err != nil {
			return qres.QuestionnaireVersionDetail{}, err
		}
	}
	return buildDetail(version, questions, options, rules)
}

func loadDefinition(ctx context.Context, db *gorm.DB, id uint) (qmodel.QuestionnaireVersion, []qmodel.QuestionnaireQuestion, []qmodel.QuestionnaireOption, []qmodel.QuestionnaireRuleVersion, error) {
	var version qmodel.QuestionnaireVersion
	if err := db.WithContext(ctx).Where("id = ?", id).First(&version).Error; err != nil {
		return version, nil, nil, nil, err
	}
	var questions []qmodel.QuestionnaireQuestion
	if err := db.WithContext(ctx).Where("questionnaire_version_id = ?", id).Order("sort ASC, id ASC").Find(&questions).Error; err != nil {
		return version, nil, nil, nil, err
	}
	questionIDs := make([]uint, 0, len(questions))
	for _, question := range questions {
		questionIDs = append(questionIDs, question.ID)
	}
	var options []qmodel.QuestionnaireOption
	if len(questionIDs) > 0 {
		if err := db.WithContext(ctx).Where("question_id IN ?", questionIDs).Order("sort ASC, id ASC").Find(&options).Error; err != nil {
			return version, nil, nil, nil, err
		}
	}
	var rules []qmodel.QuestionnaireRuleVersion
	if err := db.WithContext(ctx).Where("questionnaire_version_id = ?", id).Order("id ASC").Find(&rules).Error; err != nil {
		return version, nil, nil, nil, err
	}
	return version, questions, options, rules, nil
}

func buildVersionDefinition(version qmodel.QuestionnaireVersion, questions []qmodel.QuestionnaireQuestion, options []qmodel.QuestionnaireOption) qmodel.VersionDefinition {
	byQuestion := make(map[uint][]qmodel.OptionDefinition)
	for _, option := range options {
		byQuestion[option.QuestionID] = append(byQuestion[option.QuestionID], qmodel.OptionDefinition{Code: option.Code, Label: option.Label, Sort: option.Sort})
	}
	definition := qmodel.VersionDefinition{
		Code: version.Code, Version: version.Version, Title: version.Title, Purpose: version.Purpose,
		UsageScope: version.UsageScope, Synthetic: version.Synthetic, ProductionEnabled: version.ProductionEnabled,
		ExpectedMinutes: version.ExpectedMinutes, DefinitionSchemaVersion: version.DefinitionSchemaVersion,
		Questions: make([]qmodel.QuestionDefinition, 0, len(questions)),
	}
	for _, question := range questions {
		definition.Questions = append(definition.Questions, qmodel.QuestionDefinition{
			Code: question.Code, Type: question.Type, Title: question.Title, Required: question.Required, Sort: question.Sort,
			ValidationSchemaVersion: question.ValidationSchemaVersion, Validation: qmodel.CanonicalJSON(json.RawMessage(question.ValidationJSON)),
			Options: byQuestion[question.ID],
		})
	}
	return definition
}

func buildRuleDefinition(rule qmodel.QuestionnaireRuleVersion) qmodel.RuleDefinition {
	return qmodel.RuleDefinition{
		QuestionnaireVersionID: rule.QuestionnaireVersionID, Code: rule.Code, Version: rule.Version, Title: rule.Title,
		UsageScope: rule.UsageScope, Synthetic: rule.Synthetic, ProductionEnabled: rule.ProductionEnabled,
		ConditionSchemaVersion: rule.ConditionSchemaVersion, Condition: qmodel.CanonicalJSON(json.RawMessage(rule.ConditionJSON)),
		AttentionLevel: rule.AttentionLevel, ReasonSnapshot: rule.ReasonSnapshot, Recipients: qmodel.CanonicalJSON(json.RawMessage(rule.RecipientsJSON)),
		SLAFirstResponseMinutes: rule.SLAFirstResponseMinutes, SLAResolutionMinutes: rule.SLAResolutionMinutes,
		DedupKeyTemplate: rule.DedupKeyTemplate,
	}
}

func verifyDefinitionHash(version qmodel.QuestionnaireVersion, questions []qmodel.QuestionnaireQuestion, options []qmodel.QuestionnaireOption) error {
	hash, err := qmodel.HashDefinition(buildVersionDefinition(version, questions, options))
	if err != nil {
		return err
	}
	if hash != version.DefinitionHash {
		return qmodel.NewDomainError(qmodel.CodeOperationNotAllowed, "问卷版本定义哈希不匹配，已拒绝使用")
	}
	return nil
}

func verifyRuleHash(rule qmodel.QuestionnaireRuleVersion) error {
	hash, err := qmodel.HashDefinition(buildRuleDefinition(rule))
	if err != nil {
		return err
	}
	if hash != rule.DefinitionHash {
		return qmodel.NewDomainError(qmodel.CodeOperationNotAllowed, "关注规则定义哈希不匹配，已拒绝使用")
	}
	return nil
}

func buildDetail(version qmodel.QuestionnaireVersion, questions []qmodel.QuestionnaireQuestion, options []qmodel.QuestionnaireOption, rules []qmodel.QuestionnaireRuleVersion) (qres.QuestionnaireVersionDetail, error) {
	byQuestion := make(map[uint][]qres.QuestionnaireOption)
	for _, option := range options {
		byQuestion[option.QuestionID] = append(byQuestion[option.QuestionID], qres.QuestionnaireOption{ID: option.ID, Code: option.Code, Label: option.Label, Order: option.Sort})
	}
	questionResponses := make([]qres.QuestionnaireQuestion, 0, len(questions))
	for _, question := range questions {
		validation := map[string]any{}
		if err := json.Unmarshal(question.ValidationJSON, &validation); err != nil {
			return qres.QuestionnaireVersionDetail{}, fmt.Errorf("decode validation for question %s: %w", question.Code, err)
		}
		questionResponses = append(questionResponses, qres.QuestionnaireQuestion{
			ID: question.ID, Code: question.Code, Type: question.Type, Title: question.Title, Required: question.Required,
			Order: question.Sort, ValidationSchemaVersion: question.ValidationSchemaVersion, Validation: validation,
			Options: nonNilOptions(byQuestion[question.ID]),
		})
	}
	ruleResponses := make([]qres.QuestionnaireRuleVersion, 0, len(rules))
	for _, rule := range rules {
		condition := map[string]any{}
		var recipients []string
		if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
			return qres.QuestionnaireVersionDetail{}, fmt.Errorf("decode condition for rule %s: %w", rule.Code, err)
		}
		if err := json.Unmarshal(rule.RecipientsJSON, &recipients); err != nil {
			return qres.QuestionnaireVersionDetail{}, fmt.Errorf("decode recipients for rule %s: %w", rule.Code, err)
		}
		ruleResponses = append(ruleResponses, qres.QuestionnaireRuleVersion{
			ID: rule.ID, Code: rule.Code, Version: rule.Version, Title: rule.Title, LifecycleStatus: rule.Status,
			UsageScope: rule.UsageScope, Synthetic: rule.Synthetic, ProductionEnabled: rule.ProductionEnabled,
			ReviewRecord:           review(rule.ReviewType, rule.ReviewedBy, rule.ReviewedAt, rule.ReviewNote),
			ConditionSchemaVersion: rule.ConditionSchemaVersion, Condition: condition,
			AttentionLevel: rule.AttentionLevel, ReasonSnapshot: rule.ReasonSnapshot, Recipients: nonNilStrings(recipients),
			SLAFirstResponseMinutes: rule.SLAFirstResponseMinutes, SLAResolutionMinutes: rule.SLAResolutionMinutes,
			DedupKeyTemplate: rule.DedupKeyTemplate, PublishedAt: rule.PublishedAt, DefinitionHash: rule.DefinitionHash,
		})
	}
	return qres.QuestionnaireVersionDetail{
		QuestionnaireVersionSummary: summary(version, int64(len(questions)), int64(len(rules))),
		DefinitionSchemaVersion:     version.DefinitionSchemaVersion, ReplacesVersionID: version.ReplacesVersionID,
		Questions: questionResponses, Rules: ruleResponses,
	}, nil
}

func summary(version qmodel.QuestionnaireVersion, questionCount, ruleCount int64) qres.QuestionnaireVersionSummary {
	return qres.QuestionnaireVersionSummary{
		ID: version.ID, Code: version.Code, Version: version.Version, Title: version.Title, Purpose: version.Purpose,
		LifecycleStatus: version.Status, UsageScope: version.UsageScope, Synthetic: version.Synthetic,
		ProductionEnabled: version.ProductionEnabled,
		ReviewRecord:      review(version.ReviewType, version.ReviewedBy, version.ReviewedAt, version.ReviewNote),
		ExpectedMinutes:   version.ExpectedMinutes, QuestionCount: questionCount, RuleCount: ruleCount,
		PublishedAt: version.PublishedAt, DefinitionHash: version.DefinitionHash,
	}
}

func review(kind string, by uint, at *time.Time, note string) qres.ReviewRecord {
	reviewType := "ENGINEERING_FIXTURE_REVIEW"
	formalApproval := false
	if kind == qmodel.ReviewTypeFormal {
		reviewType = "FORMAL_MEDICAL_REVIEW"
		formalApproval = true
	}
	return qres.ReviewRecord{
		ReviewType: reviewType, ReviewedBy: fmt.Sprint(by), ReviewedAt: at,
		FormalMedicalApproval: formalApproval, Note: note,
	}
}

func versionIDs(versions []qmodel.QuestionnaireVersion) []uint {
	ids := make([]uint, 0, len(versions))
	for _, version := range versions {
		ids = append(ids, version.ID)
	}
	return ids
}

type versionCount struct {
	VersionID uint
	Count     int64
}

func countByVersion(ctx context.Context, db *gorm.DB, model any, column string, ids []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64)
	if len(ids) == 0 {
		return counts, nil
	}
	var rows []versionCount
	err := db.WithContext(ctx).Model(model).Select(column+" AS version_id, COUNT(*) AS count").Where(column+" IN ?", ids).Group(column).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.VersionID] = row.Count
	}
	return counts, nil
}

func nonNilOptions(items []qres.QuestionnaireOption) []qres.QuestionnaireOption {
	if items == nil {
		return []qres.QuestionnaireOption{}
	}
	return items
}

func nonNilStrings(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}
