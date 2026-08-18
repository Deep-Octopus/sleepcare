package questionnaire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FrozenTaskBinding struct {
	TaskID                 uint64 `json:"taskId"`
	CareClientID           uint   `json:"careClientId"`
	QuestionnaireVersionID uint   `json:"questionnaireVersionId"`
	RuleVersionIDs         []uint `json:"ruleVersionIds"`
	DeptID                 uint   `json:"deptId"`
	Synthetic              bool   `json:"synthetic"`
}

type RecordSubmissionCommand struct {
	IdempotencyKey     string         `json:"-"`
	Source             string         `json:"source"`
	ActorKind          string         `json:"actorKind"`
	ActorID            uint           `json:"actorId"`
	SourceReason       string         `json:"sourceReason"`
	ConfirmationMethod string         `json:"confirmationMethod"`
	Answers            map[string]any `json:"answers"`
	ClientOccurredAt   *time.Time     `json:"clientOccurredAt"`
	CorrelationID      string         `json:"correlationId"`
	CausationID        string         `json:"causationId"`
}

type SubmissionResult struct {
	SubmissionID          uint   `json:"submissionId"`
	AnswerRevisionID      uint   `json:"answerRevisionId"`
	RevisionNo            uint   `json:"revisionNo"`
	RuleHitIDs            []uint `json:"ruleHitIds"`
	RuleExecutionDisabled bool   `json:"ruleExecutionDisabled"`
}

type AppendAnswerRevisionCommand struct {
	IdempotencyKey     string         `json:"-"`
	SubmissionID       uint           `json:"submissionId"`
	ExpectedRevisionNo uint           `json:"expectedRevisionNo"`
	Answers            map[string]any `json:"answers"`
	Reason             string         `json:"reason"`
	ActorKind          string         `json:"actorKind"`
	ActorID            uint           `json:"actorId"`
}

type AppendAnswerRevisionResult struct {
	SubmissionID     uint `json:"submissionId"`
	AnswerRevisionID uint `json:"answerRevisionId"`
	RevisionNo       uint `json:"revisionNo"`
}

type ruleCondition struct {
	QuestionCode string `json:"questionCode"`
	Operator     string `json:"operator"`
	Value        any    `json:"value"`
}

func (s *QuestionnaireService) RecordSubmission(ctx context.Context, binding FrozenTaskBinding, command RecordSubmissionCommand) (SubmissionResult, error) {
	if err := validateSubmissionCommand(binding, command); err != nil {
		return SubmissionResult{}, err
	}
	commandCtx := withSubmissionDepartment(ctx, binding.DeptID, command.ActorID)
	operation := fmt.Sprintf("RECORD_SUBMISSION:%d", binding.TaskID)
	request := struct {
		Binding FrozenTaskBinding       `json:"binding"`
		Command RecordSubmissionCommand `json:"command"`
	}{binding, command}
	return runIdempotent(s, commandCtx, operation, command.ActorID, command.IdempotencyKey, request, func(tx *gorm.DB) (SubmissionResult, error) {
		version, questions, options, rules, err := loadDefinition(commandCtx, tx, binding.QuestionnaireVersionID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmissionResult{}, qmodel.NewDomainError(qmodel.CodeResourceNotFound, "任务绑定的问卷版本不存在")
		}
		if err != nil {
			return SubmissionResult{}, err
		}
		if err = validateQuestionnaireGate(version, binding, s.syntheticFixturesEnabled()); err != nil {
			return SubmissionResult{}, err
		}
		if err = verifyDefinitionHash(version, questions, options); err != nil {
			return SubmissionResult{}, err
		}
		boundRules, err := selectBoundRules(rules, binding.RuleVersionIDs)
		if err != nil {
			return SubmissionResult{}, err
		}
		normalized, answersJSON, err := canonicalAnswers(command.Answers)
		if err != nil {
			return SubmissionResult{}, err
		}
		if err = validateAnswers(questions, options, normalized); err != nil {
			return SubmissionResult{}, err
		}
		boundJSON, err := json.Marshal(binding.RuleVersionIDs)
		if err != nil {
			return SubmissionResult{}, err
		}
		requestHash, err := hashRequest(request)
		if err != nil {
			return SubmissionResult{}, err
		}
		now := s.now()
		submission := qmodel.QuestionnaireSubmission{
			TaskID: binding.TaskID, CareClientID: binding.CareClientID, QuestionnaireVersionID: binding.QuestionnaireVersionID,
			BoundRuleVersionIDsJSON: datatypes.JSON(boundJSON), Source: command.Source, ActorKind: command.ActorKind,
			ActorID: command.ActorID, SourceReason: strings.TrimSpace(command.SourceReason),
			ConfirmationMethod: strings.TrimSpace(command.ConfirmationMethod), RequestHash: requestHash,
			SubmittedAt: now, ClientOccurredAt: command.ClientOccurredAt, CurrentRevisionNo: 1, Synthetic: true,
		}
		if err = tx.Create(&submission).Error; err != nil {
			if duplicateError(err) {
				return SubmissionResult{}, qmodel.NewDomainError(qmodel.CodeIdempotencyConflict, "该任务已存在答卷")
			}
			return SubmissionResult{}, err
		}
		revision := qmodel.QuestionnaireAnswerRevision{
			SubmissionID: submission.ID, RevisionNo: 1, AnswersJSON: datatypes.JSON(answersJSON), Reason: "首次提交",
			ActorKind: command.ActorKind, ActorID: command.ActorID, OccurredAt: now, Synthetic: true,
		}
		if err = tx.Create(&revision).Error; err != nil {
			return SubmissionResult{}, err
		}
		if err = appendOutbox(tx, now, qmodel.EventTaskAnswerSubmitted, "QuestionnaireSubmission", submission.ID, map[string]any{
			"taskId": binding.TaskID, "careClientId": binding.CareClientID, "questionnaireVersionId": binding.QuestionnaireVersionID,
			"submissionId": submission.ID, "answerRevisionId": revision.ID, "revisionNo": revision.RevisionNo,
			"source": command.Source, "synthetic": true,
		}, command.CorrelationID, command.CausationID); err != nil {
			return SubmissionResult{}, err
		}
		hitIDs, executable, err := s.evaluateBoundRules(tx, submission, revision, normalized, boundRules, command.CorrelationID)
		if err != nil {
			return SubmissionResult{}, err
		}
		return SubmissionResult{
			SubmissionID: submission.ID, AnswerRevisionID: revision.ID, RevisionNo: 1,
			RuleHitIDs: hitIDs, RuleExecutionDisabled: !executable,
		}, nil
	})
}

func (s *QuestionnaireService) AppendAnswerRevision(ctx context.Context, command AppendAnswerRevisionCommand) (AppendAnswerRevisionResult, error) {
	if command.SubmissionID == 0 || command.ExpectedRevisionNo == 0 || command.ActorID == 0 || strings.TrimSpace(command.Reason) == "" {
		return AppendAnswerRevisionResult{}, qmodel.NewDomainError(qmodel.CodeInvalidArgument, "答卷、预期修订号、修订原因和操作者必填")
	}
	if command.ActorKind != qmodel.ActorKindClient && command.ActorKind != qmodel.ActorKindStaff {
		return AppendAnswerRevisionResult{}, qmodel.NewDomainError(qmodel.CodeInvalidArgument, "操作者类型无效")
	}
	var current qmodel.QuestionnaireSubmission
	if err := s.db().WithContext(ctx).Where("id = ?", command.SubmissionID).First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AppendAnswerRevisionResult{}, qmodel.NewDomainError(qmodel.CodeResourceNotFound, "答卷不存在")
		}
		return AppendAnswerRevisionResult{}, err
	}
	commandCtx := withSubmissionDepartment(ctx, current.DeptId, command.ActorID)
	operation := fmt.Sprintf("APPEND_ANSWER_REVISION:%d", command.SubmissionID)
	return runIdempotent(s, commandCtx, operation, command.ActorID, command.IdempotencyKey, command, func(tx *gorm.DB) (AppendAnswerRevisionResult, error) {
		var submission qmodel.QuestionnaireSubmission
		err := locking(tx).Where("id = ?", command.SubmissionID).First(&submission).Error
		if err != nil {
			return AppendAnswerRevisionResult{}, err
		}
		if submission.CurrentRevisionNo != command.ExpectedRevisionNo {
			return AppendAnswerRevisionResult{}, qmodel.NewDomainError(qmodel.CodeVersionConflict, "答卷修订号已变化")
		}
		version, questions, options, _, err := loadDefinition(commandCtx, tx, submission.QuestionnaireVersionID)
		if err != nil {
			return AppendAnswerRevisionResult{}, err
		}
		if err = verifyDefinitionHash(version, questions, options); err != nil {
			return AppendAnswerRevisionResult{}, err
		}
		normalized, answersJSON, err := canonicalAnswers(command.Answers)
		if err != nil {
			return AppendAnswerRevisionResult{}, err
		}
		if err = validateAnswers(questions, options, normalized); err != nil {
			return AppendAnswerRevisionResult{}, err
		}
		next := submission.CurrentRevisionNo + 1
		revision := qmodel.QuestionnaireAnswerRevision{
			SubmissionID: submission.ID, RevisionNo: next, AnswersJSON: datatypes.JSON(answersJSON),
			Reason: strings.TrimSpace(command.Reason), ActorKind: command.ActorKind, ActorID: command.ActorID,
			OccurredAt: s.now(), Synthetic: submission.Synthetic,
		}
		if err = tx.Create(&revision).Error; err != nil {
			return AppendAnswerRevisionResult{}, err
		}
		result := tx.Model(&qmodel.QuestionnaireSubmission{}).
			Where("id = ? AND current_revision_no = ?", submission.ID, command.ExpectedRevisionNo).
			Update("current_revision_no", next)
		if result.Error != nil {
			return AppendAnswerRevisionResult{}, result.Error
		}
		if result.RowsAffected != 1 {
			return AppendAnswerRevisionResult{}, qmodel.NewDomainError(qmodel.CodeVersionConflict, "答卷修订号已变化")
		}
		return AppendAnswerRevisionResult{SubmissionID: submission.ID, AnswerRevisionID: revision.ID, RevisionNo: next}, nil
	})
}

func (s *QuestionnaireService) evaluateBoundRules(tx *gorm.DB, submission qmodel.QuestionnaireSubmission, revision qmodel.QuestionnaireAnswerRevision, answers map[string]any, rules []qmodel.QuestionnaireRuleVersion, correlationID string) ([]uint, bool, error) {
	hitIDs := []uint{}
	executable := false
	for _, rule := range rules {
		allowed, err := ruleExecutable(rule, s.syntheticFixturesEnabled(), submission.Synthetic)
		if err != nil {
			return nil, false, err
		}
		if !allowed {
			continue
		}
		executable = true
		if err = verifyRuleHash(rule); err != nil {
			return nil, false, err
		}
		matched, err := ruleMatches(rule, answers)
		if err != nil {
			return nil, false, err
		}
		if !matched {
			continue
		}
		dedupKey := strings.ReplaceAll(rule.DedupKeyTemplate, "{submissionId}", strconv.FormatUint(uint64(submission.ID), 10))
		dedupKey = strings.ReplaceAll(dedupKey, "{ruleVersionId}", strconv.FormatUint(uint64(rule.ID), 10))
		hit := qmodel.QuestionnaireRuleHit{
			SubmissionID: submission.ID, AnswerRevisionID: revision.ID, RuleVersionID: rule.ID,
			ConditionSnapshotJSON: rule.ConditionJSON, AttentionLevel: rule.AttentionLevel,
			ReasonSnapshot: rule.ReasonSnapshot, RecipientsJSON: rule.RecipientsJSON, DedupKey: dedupKey,
			OccurredAt: s.now(), Synthetic: submission.Synthetic,
		}
		if err = tx.Create(&hit).Error; err != nil {
			return nil, false, err
		}
		hitIDs = append(hitIDs, hit.ID)
		if err = appendOutbox(tx, hit.OccurredAt, qmodel.EventRuleHitRecorded, "QuestionnaireSubmission", submission.ID, map[string]any{
			"submissionId": submission.ID, "answerRevisionId": revision.ID, "ruleHitId": hit.ID,
			"ruleVersionId": rule.ID, "attentionLevel": rule.AttentionLevel, "dedupKey": dedupKey,
			"synthetic": submission.Synthetic,
		}, correlationID, strconv.FormatUint(uint64(revision.ID), 10)); err != nil {
			return nil, false, err
		}
	}
	return hitIDs, executable, nil
}

func validateSubmissionCommand(binding FrozenTaskBinding, command RecordSubmissionCommand) error {
	if binding.TaskID == 0 || binding.CareClientID == 0 || binding.QuestionnaireVersionID == 0 || binding.DeptID == 0 {
		return qmodel.NewDomainError(qmodel.CodeInvalidArgument, "冻结任务绑定缺少必要标识")
	}
	if !binding.Synthetic {
		return qmodel.NewDomainError(qmodel.CodeOperationNotAllowed, "P1-03 仅允许测试任务绑定")
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" || command.ActorID == 0 {
		return qmodel.NewDomainError(qmodel.CodeInvalidArgument, "幂等键和操作者必填")
	}
	switch command.Source {
	case qmodel.SubmissionSourceClientSelf:
		if command.ActorKind != qmodel.ActorKindClient {
			return qmodel.NewDomainError(qmodel.CodeInvalidArgument, "自主提交必须使用康养用户操作者类型")
		}
	case qmodel.SubmissionSourceStaffAssisted:
		if command.ActorKind != qmodel.ActorKindStaff || strings.TrimSpace(command.SourceReason) == "" || strings.TrimSpace(command.ConfirmationMethod) == "" {
			return qmodel.NewDomainError(qmodel.CodeInvalidArgument, "工作人员代填必须记录原因和确认方式")
		}
	default:
		return qmodel.NewDomainError(qmodel.CodeInvalidArgument, "答卷来源无效")
	}
	return nil
}

func validateQuestionnaireGate(version qmodel.QuestionnaireVersion, binding FrozenTaskBinding, fixturesEnabled bool) error {
	if version.Status == qmodel.LifecycleDisabled {
		return qmodel.NewDomainError(qmodel.CodeContentDisabled, "任务绑定的问卷版本已禁用")
	}
	if version.Status != qmodel.LifecyclePublished || version.PublishedAt == nil || version.ReviewedAt == nil {
		return qmodel.NewDomainError(qmodel.CodeContentNotPublished, "任务绑定的问卷版本未完成发布审批")
	}
	if version.UsageScope == qmodel.UsageScopeTestOnly {
		if !fixturesEnabled || !version.Synthetic || version.ProductionEnabled || version.ReviewType != qmodel.ReviewTypeEngineering || !binding.Synthetic {
			return qmodel.NewDomainError(qmodel.CodeContentDisabled, "测试问卷版本不满足测试环境门禁")
		}
		return nil
	}
	if version.UsageScope == qmodel.UsageScopeFormal {
		if version.Synthetic || !version.ProductionEnabled || version.ReviewType != qmodel.ReviewTypeFormal {
			return qmodel.NewDomainError(qmodel.CodeContentDisabled, "正式问卷版本不满足生产启用和正式审批门禁")
		}
		return nil
	}
	return qmodel.NewDomainError(qmodel.CodeContentDisabled, "问卷使用范围无效")
}

func ruleExecutable(rule qmodel.QuestionnaireRuleVersion, fixturesEnabled, submissionSynthetic bool) (bool, error) {
	if rule.Status != qmodel.LifecyclePublished || rule.PublishedAt == nil || rule.ReviewedAt == nil {
		return false, nil
	}
	if rule.UsageScope == qmodel.UsageScopeTestOnly {
		return submissionSynthetic && fixturesEnabled && rule.Synthetic && !rule.ProductionEnabled && rule.ReviewType == qmodel.ReviewTypeEngineering, nil
	}
	if rule.UsageScope == qmodel.UsageScopeFormal {
		return !submissionSynthetic && !rule.Synthetic && rule.ProductionEnabled && rule.ReviewType == qmodel.ReviewTypeFormal, nil
	}
	return false, qmodel.NewDomainError(qmodel.CodeContentDisabled, "关注规则使用范围无效")
}

func selectBoundRules(rules []qmodel.QuestionnaireRuleVersion, ids []uint) ([]qmodel.QuestionnaireRuleVersion, error) {
	byID := make(map[uint]qmodel.QuestionnaireRuleVersion, len(rules))
	for _, rule := range rules {
		byID[rule.ID] = rule
	}
	selected := make([]qmodel.QuestionnaireRuleVersion, 0, len(ids))
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 {
			return nil, qmodel.NewDomainError(qmodel.CodeInvalidArgument, "冻结规则版本标识无效")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, qmodel.NewDomainError(qmodel.CodeInvalidArgument, "冻结规则版本列表包含重复项")
		}
		rule, ok := byID[id]
		if !ok {
			return nil, qmodel.NewDomainError(qmodel.CodeResourceNotFound, "冻结规则版本不存在或不属于绑定问卷")
		}
		seen[id] = struct{}{}
		selected = append(selected, rule)
	}
	return selected, nil
}

func ruleMatches(rule qmodel.QuestionnaireRuleVersion, answers map[string]any) (bool, error) {
	condition := ruleCondition{}
	decoder := json.NewDecoder(strings.NewReader(string(rule.ConditionJSON)))
	decoder.UseNumber()
	if err := decoder.Decode(&condition); err != nil || strings.TrimSpace(condition.QuestionCode) == "" {
		return false, qmodel.NewDomainError(qmodel.CodeOperationNotAllowed, "关注规则条件定义无效")
	}
	answer, ok := answers[condition.QuestionCode]
	if !ok {
		return false, nil
	}
	switch condition.Operator {
	case qmodel.RuleOperatorEquals:
		return jsonEqual(answer, condition.Value), nil
	case qmodel.RuleOperatorContains:
		items, ok := answer.([]any)
		if !ok {
			return false, nil
		}
		for _, item := range items {
			if jsonEqual(item, condition.Value) {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, qmodel.NewDomainError(qmodel.CodeOperationNotAllowed, "关注规则操作符不受支持")
	}
}

func runIdempotent[T any](s *QuestionnaireService, ctx context.Context, operation string, actorID uint, key string, request any, fn func(*gorm.DB) (T, error)) (T, error) {
	var zero T
	key = strings.TrimSpace(key)
	if key == "" {
		return zero, qmodel.NewDomainError(qmodel.CodeInvalidArgument, "Idempotency-Key 必填")
	}
	hash, err := hashRequest(request)
	if err != nil {
		return zero, err
	}
	err = s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var receipt qmodel.QuestionnaireCommandReceipt
		receiptErr := tx.Where("actor_id = ? AND operation = ? AND idempotency_key = ?", actorID, operation, key).First(&receipt).Error
		if receiptErr == nil {
			if receipt.RequestHash != hash {
				return qmodel.NewDomainError(qmodel.CodeIdempotencyConflict, "幂等键已用于不同请求")
			}
			if err := json.Unmarshal([]byte(receipt.ResultJSON), &zero); err != nil {
				return err
			}
			return nil
		}
		if !errors.Is(receiptErr, gorm.ErrRecordNotFound) {
			return receiptErr
		}
		result, err := fn(tx)
		if err != nil {
			return err
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return err
		}
		receipt = qmodel.QuestionnaireCommandReceipt{Operation: operation, ActorID: actorID, IdempotencyKey: key, RequestHash: hash, ResultJSON: string(resultJSON)}
		if err = tx.Create(&receipt).Error; err != nil {
			if duplicateError(err) {
				return qmodel.NewDomainError(qmodel.CodeIdempotencyConflict, "幂等请求发生并发冲突，请重试")
			}
			return err
		}
		zero = result
		return nil
	})
	return zero, err
}

func appendOutbox(tx *gorm.DB, occurredAt time.Time, eventType, aggregateType string, aggregateID uint, payload any, correlationID, causationID string) error {
	return platformoutbox.Append(tx, platformoutbox.AppendInput{
		EventType: eventType, AggregateType: aggregateType, AggregateID: aggregateID, Payload: payload,
		OccurredAt: occurredAt, CorrelationID: correlationID, CausationID: causationID, Synthetic: true,
	})
}

func hashRequest(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func withSubmissionDepartment(ctx context.Context, departmentID, actorID uint) context.Context {
	if identity, ok := datascope.FromContext(ctx); ok && identity != nil {
		copyIdentity := *identity
		copyIdentity.DeptID = departmentID
		if copyIdentity.UserID == 0 {
			copyIdentity.UserID = actorID
		}
		return datascope.WithIdentity(ctx, &copyIdentity)
	}
	return datascope.WithIdentity(ctx, &datascope.Identity{UserID: actorID, DeptID: departmentID, Scope: datascope.ScopeDept, DeptIDs: []uint{departmentID}})
}

func locking(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}

func duplicateError(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique")
}
