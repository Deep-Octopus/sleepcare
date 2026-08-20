package clientaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	clientreq "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/request"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	caseworkservice "github.com/flipped-aurora/gin-vue-admin/server/service/casework"
	questionnaireservice "github.com/flipped-aurora/gin-vue-admin/server/service/questionnaire"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type taskEnvelope struct {
	Task pathmodel.TaskInstance
	Plan pathmodel.PlanInstance
}

func (s *ClientAccessService) ListTasks(ctx context.Context, req clientreq.TaskSearch) ([]clientres.TaskSummary, int64, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	taskIDs := identity.AllowedTaskIDs
	if identity.AuthType == clientmodel.SessionAuthAccount {
		taskIDs = []uint{}
		if err = s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
			Where("care_client_id = ? AND execution_role = ? AND synthetic = ?", identity.CareClientID, pathmodel.ExecutionRoleCareClient, true).
			Order("id ASC").Pluck("id", &taskIDs).Error; err != nil {
			return nil, 0, err
		}
	}
	for _, taskID := range taskIDs {
		if err = s.openDueTask(ctx, taskID); err != nil {
			return nil, 0, err
		}
	}
	query := s.db().WithContext(ctx).Model(&pathmodel.TaskInstance{}).
		Where("care_client_id = ? AND execution_role = ? AND synthetic = ?", identity.CareClientID, pathmodel.ExecutionRoleCareClient, true)
	if identity.AuthType != clientmodel.SessionAuthAccount {
		query = query.Where("id IN ?", taskIDs)
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
	var tasks []pathmodel.TaskInstance
	if err = query.Order("open_at ASC, sort ASC, id ASC").Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	planIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		planIDs = append(planIDs, task.PlanInstanceID)
	}
	var plans []pathmodel.PlanInstance
	if len(planIDs) > 0 {
		if err = s.db().WithContext(ctx).Where("id IN ? AND care_client_id = ? AND synthetic = ?", planIDs, identity.CareClientID, true).Find(&plans).Error; err != nil {
			return nil, 0, err
		}
	}
	planByID := make(map[uint]pathmodel.PlanInstance, len(plans))
	for _, plan := range plans {
		planByID[plan.ID] = plan
	}
	items := make([]clientres.TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, s.taskSummary(task, planByID[task.PlanInstanceID]))
	}
	return items, total, nil
}

func (s *ClientAccessService) GetTask(ctx context.Context, taskID uint) (clientres.TaskDetail, error) {
	if err := s.openDueTask(ctx, taskID); err != nil {
		return clientres.TaskDetail{}, err
	}
	envelope, err := s.loadTask(ctx, s.db().WithContext(ctx), taskID, false)
	if err != nil {
		return clientres.TaskDetail{}, err
	}
	if err = s.ensureTaskDetailAvailable(envelope); err != nil {
		return clientres.TaskDetail{}, err
	}
	facts, err := taskFacts(ctx, s.db().WithContext(ctx), taskID)
	if err != nil {
		return clientres.TaskDetail{}, err
	}
	summary := s.taskSummary(envelope.Task, envelope.Plan)
	return clientres.TaskDetail{
		TaskSummary:  summary,
		Opened:       facts[pathmodel.EventClientTaskOpened],
		Consented:    facts[pathmodel.EventClientTaskConsented],
		Started:      facts[pathmodel.EventTaskAnswerStarted],
		CanSaveDraft: summary.Accessible && envelope.Task.ExecutionStatus == pathmodel.ExecutionInProgress && facts[pathmodel.EventTaskAnswerStarted],
		CanSubmit:    summary.Accessible && envelope.Task.ExecutionStatus == pathmodel.ExecutionInProgress && facts[pathmodel.EventClientTaskConsented] && facts[pathmodel.EventTaskAnswerStarted],
	}, nil
}

func (s *ClientAccessService) GetQuestionnaire(ctx context.Context, taskID uint) (clientres.Questionnaire, error) {
	if err := s.openDueTask(ctx, taskID); err != nil {
		return clientres.Questionnaire{}, err
	}
	envelope, err := s.loadTask(ctx, s.db().WithContext(ctx), taskID, false)
	if err != nil {
		return clientres.Questionnaire{}, err
	}
	if err = s.ensureTaskActionable(envelope); err != nil {
		return clientres.Questionnaire{}, err
	}
	facts, err := taskFacts(ctx, s.db().WithContext(ctx), taskID)
	if err != nil {
		return clientres.Questionnaire{}, err
	}
	if !facts[pathmodel.EventClientTaskConsented] || !facts[pathmodel.EventTaskAnswerStarted] || envelope.Task.ExecutionStatus != pathmodel.ExecutionInProgress {
		return clientres.Questionnaire{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "请先确认任务说明并开始填写")
	}
	if envelope.Task.QuestionnaireVersionID == nil || *envelope.Task.QuestionnaireVersionID == 0 {
		return clientres.Questionnaire{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "当前任务没有可填写内容")
	}
	ruleIDs, err := decodeRuleIDs(envelope.Task.BoundRuleVersionIDsJSON)
	if err != nil {
		return clientres.Questionnaire{}, err
	}
	binding := questionnaireservice.FrozenTaskBinding{
		TaskID: uint64(envelope.Task.ID), CareClientID: envelope.Task.CareClientID,
		QuestionnaireVersionID: *envelope.Task.QuestionnaireVersionID, RuleVersionIDs: ruleIDs,
		DeptID: envelope.Task.DeptId, Synthetic: envelope.Task.Synthetic,
	}
	fixturesEnabled := s.syntheticFixturesEnabled()
	questionnaireService := questionnaireservice.QuestionnaireService{DB: s.db(), SyntheticFixturesEnabled: &fixturesEnabled}
	if err = questionnaireService.ValidateFrozenBinding(ctx, binding.QuestionnaireVersionID, binding.RuleVersionIDs, binding.Synthetic); err != nil {
		return clientres.Questionnaire{}, normalizeQuestionnaireError(err)
	}
	var version qmodel.QuestionnaireVersion
	if err = s.db().WithContext(ctx).Where("id = ?", binding.QuestionnaireVersionID).First(&version).Error; err != nil {
		return clientres.Questionnaire{}, err
	}
	var questions []qmodel.QuestionnaireQuestion
	if err = s.db().WithContext(ctx).Where("questionnaire_version_id = ?", version.ID).Order("sort ASC, id ASC").Find(&questions).Error; err != nil {
		return clientres.Questionnaire{}, err
	}
	questionIDs := make([]uint, 0, len(questions))
	for _, question := range questions {
		questionIDs = append(questionIDs, question.ID)
	}
	var options []qmodel.QuestionnaireOption
	if len(questionIDs) > 0 {
		if err = s.db().WithContext(ctx).Where("question_id IN ?", questionIDs).Order("sort ASC, id ASC").Find(&options).Error; err != nil {
			return clientres.Questionnaire{}, err
		}
	}
	optionsByQuestion := make(map[uint][]clientres.QuestionnaireOption)
	for _, option := range options {
		optionsByQuestion[option.QuestionID] = append(optionsByQuestion[option.QuestionID], clientres.QuestionnaireOption{
			Code: option.Code, Label: option.Label, Order: option.Sort,
		})
	}
	items := make([]clientres.QuestionnaireQuestion, 0, len(questions))
	for _, question := range questions {
		validation := map[string]any{}
		if len(question.ValidationJSON) > 0 {
			if err = json.Unmarshal(question.ValidationJSON, &validation); err != nil {
				return clientres.Questionnaire{}, fmt.Errorf("decode validation for question %s: %w", question.Code, err)
			}
		}
		questionOptions := optionsByQuestion[question.ID]
		if questionOptions == nil {
			questionOptions = []clientres.QuestionnaireOption{}
		}
		items = append(items, clientres.QuestionnaireQuestion{
			Code: question.Code, Type: question.Type, Title: question.Title, Required: question.Required,
			Order: question.Sort, Validation: validation, Options: questionOptions,
		})
	}
	var draftResponse *clientres.Draft
	var draft qmodel.QuestionnaireTaskDraft
	draftErr := s.db().WithContext(ctx).Where("task_id = ? AND care_client_id = ? AND consumed_at IS NULL", taskID, envelope.Task.CareClientID).First(&draft).Error
	if draftErr == nil {
		answers := map[string]any{}
		if err = json.Unmarshal(draft.AnswersJSON, &answers); err != nil {
			return clientres.Questionnaire{}, err
		}
		draftResponse = &clientres.Draft{Version: draft.Version, Answers: answers, SavedAt: draft.SavedAt}
	} else if !errors.Is(draftErr, gorm.ErrRecordNotFound) {
		return clientres.Questionnaire{}, draftErr
	}
	return clientres.Questionnaire{
		ID: version.ID, Title: version.Title, Purpose: version.Purpose, ExpectedMinutes: version.ExpectedMinutes,
		TaskVersion: envelope.Task.Version, Questions: items, Draft: draftResponse,
	}, nil
}

func (s *ClientAccessService) RecordInteraction(ctx context.Context, taskID uint, key string, req clientreq.RecordInteraction) (clientres.InteractionResult, error) {
	if !clientmodel.IsInteractionType(req.InteractionType) {
		return clientres.InteractionResult{}, clientmodel.NewDomainError(clientmodel.CodeInvalidArgument, "交互类型无效")
	}
	return runIdempotent(s, ctx, operation("INTERACTION_"+req.InteractionType, taskID), key, req, func(tx *gorm.DB) (clientres.InteractionResult, error) {
		envelope, err := s.loadTask(ctx, tx, taskID, true)
		if err != nil {
			return clientres.InteractionResult{}, err
		}
		eventType := interactionEventType(req.InteractionType)
		exists, err := eventExists(tx, taskID, eventType)
		if err != nil {
			return clientres.InteractionResult{}, err
		}
		if exists {
			return clientres.InteractionResult{
				TaskID: taskID, InteractionType: req.InteractionType,
				ExecutionStatus: envelope.Task.ExecutionStatus, TaskVersion: envelope.Task.Version,
			}, nil
		}
		if err = s.ensureTaskActionable(envelope); err != nil {
			return clientres.InteractionResult{}, err
		}
		if envelope.Task.Version != req.ExpectedVersion {
			return clientres.InteractionResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "任务版本已变化")
		}
		if err = validateInteractionSequence(tx, taskID, req.InteractionType); err != nil {
			return clientres.InteractionResult{}, err
		}
		fromStatus := envelope.Task.ExecutionStatus
		toStatus := fromStatus
		if req.InteractionType == clientmodel.InteractionStarted {
			if fromStatus != pathmodel.ExecutionOpen {
				return clientres.InteractionResult{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "任务当前状态不能开始填写")
			}
			toStatus = pathmodel.ExecutionInProgress
		}
		updates := map[string]any{"version": gorm.Expr("version + 1")}
		if toStatus != fromStatus {
			updates["execution_status"] = toStatus
		}
		result := tx.Model(&pathmodel.TaskInstance{}).
			Where("id = ? AND care_client_id = ? AND version = ?", taskID, envelope.Task.CareClientID, req.ExpectedVersion).
			Updates(updates)
		if result.Error != nil {
			return clientres.InteractionResult{}, result.Error
		}
		if result.RowsAffected != 1 {
			return clientres.InteractionResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "任务版本已变化")
		}
		identity, _ := identityFromContext(ctx)
		taskIDCopy := taskID
		event := pathmodel.CarePathEvent{
			EventID: stableFactID(taskID, eventType), EventType: eventType,
			CareClientID: envelope.Task.CareClientID, EnrollmentID: envelope.Plan.EnrollmentID,
			PlanInstanceID: envelope.Plan.ID, TaskInstanceID: &taskIDCopy,
			ActorID: identity.CareClientID, Source: pathmodel.EventSourceClient,
			FromStatus: fromStatus, ToStatus: toStatus, OccurredAt: s.now(), Synthetic: true,
			DeptId: envelope.Task.DeptId, CreatedBy: identity.CareClientID,
		}
		if err = tx.Create(&event).Error; err != nil {
			return clientres.InteractionResult{}, err
		}
		return clientres.InteractionResult{
			TaskID: taskID, InteractionType: req.InteractionType,
			ExecutionStatus: toStatus, TaskVersion: req.ExpectedVersion + 1,
		}, nil
	})
}

func (s *ClientAccessService) SaveDraft(ctx context.Context, taskID uint, key string, req clientreq.SaveDraft) (clientres.DraftResult, error) {
	return runIdempotent(s, ctx, operation("SAVE_DRAFT", taskID), key, req, func(tx *gorm.DB) (clientres.DraftResult, error) {
		envelope, err := s.loadTask(ctx, tx, taskID, true)
		if err != nil {
			return clientres.DraftResult{}, err
		}
		if err = s.ensureTaskActionable(envelope); err != nil {
			return clientres.DraftResult{}, err
		}
		if envelope.Task.ExecutionStatus != pathmodel.ExecutionInProgress {
			return clientres.DraftResult{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "请先完成任务说明确认并开始填写")
		}
		started, err := eventExists(tx, taskID, pathmodel.EventTaskAnswerStarted)
		if err != nil {
			return clientres.DraftResult{}, err
		}
		if !started {
			return clientres.DraftResult{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "任务尚未开始填写")
		}
		binding, err := taskBinding(envelope.Task)
		if err != nil {
			return clientres.DraftResult{}, err
		}
		fixturesEnabled := s.syntheticFixturesEnabled()
		questionnaireService := questionnaireservice.QuestionnaireService{DB: tx, Now: s.now, SyntheticFixturesEnabled: &fixturesEnabled}
		_, canonical, err := questionnaireService.PrepareTaskDraft(ctx, binding, req.Answers)
		if err != nil {
			return clientres.DraftResult{}, normalizeQuestionnaireError(err)
		}
		now := s.now()
		var draft qmodel.QuestionnaireTaskDraft
		draftErr := locking(tx).Where("task_id = ? AND care_client_id = ?", taskID, envelope.Task.CareClientID).First(&draft).Error
		if errors.Is(draftErr, gorm.ErrRecordNotFound) {
			if req.ExpectedVersion != 0 {
				return clientres.DraftResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "草稿版本已变化")
			}
			draft = qmodel.QuestionnaireTaskDraft{
				TaskID: uint64(taskID), CareClientID: envelope.Task.CareClientID,
				QuestionnaireVersionID: binding.QuestionnaireVersionID, AnswersJSON: datatypes.JSON(canonical),
				Version: 1, SavedAt: now, Synthetic: true, DeptId: envelope.Task.DeptId,
			}
			if err = tx.Create(&draft).Error; err != nil {
				return clientres.DraftResult{}, err
			}
			return clientres.DraftResult{TaskID: taskID, Version: 1, SavedAt: now}, nil
		}
		if draftErr != nil {
			return clientres.DraftResult{}, draftErr
		}
		if draft.ConsumedAt != nil {
			return clientres.DraftResult{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "任务已提交，草稿不可修改")
		}
		if draft.Version != req.ExpectedVersion {
			return clientres.DraftResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "草稿版本已变化")
		}
		result := tx.Model(&qmodel.QuestionnaireTaskDraft{}).
			Where("id = ? AND version = ? AND consumed_at IS NULL", draft.ID, req.ExpectedVersion).
			Updates(map[string]any{"answers_json": datatypes.JSON(canonical), "saved_at": now, "version": gorm.Expr("version + 1")})
		if result.Error != nil {
			return clientres.DraftResult{}, result.Error
		}
		if result.RowsAffected != 1 {
			return clientres.DraftResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "草稿版本已变化")
		}
		return clientres.DraftResult{TaskID: taskID, Version: req.ExpectedVersion + 1, SavedAt: now}, nil
	})
}

func (s *ClientAccessService) SubmitTask(ctx context.Context, taskID uint, key string, req clientreq.SubmitTask) (clientres.SubmitResult, error) {
	if req.Source != qmodel.SubmissionSourceClientSelf {
		return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeInvalidArgument, "提交来源无效")
	}
	return runIdempotent(s, ctx, operation("SUBMIT_TASK", taskID), key, req, func(tx *gorm.DB) (clientres.SubmitResult, error) {
		envelope, err := s.loadTask(ctx, tx, taskID, true)
		if err != nil {
			return clientres.SubmitResult{}, err
		}
		if envelope.Task.ExecutionStatus == pathmodel.ExecutionSubmitted {
			return existingSubmissionResult(tx, envelope.Task, req.Answers)
		}
		if err = s.ensureTaskActionable(envelope); err != nil {
			return clientres.SubmitResult{}, err
		}
		if envelope.Task.ExecutionStatus != pathmodel.ExecutionInProgress {
			return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "任务尚未开始填写")
		}
		if envelope.Task.Version != req.ExpectedTaskVersion {
			return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "任务版本已变化")
		}
		facts, err := taskFacts(ctx, tx, taskID)
		if err != nil {
			return clientres.SubmitResult{}, err
		}
		if !facts[pathmodel.EventClientTaskConsented] || !facts[pathmodel.EventTaskAnswerStarted] {
			return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "请先确认任务说明并开始填写")
		}
		binding, err := taskBinding(envelope.Task)
		if err != nil {
			return clientres.SubmitResult{}, err
		}
		identity, _ := identityFromContext(ctx)
		fixturesEnabled := s.syntheticFixturesEnabled()
		questionnaireService := questionnaireservice.QuestionnaireService{DB: tx, Now: s.now, SyntheticFixturesEnabled: &fixturesEnabled}
		submission, err := questionnaireService.RecordSubmission(ctx, binding, questionnaireservice.RecordSubmissionCommand{
			IdempotencyKey: key, Source: qmodel.SubmissionSourceClientSelf,
			ActorKind: qmodel.ActorKindClient, ActorID: identity.CareClientID,
			ConfirmationMethod: "CLIENT_WEB_CONFIRMATION", Answers: req.Answers,
			ClientOccurredAt: req.ClientOccurredAt, CorrelationID: identity.SessionID, CausationID: DigestToken(key),
		})
		if err != nil {
			return clientres.SubmitResult{}, normalizeQuestionnaireError(err)
		}
		openedCases, err := (&caseworkservice.CaseWorkService{DB: tx, Now: s.now}).OpenFromRuleHits(ctx, caseworkservice.OpenFromRuleHitsCommand{
			CareClientID:  envelope.Task.CareClientID,
			TaskID:        taskID,
			SubmissionID:  submission.SubmissionID,
			RuleHitIDs:    submission.RuleHitIDs,
			CorrelationID: identity.SessionID,
		})
		if err != nil {
			return clientres.SubmitResult{}, normalizeCaseWorkError(err)
		}
		now := s.now()
		reviewStatus := pathmodel.ReviewNotRequired
		if envelope.Task.ReviewRole != "" {
			reviewStatus = pathmodel.ReviewPending
		}
		result := tx.Model(&pathmodel.TaskInstance{}).
			Where("id = ? AND care_client_id = ? AND version = ? AND execution_status = ?", taskID, envelope.Task.CareClientID, req.ExpectedTaskVersion, pathmodel.ExecutionInProgress).
			Updates(map[string]any{
				"execution_status": pathmodel.ExecutionSubmitted, "review_status": reviewStatus,
				"submitted_at": now, "version": gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return clientres.SubmitResult{}, result.Error
		}
		if result.RowsAffected != 1 {
			return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeVersionConflict, "任务版本已变化")
		}
		taskIDCopy := taskID
		event := pathmodel.CarePathEvent{
			EventID: stableFactID(taskID, pathmodel.EventTaskAnswerSubmitted), EventType: pathmodel.EventTaskAnswerSubmitted,
			CareClientID: envelope.Task.CareClientID, EnrollmentID: envelope.Plan.EnrollmentID,
			PlanInstanceID: envelope.Plan.ID, TaskInstanceID: &taskIDCopy,
			ActorID: identity.CareClientID, Source: pathmodel.EventSourceClient,
			FromStatus: pathmodel.ExecutionInProgress, ToStatus: pathmodel.ExecutionSubmitted,
			OccurredAt: now, Synthetic: true, DeptId: envelope.Task.DeptId, CreatedBy: identity.CareClientID,
		}
		if err = tx.Create(&event).Error; err != nil {
			return clientres.SubmitResult{}, err
		}
		if err = tx.Model(&qmodel.QuestionnaireTaskDraft{}).
			Where("task_id = ? AND care_client_id = ? AND consumed_at IS NULL", taskID, envelope.Task.CareClientID).
			Updates(map[string]any{"consumed_at": now}).Error; err != nil {
			return clientres.SubmitResult{}, err
		}
		return clientres.SubmitResult{
			TaskID: taskID, SubmissionID: submission.SubmissionID,
			ExecutionStatus: pathmodel.ExecutionSubmitted, ReviewStatus: reviewStatus,
			TaskVersion: req.ExpectedTaskVersion + 1, RuleHitIDs: nonNilUint(submission.RuleHitIDs), AttentionCaseIDs: nonNilUint(openedCases.AttentionCaseIDs),
		}, nil
	})
}

func (s *ClientAccessService) loadTask(ctx context.Context, db *gorm.DB, taskID uint, lock bool) (taskEnvelope, error) {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return taskEnvelope{}, err
	}
	if taskID == 0 || !allowedTask(identity, taskID) {
		return taskEnvelope{}, scopeDenied()
	}
	query := db.WithContext(ctx)
	if lock {
		query = locking(query)
	}
	var task pathmodel.TaskInstance
	if err = query.Where("id = ? AND care_client_id = ? AND execution_role = ? AND synthetic = ?", taskID, identity.CareClientID, pathmodel.ExecutionRoleCareClient, true).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return taskEnvelope{}, scopeDenied()
		}
		return taskEnvelope{}, err
	}
	var plan pathmodel.PlanInstance
	if err = db.WithContext(ctx).Where("id = ? AND care_client_id = ? AND synthetic = ?", task.PlanInstanceID, identity.CareClientID, true).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return taskEnvelope{}, scopeDenied()
		}
		return taskEnvelope{}, err
	}
	return taskEnvelope{Task: task, Plan: plan}, nil
}

func (s *ClientAccessService) ensureTaskDetailAvailable(envelope taskEnvelope) error {
	if envelope.Task.ExecutionStatus == pathmodel.ExecutionSubmitted {
		return nil
	}
	return s.ensureTaskActionable(envelope)
}

func (s *ClientAccessService) ensureTaskActionable(envelope taskEnvelope) error {
	if err := taskActionError(envelope.Task); err != nil {
		return err
	}
	if envelope.Plan.Status != pathmodel.EnrollmentActive {
		return clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "当前计划不可填写")
	}
	switch envelope.Task.TimingStatus(s.now()) {
	case pathmodel.TimingNotOpen:
		return clientmodel.NewDomainError(clientmodel.CodeTaskNotOpen, "任务尚未开放")
	case pathmodel.TimingExpired:
		return clientmodel.NewDomainError(clientmodel.CodeTaskExpired, "任务已过期")
	}
	if envelope.Task.ExecutionStatus != pathmodel.ExecutionOpen && envelope.Task.ExecutionStatus != pathmodel.ExecutionInProgress {
		return clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "任务当前状态不可填写")
	}
	return nil
}

func (s *ClientAccessService) taskSummary(task pathmodel.TaskInstance, plan pathmodel.PlanInstance) clientres.TaskSummary {
	timing := task.TimingStatus(s.now())
	accessible := plan.Status == pathmodel.EnrollmentActive &&
		(timing == pathmodel.TimingWithinWindow || timing == pathmodel.TimingOverdue) &&
		(task.ExecutionStatus == pathmodel.ExecutionOpen || task.ExecutionStatus == pathmodel.ExecutionInProgress)
	return clientres.TaskSummary{
		ID: task.ID, DayCode: task.DayCode, Title: task.Title,
		ExecutionStatus: task.ExecutionStatus, TimingStatus: timing, ReviewStatus: task.ReviewStatus,
		OpenAt: task.OpenAt, DueAt: task.DueAt, ExpiresAt: task.ExpiresAt, SubmittedAt: task.SubmittedAt,
		Version: task.Version, Accessible: accessible, HasQuestionnaire: task.QuestionnaireVersionID != nil,
	}
}

func (s *ClientAccessService) openDueTask(ctx context.Context, taskID uint) error {
	identity, err := identityFromContext(ctx)
	if err != nil {
		return err
	}
	if !allowedTask(identity, taskID) {
		return scopeDenied()
	}
	return s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task pathmodel.TaskInstance
		if err := locking(tx).Where("id = ? AND care_client_id = ? AND synthetic = ?", taskID, identity.CareClientID, true).First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return scopeDenied()
			}
			return err
		}
		if task.ExecutionStatus != pathmodel.ExecutionScheduled || s.now().Before(task.OpenAt) {
			return nil
		}
		var plan pathmodel.PlanInstance
		if err := tx.Where("id = ? AND care_client_id = ? AND synthetic = ?", task.PlanInstanceID, identity.CareClientID, true).First(&plan).Error; err != nil {
			return err
		}
		if plan.Status != pathmodel.EnrollmentActive {
			return nil
		}
		now := s.now()
		result := tx.Model(&pathmodel.TaskInstance{}).
			Where("id = ? AND execution_status = ?", task.ID, pathmodel.ExecutionScheduled).
			Updates(map[string]any{"execution_status": pathmodel.ExecutionOpen, "opened_at": now, "version": gorm.Expr("version + 1")})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		taskIDCopy := task.ID
		if err := tx.Create(&pathmodel.CarePathEvent{
			EventID: uuid.NewString(), EventType: pathmodel.EventTaskOpened,
			CareClientID: task.CareClientID, EnrollmentID: plan.EnrollmentID, PlanInstanceID: plan.ID,
			TaskInstanceID: &taskIDCopy, ActorID: identity.CareClientID, Source: pathmodel.EventSourceSystem,
			FromStatus: pathmodel.ExecutionScheduled, ToStatus: pathmodel.ExecutionOpen,
			OccurredAt: now, Synthetic: true, DeptId: task.DeptId, CreatedBy: identity.CareClientID,
		}).Error; err != nil {
			return err
		}
		return platformoutbox.Append(tx, platformoutbox.AppendInput{
			EventType: pathmodel.EventTaskOpened, AggregateType: "CareTask", AggregateID: task.ID,
			Payload: map[string]any{
				"careClientId": task.CareClientID, "enrollmentId": plan.EnrollmentID,
				"planInstanceId": plan.ID, "taskInstanceId": task.ID, "dayCode": task.DayCode,
				"openAt": task.OpenAt, "openedAt": now, "synthetic": true,
			},
			OccurredAt: now, CausationID: fmt.Sprintf("%d:%d:%s", plan.ID, task.ID, task.OpenAt.UTC().Format(time.RFC3339Nano)),
			Synthetic: true, DeptID: task.DeptId, CreatedBy: identity.CareClientID,
		})
	})
}

func interactionEventType(interaction string) string {
	switch interaction {
	case clientmodel.InteractionOpened:
		return pathmodel.EventClientTaskOpened
	case clientmodel.InteractionConsented:
		return pathmodel.EventClientTaskConsented
	default:
		return pathmodel.EventTaskAnswerStarted
	}
}

func validateInteractionSequence(db *gorm.DB, taskID uint, interaction string) error {
	if interaction == clientmodel.InteractionOpened {
		return nil
	}
	opened, err := eventExists(db, taskID, pathmodel.EventClientTaskOpened)
	if err != nil {
		return err
	}
	if !opened {
		return clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "请先打开任务")
	}
	if interaction == clientmodel.InteractionConsented {
		return nil
	}
	consented, err := eventExists(db, taskID, pathmodel.EventClientTaskConsented)
	if err != nil {
		return err
	}
	if !consented {
		return clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "请先确认任务说明")
	}
	return nil
}

func eventExists(db *gorm.DB, taskID uint, eventType string) (bool, error) {
	var count int64
	err := db.Model(&pathmodel.CarePathEvent{}).
		Where("task_instance_id = ? AND event_type = ? AND synthetic = ?", taskID, eventType, true).
		Count(&count).Error
	return count > 0, err
}

func taskFacts(ctx context.Context, db *gorm.DB, taskID uint) (map[string]bool, error) {
	types := []string{pathmodel.EventClientTaskOpened, pathmodel.EventClientTaskConsented, pathmodel.EventTaskAnswerStarted}
	var rows []pathmodel.CarePathEvent
	if err := db.WithContext(ctx).Select("event_type").Where("task_instance_id = ? AND event_type IN ? AND synthetic = ?", taskID, types, true).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(types))
	for _, row := range rows {
		result[row.EventType] = true
	}
	return result, nil
}

func stableFactID(taskID uint, eventType string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("client-task:%d:%s", taskID, eventType))).String()
}

func taskBinding(task pathmodel.TaskInstance) (questionnaireservice.FrozenTaskBinding, error) {
	if task.QuestionnaireVersionID == nil || *task.QuestionnaireVersionID == 0 {
		return questionnaireservice.FrozenTaskBinding{}, clientmodel.NewDomainError(clientmodel.CodeOperationNotAllowed, "当前任务没有可填写内容")
	}
	ruleIDs, err := decodeRuleIDs(task.BoundRuleVersionIDsJSON)
	if err != nil {
		return questionnaireservice.FrozenTaskBinding{}, err
	}
	return questionnaireservice.FrozenTaskBinding{
		TaskID: uint64(task.ID), CareClientID: task.CareClientID,
		QuestionnaireVersionID: *task.QuestionnaireVersionID, RuleVersionIDs: ruleIDs,
		DeptID: task.DeptId, Synthetic: task.Synthetic,
	}, nil
}

func existingSubmissionResult(db *gorm.DB, task pathmodel.TaskInstance, answers map[string]any) (clientres.SubmitResult, error) {
	var submission qmodel.QuestionnaireSubmission
	if err := db.Where("task_id = ? AND care_client_id = ?", task.ID, task.CareClientID).First(&submission).Error; err != nil {
		return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeIdempotencyConflict, "任务状态与答卷记录不一致")
	}
	var revision qmodel.QuestionnaireAnswerRevision
	if err := db.Where("submission_id = ? AND revision_no = ?", submission.ID, submission.CurrentRevisionNo).First(&revision).Error; err != nil {
		return clientres.SubmitResult{}, err
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeSubmissionInvalid, "答案不是有效的 JSON 对象")
	}
	if !bytes.Equal(qmodel.CanonicalJSON(raw), qmodel.CanonicalJSON(json.RawMessage(revision.AnswersJSON))) {
		return clientres.SubmitResult{}, clientmodel.NewDomainError(clientmodel.CodeIdempotencyConflict, "该任务已提交不同内容")
	}
	var hits []qmodel.QuestionnaireRuleHit
	if err = db.Where("submission_id = ?", submission.ID).Order("id ASC").Find(&hits).Error; err != nil {
		return clientres.SubmitResult{}, err
	}
	hitIDs := make([]uint, 0, len(hits))
	for _, hit := range hits {
		hitIDs = append(hitIDs, hit.ID)
	}
	sort.Slice(hitIDs, func(i, j int) bool { return hitIDs[i] < hitIDs[j] })
	var cases []caseworkmodel.AttentionCase
	if err = db.Where("submission_id = ?", submission.ID).Order("id ASC").Find(&cases).Error; err != nil {
		return clientres.SubmitResult{}, err
	}
	caseIDs := make([]uint, 0, len(cases))
	for _, attentionCase := range cases {
		caseIDs = append(caseIDs, attentionCase.ID)
	}
	return clientres.SubmitResult{
		TaskID: task.ID, SubmissionID: submission.ID, ExecutionStatus: task.ExecutionStatus,
		ReviewStatus: task.ReviewStatus, TaskVersion: task.Version,
		RuleHitIDs: nonNilUint(hitIDs), AttentionCaseIDs: nonNilUint(caseIDs),
	}, nil
}

func nonNilUint(values []uint) []uint {
	if values == nil {
		return []uint{}
	}
	return values
}
