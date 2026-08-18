package questionnaire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	qreq "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var questionnaireModels = []any{
	&qmodel.QuestionnaireVersion{}, &qmodel.QuestionnaireQuestion{}, &qmodel.QuestionnaireOption{},
	&qmodel.QuestionnaireRuleVersion{}, &qmodel.QuestionnaireSubmission{}, &qmodel.QuestionnaireAnswerRevision{},
	&qmodel.QuestionnaireRuleHit{}, &qmodel.QuestionnaireCommandReceipt{}, &qmodel.OutboxEvent{},
}

func TestValidateAnswersSupportsSixBasicTypes(t *testing.T) {
	minLength, maxLength := 2, 5
	minNumber, maxNumber := 1.0, 10.0
	questions := []qmodel.QuestionnaireQuestion{
		{GVA_MODEL: global.GVA_MODEL{ID: 1}, Code: "single", Type: qmodel.QuestionTypeSingleChoice, Required: true, ValidationJSON: jsonBytes(map[string]any{})},
		{GVA_MODEL: global.GVA_MODEL{ID: 2}, Code: "multiple", Type: qmodel.QuestionTypeMultipleChoice, Required: true, ValidationJSON: jsonBytes(map[string]any{})},
		{GVA_MODEL: global.GVA_MODEL{ID: 3}, Code: "text", Type: qmodel.QuestionTypeText, Required: true, ValidationJSON: jsonBytes(validationRules{MinLength: &minLength, MaxLength: &maxLength})},
		{GVA_MODEL: global.GVA_MODEL{ID: 4}, Code: "number", Type: qmodel.QuestionTypeNumber, Required: true, ValidationJSON: jsonBytes(validationRules{Min: &minNumber, Max: &maxNumber})},
		{GVA_MODEL: global.GVA_MODEL{ID: 5}, Code: "date", Type: qmodel.QuestionTypeDate, Required: true, ValidationJSON: jsonBytes(map[string]any{})},
		{GVA_MODEL: global.GVA_MODEL{ID: 6}, Code: "boolean", Type: qmodel.QuestionTypeBoolean, Required: true, ValidationJSON: jsonBytes(map[string]any{})},
	}
	options := []qmodel.QuestionnaireOption{
		{QuestionID: 1, Code: "A"}, {QuestionID: 1, Code: "B"},
		{QuestionID: 2, Code: "A"}, {QuestionID: 2, Code: "B"},
	}
	valid, _, err := canonicalAnswers(map[string]any{
		"single": "A", "multiple": []string{"A", "B"}, "text": "合成", "number": 5,
		"date": "2026-08-18", "boolean": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = validateAnswers(questions, options, valid); err != nil {
		t.Fatalf("valid answers rejected: %v", err)
	}

	cases := []map[string]any{
		{"single": "UNKNOWN", "multiple": []string{"A"}, "text": "合成", "number": 5, "date": "2026-08-18", "boolean": true},
		{"single": "A", "multiple": []string{"A", "A"}, "text": "合成", "number": 5, "date": "2026-08-18", "boolean": true},
		{"single": "A", "multiple": []string{"A"}, "text": "过长的合成文本", "number": 5, "date": "2026-08-18", "boolean": true},
		{"single": "A", "multiple": []string{"A"}, "text": "合成", "number": 11, "date": "2026-08-18", "boolean": true},
		{"single": "A", "multiple": []string{"A"}, "text": "合成", "number": 5, "date": "18-08-2026", "boolean": true},
		{"single": "A", "multiple": []string{"A"}, "text": "合成", "number": 5, "date": "2026-08-18", "boolean": "true"},
	}
	for i, answers := range cases {
		normalized, _, normalizeErr := canonicalAnswers(answers)
		if normalizeErr != nil {
			t.Fatalf("case %d normalization failed: %v", i, normalizeErr)
		}
		if err = validateAnswers(questions, options, normalized); err == nil {
			t.Fatalf("invalid case %d was accepted", i)
		}
	}
}

func TestRecordSubmissionFreezesRulesIsIdempotentAndRevisionDoesNotReevaluate(t *testing.T) {
	db := newQuestionnaireDB(t, true)
	fixture := seedServiceDefinition(t, db, qmodel.LifecyclePublished)
	service := newTestService(db)
	ctx := submissionContext()
	binding := FrozenTaskBinding{
		TaskID: 70001, CareClientID: 20001, QuestionnaireVersionID: fixture.version.ID,
		RuleVersionIDs: []uint{fixture.rule.ID}, DeptID: 9101, Synthetic: true,
	}
	command := submissionCommand("submission-key", "CONTINUE_WITH_ATTENTION")
	first, err := service.RecordSubmission(ctx, binding, command)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.RuleHitIDs) != 1 || first.RuleExecutionDisabled {
		t.Fatalf("unexpected first result: %+v", first)
	}
	second, err := service.RecordSubmission(ctx, binding, command)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Fatalf("idempotent result differs: %+v vs %+v", first, second)
	}
	changed := submissionCommand("submission-key", "CONTINUE_WITHOUT_ATTENTION")
	if _, err = service.RecordSubmission(ctx, binding, changed); domainCode(err) != qmodel.CodeIdempotencyConflict {
		t.Fatalf("changed request should conflict, got %v", err)
	}

	newRule := seedRule(t, db, fixture.version.ID, "SYN-LATER-RULE", qmodel.LifecyclePublished)
	secondBinding := binding
	secondBinding.TaskID = 70002
	secondBinding.RuleVersionIDs = []uint{fixture.rule.ID}
	secondCommand := submissionCommand("second-key", "CONTINUE_WITH_ATTENTION")
	secondSubmission, err := service.RecordSubmission(ctx, secondBinding, secondCommand)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondSubmission.RuleHitIDs) != 1 {
		t.Fatalf("new unbound rule %d changed frozen evaluation: %+v", newRule.ID, secondSubmission)
	}

	revision, err := service.AppendAnswerRevision(ctx, AppendAnswerRevisionCommand{
		IdempotencyKey: "revision-key", SubmissionID: first.SubmissionID, ExpectedRevisionNo: 1,
		Answers: map[string]any{"synthetic_process_confirmation": "CONTINUE_WITHOUT_ATTENTION"},
		Reason:  "合成更正验证", ActorKind: qmodel.ActorKindStaff, ActorID: 9202,
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.RevisionNo != 2 {
		t.Fatalf("revision=%+v", revision)
	}
	assertCount(t, db, &qmodel.QuestionnaireRuleHit{}, "submission_id = ?", first.SubmissionID, 1)
	assertCount(t, db, &qmodel.OutboxEvent{}, "aggregate_id = ?", fmt.Sprint(first.SubmissionID), 2)
	assertCount(t, db, &qmodel.QuestionnaireAnswerRevision{}, "submission_id = ?", first.SubmissionID, 2)

	var originalHit qmodel.QuestionnaireRuleHit
	if err = db.Where("id = ?", first.RuleHitIDs[0]).First(&originalHit).Error; err != nil {
		t.Fatal(err)
	}
	duplicate := originalHit
	duplicate.ID = 0
	if err = db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate rule hit should be rejected")
	}
}

func TestRecordSubmissionSkipsUnpublishedAndDisabledRules(t *testing.T) {
	for _, status := range []string{qmodel.LifecycleApproved, qmodel.LifecycleDisabled} {
		t.Run(status, func(t *testing.T) {
			db := newQuestionnaireDB(t, true)
			fixture := seedServiceDefinition(t, db, status)
			service := newTestService(db)
			result, err := service.RecordSubmission(submissionContext(), FrozenTaskBinding{
				TaskID: 71000 + uint64(fixture.rule.ID), CareClientID: 20001, QuestionnaireVersionID: fixture.version.ID,
				RuleVersionIDs: []uint{fixture.rule.ID}, DeptID: 9101, Synthetic: true,
			}, submissionCommand("skip-"+status, "CONTINUE_WITH_ATTENTION"))
			if err != nil {
				t.Fatal(err)
			}
			if !result.RuleExecutionDisabled || len(result.RuleHitIDs) != 0 {
				t.Fatalf("rule should be skipped: %+v", result)
			}
			assertCount(t, db, &qmodel.OutboxEvent{}, "aggregate_id = ?", fmt.Sprint(result.SubmissionID), 1)
		})
	}
}

func TestPublicationGatesSeparateTestOnlyAndFormalContent(t *testing.T) {
	now := time.Now()
	testVersion := qmodel.QuestionnaireVersion{
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true,
		ProductionEnabled: false, ReviewType: qmodel.ReviewTypeEngineering, ReviewedAt: &now, PublishedAt: &now,
	}
	syntheticBinding := FrozenTaskBinding{Synthetic: true}
	if err := validateQuestionnaireGate(testVersion, syntheticBinding, true); err != nil {
		t.Fatalf("valid test-only gate rejected: %v", err)
	}
	if err := validateQuestionnaireGate(testVersion, syntheticBinding, false); domainCode(err) != qmodel.CodeContentDisabled {
		t.Fatalf("disabled fixtures should reject test-only content: %v", err)
	}

	formalVersion := qmodel.QuestionnaireVersion{
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeFormal, Synthetic: false,
		ProductionEnabled: true, ReviewType: qmodel.ReviewTypeFormal, ReviewedAt: &now, PublishedAt: &now,
	}
	if err := validateQuestionnaireGate(formalVersion, FrozenTaskBinding{Synthetic: false}, false); err != nil {
		t.Fatalf("valid formal gate rejected: %v", err)
	}
	formalVersion.ReviewType = qmodel.ReviewTypeEngineering
	if err := validateQuestionnaireGate(formalVersion, FrozenTaskBinding{Synthetic: false}, false); domainCode(err) != qmodel.CodeContentDisabled {
		t.Fatalf("formal content without formal review should be rejected: %v", err)
	}

	testRule := qmodel.QuestionnaireRuleVersion{
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true,
		ReviewType: qmodel.ReviewTypeEngineering, ReviewedAt: &now, PublishedAt: &now,
	}
	if allowed, err := ruleExecutable(testRule, true, true); err != nil || !allowed {
		t.Fatalf("valid test rule gate rejected: allowed=%v err=%v", allowed, err)
	}
	if allowed, err := ruleExecutable(testRule, true, false); err != nil || allowed {
		t.Fatalf("test rule must not run for formal submission: allowed=%v err=%v", allowed, err)
	}
	formalRule := qmodel.QuestionnaireRuleVersion{
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeFormal, ProductionEnabled: true,
		ReviewType: qmodel.ReviewTypeFormal, ReviewedAt: &now, PublishedAt: &now,
	}
	if allowed, err := ruleExecutable(formalRule, false, false); err != nil || !allowed {
		t.Fatalf("valid formal rule gate rejected: allowed=%v err=%v", allowed, err)
	}
	if allowed, err := ruleExecutable(formalRule, true, true); err != nil || allowed {
		t.Fatalf("formal rule must not run for synthetic submission: allowed=%v err=%v", allowed, err)
	}
}

func TestRecordSubmissionRollsBackWhenOutboxWriteFails(t *testing.T) {
	modelsWithoutOutbox := append([]any{}, questionnaireModels[:len(questionnaireModels)-1]...)
	db := testutil.NewMemoryDB(t, append(modelsWithoutOutbox, testutil.WithDataScopeCallbacks())...)
	fixture := seedServiceDefinition(t, db, qmodel.LifecyclePublished)
	service := newTestService(db)
	_, err := service.RecordSubmission(submissionContext(), FrozenTaskBinding{
		TaskID: 72001, CareClientID: 20001, QuestionnaireVersionID: fixture.version.ID,
		RuleVersionIDs: []uint{fixture.rule.ID}, DeptID: 9101, Synthetic: true,
	}, submissionCommand("rollback-key", "CONTINUE_WITH_ATTENTION"))
	if err == nil {
		t.Fatal("expected missing outbox table error")
	}
	assertCount(t, db, &qmodel.QuestionnaireSubmission{}, "task_id = ?", 72001, 0)
	assertCount(t, db, &qmodel.QuestionnaireAnswerRevision{}, "1 = 1", 0, 0)
	assertCount(t, db, &qmodel.QuestionnaireCommandReceipt{}, "1 = 1", 0, 0)
}

func TestQuestionnaireReadAccessAndDefinitionHash(t *testing.T) {
	db := newQuestionnaireDB(t, false)
	fixture := seedServiceDefinition(t, db, qmodel.LifecyclePublished)
	profiles := []careclient.CareAuthorityProfile{
		{AuthorityID: 9801, RoleType: careclient.AuthorityRoleCareSteward, Active: true, Synthetic: true},
		{AuthorityID: 9802, RoleType: careclient.AuthorityRoleClinician, Active: true, Synthetic: true},
		{AuthorityID: 9803, RoleType: careclient.AuthorityRoleSupervisor, Active: true, Synthetic: true},
		{AuthorityID: 9804, RoleType: careclient.AuthorityRoleContentAdmin, Active: true, Synthetic: true},
	}
	if err := db.Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	service := newTestService(db)
	for _, role := range []uint{9802, 9803, 9804} {
		ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 9202, AuthorityID: role})
		items, total, err := service.ListVersions(ctx, qreq.QuestionnaireVersionSearch{})
		if err != nil || total != 1 || len(items) != 1 {
			t.Fatalf("role %d list failed: total=%d items=%d err=%v", role, total, len(items), err)
		}
		detail, err := service.GetVersion(ctx, fixture.version.ID)
		if err != nil || len(detail.Questions) != 1 || len(detail.Rules) != 1 {
			t.Fatalf("role %d detail failed: %+v err=%v", role, detail, err)
		}
	}
	for _, role := range []uint{9801, 888} {
		ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 9201, AuthorityID: role})
		if _, _, err := service.ListVersions(ctx, qreq.QuestionnaireVersionSearch{}); domainCode(err) != qmodel.CodeAccessScopeDenied {
			t.Fatalf("role %d should be denied, got %v", role, err)
		}
	}
	if err := db.Model(&qmodel.QuestionnaireQuestion{}).Where("id = ?", fixture.question.ID).Update("title", "被篡改的合成题目").Error; err != nil {
		t.Fatal(err)
	}
	ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{UserID: 9202, AuthorityID: 9802})
	if _, err := service.GetVersion(ctx, fixture.version.ID); domainCode(err) != qmodel.CodeOperationNotAllowed {
		t.Fatalf("hash mismatch should fail, got %v", err)
	}
}

func TestValidateFrozenBindingRejectsQuestionnaireAndRuleHashMismatches(t *testing.T) {
	db := newQuestionnaireDB(t, false)
	fixture := seedServiceDefinition(t, db, qmodel.LifecyclePublished)
	service := newTestService(db)
	ctx := context.Background()

	if err := service.ValidateFrozenBinding(ctx, fixture.version.ID, []uint{fixture.rule.ID}, true); err != nil {
		t.Fatalf("valid frozen binding rejected: %v", err)
	}
	if err := db.Model(&qmodel.QuestionnaireQuestion{}).Where("id = ?", fixture.question.ID).
		Update("title", "被篡改的合成题目").Error; err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateFrozenBinding(ctx, fixture.version.ID, []uint{fixture.rule.ID}, true); domainCode(err) != qmodel.CodeOperationNotAllowed {
		t.Fatalf("questionnaire hash mismatch should reject binding, got %v", err)
	}
	if err := db.Model(&qmodel.QuestionnaireQuestion{}).Where("id = ?", fixture.question.ID).
		Update("title", fixture.question.Title).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&qmodel.QuestionnaireRuleVersion{}).Where("id = ?", fixture.rule.ID).
		Update("title", "被篡改的合成规则").Error; err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateFrozenBinding(ctx, fixture.version.ID, []uint{fixture.rule.ID}, true); domainCode(err) != qmodel.CodeOperationNotAllowed {
		t.Fatalf("rule hash mismatch should reject binding, got %v", err)
	}
}

type serviceFixture struct {
	version  qmodel.QuestionnaireVersion
	question qmodel.QuestionnaireQuestion
	rule     qmodel.QuestionnaireRuleVersion
}

func newQuestionnaireDB(t *testing.T, dataScope bool) *gorm.DB {
	models := append([]any{}, questionnaireModels...)
	models = append(models, &careclient.CareAuthorityProfile{})
	if dataScope {
		models = append(models, testutil.WithDataScopeCallbacks())
	}
	return testutil.NewMemoryDB(t, models...)
}

func newTestService(db *gorm.DB) *QuestionnaireService {
	enabled := true
	fixed := time.Date(2026, time.August, 18, 10, 0, 0, 0, time.UTC)
	return &QuestionnaireService{DB: db, Now: func() time.Time { return fixed }, SyntheticFixturesEnabled: &enabled}
}

func seedServiceDefinition(t *testing.T, db *gorm.DB, ruleStatus string) serviceFixture {
	t.Helper()
	published := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.UTC)
	versionDefinition := qmodel.VersionDefinition{
		Code: "SYN-SERVICE-TEST", Version: "1.0.0", Title: "合成服务测试问卷", Purpose: "合成软件验证",
		UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false, ExpectedMinutes: 1,
		DefinitionSchemaVersion: "v1", Questions: []qmodel.QuestionDefinition{{
			Code: "synthetic_process_confirmation", Type: qmodel.QuestionTypeSingleChoice, Title: "是否继续合成验证？",
			Required: true, Sort: 1, ValidationSchemaVersion: "v1", Validation: json.RawMessage(jsonBytes(map[string]any{})),
			Options: []qmodel.OptionDefinition{{Code: "CONTINUE_WITHOUT_ATTENTION", Label: "不生成关注", Sort: 1}, {Code: "CONTINUE_WITH_ATTENTION", Label: "生成关注", Sort: 2}},
		}},
	}
	versionHash, err := qmodel.HashDefinition(versionDefinition)
	if err != nil {
		t.Fatal(err)
	}
	version := qmodel.QuestionnaireVersion{
		Code: versionDefinition.Code, Version: versionDefinition.Version, Title: versionDefinition.Title, Purpose: versionDefinition.Purpose,
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true,
		ReviewType: qmodel.ReviewTypeEngineering, ReviewedBy: 9203, ReviewedAt: &published, PublishedAt: &published,
		ExpectedMinutes: 1, DefinitionSchemaVersion: "v1", DefinitionHash: versionHash, RowVersion: 1,
	}
	if err = db.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	question := qmodel.QuestionnaireQuestion{
		QuestionnaireVersionID: version.ID, Code: "synthetic_process_confirmation", Type: qmodel.QuestionTypeSingleChoice,
		Title: "是否继续合成验证？", Required: true, Sort: 1, ValidationSchemaVersion: "v1", ValidationJSON: jsonBytes(map[string]any{}),
	}
	if err = db.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	options := []qmodel.QuestionnaireOption{
		{QuestionID: question.ID, Code: "CONTINUE_WITHOUT_ATTENTION", Label: "不生成关注", Sort: 1},
		{QuestionID: question.ID, Code: "CONTINUE_WITH_ATTENTION", Label: "生成关注", Sort: 2},
	}
	if err = db.Create(&options).Error; err != nil {
		t.Fatal(err)
	}
	rule := seedRule(t, db, version.ID, "SYN-RULE-ONE", ruleStatus)
	return serviceFixture{version: version, question: question, rule: rule}
}

func seedRule(t *testing.T, db *gorm.DB, versionID uint, code, status string) qmodel.QuestionnaireRuleVersion {
	t.Helper()
	published := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.UTC)
	condition := jsonBytes(map[string]any{"questionCode": "synthetic_process_confirmation", "operator": qmodel.RuleOperatorEquals, "value": "CONTINUE_WITH_ATTENTION"})
	recipients := jsonBytes([]string{"CLINICIAN"})
	definition := qmodel.RuleDefinition{
		QuestionnaireVersionID: versionID, Code: code, Version: "1.0.0", Title: "合成关注规则",
		UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		ConditionSchemaVersion: "v1", Condition: json.RawMessage(condition), AttentionLevel: "SYNTHETIC_ATTENTION",
		ReasonSnapshot: "合成规则命中，不表达医疗含义。", Recipients: json.RawMessage(recipients),
		DedupKeyTemplate: "submission:{submissionId}:rule:" + code,
	}
	hash, err := qmodel.HashDefinition(definition)
	if err != nil {
		t.Fatal(err)
	}
	rule := qmodel.QuestionnaireRuleVersion{
		QuestionnaireVersionID: versionID, Code: code, Version: "1.0.0", Title: definition.Title,
		Status: status, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ReviewType: qmodel.ReviewTypeEngineering,
		ReviewedBy: 9203, ReviewedAt: &published, ReviewNote: "合成工程复核", ConditionSchemaVersion: "v1",
		ConditionJSON: datatypes.JSON(condition), AttentionLevel: definition.AttentionLevel, ReasonSnapshot: definition.ReasonSnapshot,
		RecipientsJSON: datatypes.JSON(recipients), DedupKeyTemplate: definition.DedupKeyTemplate,
		PublishedAt: &published, DefinitionHash: hash, RowVersion: 1,
	}
	if err = db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	return rule
}

func submissionContext() context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: 9202, AuthorityID: 9802, DeptID: 9101, DeptIDs: []uint{9101}, Scope: datascope.ScopeDept,
	})
}

func submissionCommand(key, answer string) RecordSubmissionCommand {
	return RecordSubmissionCommand{
		IdempotencyKey: key, Source: qmodel.SubmissionSourceStaffAssisted, ActorKind: qmodel.ActorKindStaff, ActorID: 9202,
		SourceReason: "合成代填验证", ConfirmationMethod: "SYNTHETIC_CONFIRMATION",
		Answers: map[string]any{"synthetic_process_confirmation": answer}, CorrelationID: "synthetic-correlation",
	}
}

func jsonBytes(value any) datatypes.JSON {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(payload)
}

func domainCode(err error) int {
	var domainErr *qmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

func assertCount(t *testing.T, db *gorm.DB, model any, where string, arg any, want int64) {
	t.Helper()
	var got int64
	query := db.Model(model)
	if where != "1 = 1" {
		query = query.Where(where, arg)
	}
	if err := query.Count(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%T count=%d, want %d", model, got, want)
	}
}
