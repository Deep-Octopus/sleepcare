package carepath

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathreq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	pathres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *CarePathService) PreviewPlan(ctx context.Context, careClientID uint, key string, req pathreq.PreviewPlan) (pathres.PlanPreview, error) {
	_, client, err := s.decisionAndClient(ctx, careClientID, true)
	if err != nil {
		return pathres.PlanPreview{}, err
	}
	if req.PlanTemplateVersionID == 0 || req.AnchorAt.IsZero() {
		return pathres.PlanPreview{}, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "计划模板版本和 anchorAt 必填")
	}
	value, err := s.loadTemplate(ctx, s.db(), req.PlanTemplateVersionID)
	if err != nil {
		return pathres.PlanPreview{}, err
	}
	if err = s.validateTemplate(ctx, value); err != nil {
		return pathres.PlanPreview{}, err
	}
	commandCtx := withDepartment(ctx, client.DeptId)
	request := struct {
		CareClientID uint                `json:"careClientId"`
		Request      pathreq.PreviewPlan `json:"request"`
	}{CareClientID: careClientID, Request: req}
	return runIdempotent(s, commandCtx, operation("PREVIEW_PLAN", careClientID), key, request, func(tx *gorm.DB) (pathres.PlanPreview, error) {
		now := s.now()
		preview := pathmodel.PlanPreview{
			PreviewID: uuid.NewString(), CareClientID: careClientID, PlanTemplateVersionID: value.Template.ID,
			AnchorAt: req.AnchorAt, ExpiresAt: now.Add(s.previewTTL()), TemplateDefinitionHash: value.Template.DefinitionHash,
			Synthetic: true,
		}
		if err := tx.Create(&preview).Error; err != nil {
			return pathres.PlanPreview{}, err
		}
		result, err := buildPreviewResponse(preview, value.Tasks)
		if err != nil {
			return pathres.PlanPreview{}, err
		}
		return result, nil
	})
}

func buildPreviewResponse(preview pathmodel.PlanPreview, definitions []pathmodel.PlanTaskDefinition) (pathres.PlanPreview, error) {
	tasks := make([]pathres.PreviewTask, 0, len(definitions))
	for _, definition := range definitions {
		item, err := taskDefinitionResponse(definition)
		if err != nil {
			return pathres.PlanPreview{}, err
		}
		openAt := preview.AnchorAt.Add(seconds(definition.OpenOffsetSeconds))
		dueAt := preview.AnchorAt.Add(seconds(definition.DueOffsetSeconds))
		var expires *time.Time
		if definition.ExpiresOffsetSeconds != nil {
			value := preview.AnchorAt.Add(seconds(*definition.ExpiresOffsetSeconds))
			expires = &value
		}
		tasks = append(tasks, pathres.PreviewTask{PlanTaskDefinition: item, OpenAt: openAt, DueAt: dueAt, ExpiresAt: expires})
	}
	return pathres.PlanPreview{
		PreviewID: preview.PreviewID, CareClientID: preview.CareClientID,
		PlanTemplateVersionID: preview.PlanTemplateVersionID, AnchorAt: preview.AnchorAt,
		ExpiresAt: preview.ExpiresAt, Tasks: tasks,
	}, nil
}

func (s *CarePathService) StartPlan(ctx context.Context, careClientID uint, key string, req pathreq.StartPlan) (pathres.PlanInstanceResult, error) {
	decision, client, err := s.decisionAndClient(ctx, careClientID, true)
	if err != nil {
		return pathres.PlanInstanceResult{}, err
	}
	if req.ExpectedClientVersion == 0 || strings.TrimSpace(req.PreviewID) == "" {
		return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "expectedClientVersion 和 previewId 必填")
	}
	commandCtx := withDepartment(ctx, client.DeptId)
	request := struct {
		CareClientID uint              `json:"careClientId"`
		Request      pathreq.StartPlan `json:"request"`
	}{CareClientID: careClientID, Request: req}
	return runIdempotent(s, commandCtx, operation("START_PLAN", careClientID), key, request, func(tx *gorm.DB) (pathres.PlanInstanceResult, error) {
		var preview pathmodel.PlanPreview
		err := lockQuery(tx).Where("preview_id = ? AND care_client_id = ? AND synthetic = ?", strings.TrimSpace(req.PreviewID), careClientID, true).First(&preview).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeResourceNotFound, "计划预览不存在或不在当前责任范围")
		}
		if err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		if preview.ConsumedAt != nil {
			if preview.PlanInstanceID == nil {
				return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeOperationNotAllowed, "计划预览消费状态不完整")
			}
			return loadPlanResult(tx, *preview.PlanInstanceID)
		}
		now := s.now()
		if !now.Before(preview.ExpiresAt) {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeOperationNotAllowed, "计划预览已过期，请重新预览")
		}
		value, err := s.loadTemplate(commandCtx, tx, preview.PlanTemplateVersionID)
		if err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		if err = s.validateTemplate(commandCtx, value); err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		if preview.TemplateDefinitionHash != value.Template.DefinitionHash {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "预览绑定的模板定义已不一致")
		}
		var lockedClient caremodel.CareClient
		err = lockQuery(tx).Where("id = ?", careClientID).First(&lockedClient).Error
		if err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		if lockedClient.Status != caremodel.ClientStatusActive || !lockedClient.Synthetic {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeCareClientUnavailable, "P1-04 只允许为活动的测试康养用户操作计划")
		}
		if lockedClient.Version != req.ExpectedClientVersion {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "康养用户已被其他操作更新")
		}
		var existing pathmodel.Enrollment
		existingErr := lockQuery(tx).Where("care_client_id = ? AND path_code = ? AND active_slot = ?", careClientID, value.Path.Code, value.Path.Code).First(&existing).Error
		if existingErr == nil {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeActiveEnrollmentConflict, "该康养用户已存在活动的 OSA 路径加入记录")
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return pathres.PlanInstanceResult{}, existingErr
		}
		activeSlot := value.Path.Code
		enrollment := pathmodel.Enrollment{
			CareClientID: careClientID, PathDefinitionVersionID: value.Path.ID, PathCode: value.Path.Code,
			ActiveSlot: &activeSlot, Status: pathmodel.EnrollmentActive, StartedAt: &now,
			Version: 1, Synthetic: true,
		}
		if err = tx.Create(&enrollment).Error; err != nil {
			if duplicateError(err) {
				return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeActiveEnrollmentConflict, "并发启动产生了活动路径冲突")
			}
			return pathres.PlanInstanceResult{}, err
		}
		plan := pathmodel.PlanInstance{
			EnrollmentID: enrollment.ID, CareClientID: careClientID, PlanTemplateVersionID: value.Template.ID,
			PreviewID: preview.ID, AnchorAt: preview.AnchorAt, Status: pathmodel.EnrollmentActive,
			PauseStrategy: value.Template.PauseStrategy, Version: 1, Synthetic: true,
		}
		if err = tx.Create(&plan).Error; err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		taskIDs := make([]uint, 0, len(value.Tasks))
		openedTasks := make([]pathmodel.TaskInstance, 0, 1)
		for _, definition := range value.Tasks {
			openAt := preview.AnchorAt.Add(seconds(definition.OpenOffsetSeconds))
			dueAt := preview.AnchorAt.Add(seconds(definition.DueOffsetSeconds))
			var expiresAt *time.Time
			if definition.ExpiresOffsetSeconds != nil {
				value := preview.AnchorAt.Add(seconds(*definition.ExpiresOffsetSeconds))
				expiresAt = &value
			}
			executionStatus := pathmodel.ExecutionScheduled
			var openedAt *time.Time
			if !now.Before(openAt) {
				executionStatus = pathmodel.ExecutionOpen
				openedAt = &now
			}
			reviewStatus := pathmodel.ReviewNotRequired
			if definition.ReviewRequired {
				reviewStatus = pathmodel.ReviewNotReady
			}
			task := pathmodel.TaskInstance{
				PlanInstanceID: plan.ID, CareClientID: careClientID, TaskDefinitionID: definition.ID,
				DayCode: definition.DayCode, Title: definition.Title, Sort: definition.Sort,
				ExecutionRole: definition.ExecutionRole, ExecutionStatus: executionStatus,
				ReviewStatus: reviewStatus, ReviewRole: definition.ReviewRole,
				OpenAt: openAt, DueAt: dueAt, ExpiresAt: expiresAt,
				QuestionnaireVersionID:  definition.QuestionnaireVersionID,
				BoundRuleVersionIDsJSON: append([]byte(nil), definition.BoundRuleVersionIDsJSON...),
				LateSubmissionPolicy:    value.Template.LateSubmissionPolicy,
				NotificationPolicy:      definition.NotificationPolicy, OpenedAt: openedAt,
				Version: 1, Synthetic: true,
			}
			if err = tx.Create(&task).Error; err != nil {
				return pathres.PlanInstanceResult{}, err
			}
			taskIDs = append(taskIDs, task.ID)
			if executionStatus == pathmodel.ExecutionOpen {
				openedTasks = append(openedTasks, task)
			}
		}
		if err = appendDomainEvent(tx, pathmodel.CarePathEvent{
			EventType: pathmodel.EventPlanStarted, CareClientID: careClientID, EnrollmentID: enrollment.ID,
			PlanInstanceID: plan.ID, ActorID: actorID(commandCtx), Source: sourceForRole(decision.RoleType),
			FromStatus: pathmodel.EnrollmentPendingStart, ToStatus: pathmodel.EnrollmentActive,
			OccurredAt: now, Synthetic: true, DeptId: client.DeptId,
		}); err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		if err = appendOutbox(tx, pathmodel.EventPlanStarted, "CarePlan", plan.ID, map[string]any{
			"careClientId": careClientID, "enrollmentId": enrollment.ID, "planInstanceId": plan.ID,
			"planTemplateVersionId": value.Template.ID, "anchorAt": preview.AnchorAt, "taskIds": taskIDs,
			"synthetic": true,
		}, now, preview.PreviewID, client.DeptId); err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		for _, task := range openedTasks {
			taskID := task.ID
			if err = appendDomainEvent(tx, pathmodel.CarePathEvent{
				EventType: pathmodel.EventTaskOpened, CareClientID: careClientID, EnrollmentID: enrollment.ID,
				PlanInstanceID: plan.ID, TaskInstanceID: &taskID, ActorID: actorID(commandCtx),
				Source: pathmodel.EventSourceSystem, FromStatus: pathmodel.ExecutionScheduled,
				ToStatus: pathmodel.ExecutionOpen, OccurredAt: now, Synthetic: true,
				DeptId: client.DeptId,
			}); err != nil {
				return pathres.PlanInstanceResult{}, err
			}
			if err = appendOutbox(tx, pathmodel.EventTaskOpened, "CareTask", task.ID, map[string]any{
				"careClientId": careClientID, "enrollmentId": enrollment.ID, "planInstanceId": plan.ID,
				"taskInstanceId": task.ID, "dayCode": task.DayCode, "openAt": task.OpenAt,
				"openedAt": now, "synthetic": true,
			}, now, preview.PreviewID, client.DeptId); err != nil {
				return pathres.PlanInstanceResult{}, err
			}
		}
		if err = tx.Model(&preview).Updates(map[string]any{"consumed_at": now, "plan_instance_id": plan.ID}).Error; err != nil {
			return pathres.PlanInstanceResult{}, err
		}
		clientUpdate := tx.Model(&caremodel.CareClient{}).Where("id = ? AND version = ?", careClientID, req.ExpectedClientVersion).
			Update("version", gorm.Expr("version + 1"))
		if clientUpdate.Error != nil {
			return pathres.PlanInstanceResult{}, clientUpdate.Error
		}
		if clientUpdate.RowsAffected != 1 {
			return pathres.PlanInstanceResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "康养用户已被其他操作更新")
		}
		return pathres.PlanInstanceResult{
			EnrollmentID: enrollment.ID, PlanInstanceID: plan.ID, CareClientID: careClientID,
			AnchorAt: plan.AnchorAt, Status: plan.Status, TaskIDs: taskIDs, Version: plan.Version,
		}, nil
	})
}

func (s *CarePathService) PausePlan(ctx context.Context, planID uint, key string, req pathreq.PlanStateAction) (pathres.PlanActionResult, error) {
	return s.changePlanState(ctx, planID, key, req, pathmodel.EnrollmentActive, pathmodel.EnrollmentPaused, pathmodel.EventPlanPaused)
}

func (s *CarePathService) ResumePlan(ctx context.Context, planID uint, key string, req pathreq.PlanStateAction) (pathres.PlanActionResult, error) {
	return s.changePlanState(ctx, planID, key, req, pathmodel.EnrollmentPaused, pathmodel.EnrollmentActive, pathmodel.EventPlanResumed)
}

func (s *CarePathService) changePlanState(ctx context.Context, planID uint, key string, req pathreq.PlanStateAction, from, to, eventType string) (pathres.PlanActionResult, error) {
	reason := strings.TrimSpace(req.Reason)
	if planID == 0 || req.ExpectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 1000 {
		return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "计划ID、expectedVersion 和不超过 1000 字符的原因必填")
	}
	if !s.syntheticFixturesEnabled() {
		return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 测试计划能力未启用")
	}
	var visiblePlan pathmodel.PlanInstance
	if err := s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", planID, true).First(&visiblePlan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pathres.PlanActionResult{}, pathmodel.NewForbiddenError("计划不存在或不在当前数据范围")
		}
		return pathres.PlanActionResult{}, err
	}
	decision, client, err := s.decisionAndClient(ctx, visiblePlan.CareClientID, true)
	if err != nil {
		return pathres.PlanActionResult{}, err
	}
	commandCtx := withDepartment(ctx, client.DeptId)
	return runIdempotent(s, commandCtx, operation(eventType, planID), key, req, func(tx *gorm.DB) (pathres.PlanActionResult, error) {
		var plan pathmodel.PlanInstance
		if err := lockQuery(tx).Where("id = ? AND synthetic = ?", planID, true).First(&plan).Error; err != nil {
			return pathres.PlanActionResult{}, err
		}
		if plan.Version != req.ExpectedVersion {
			return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "计划已被其他操作更新")
		}
		if plan.Status != from || !pathmodel.CanTransitionEnrollment(from, to) {
			return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeOperationNotAllowed, "当前计划状态不允许该动作")
		}
		if plan.PauseStrategy != pathmodel.PauseStrategyKeepWindows {
			return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeOperationNotAllowed, "P1-04 仅支持 KEEP_WINDOWS 暂停策略")
		}
		now := s.now()
		planUpdates := map[string]any{"status": to, "version": gorm.Expr("version + 1")}
		if to == pathmodel.EnrollmentPaused {
			planUpdates["paused_at"] = now
		} else {
			planUpdates["paused_at"] = nil
		}
		result := tx.Model(&pathmodel.PlanInstance{}).Where("id = ? AND version = ? AND status = ?", plan.ID, req.ExpectedVersion, from).Updates(planUpdates)
		if result.Error != nil {
			return pathres.PlanActionResult{}, result.Error
		}
		if result.RowsAffected != 1 {
			return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "计划已被其他操作更新")
		}
		enrollmentResult := tx.Model(&pathmodel.Enrollment{}).Where("id = ? AND status = ?", plan.EnrollmentID, from).
			Updates(map[string]any{"status": to, "version": gorm.Expr("version + 1")})
		if enrollmentResult.Error != nil {
			return pathres.PlanActionResult{}, enrollmentResult.Error
		}
		if enrollmentResult.RowsAffected != 1 {
			return pathres.PlanActionResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "路径加入记录状态不一致")
		}
		if err := appendDomainEvent(tx, pathmodel.CarePathEvent{
			EventType: eventType, CareClientID: plan.CareClientID, EnrollmentID: plan.EnrollmentID,
			PlanInstanceID: plan.ID, ActorID: actorID(commandCtx), Source: sourceForRole(decision.RoleType),
			Reason: reason, FromStatus: from, ToStatus: to,
			OccurredAt: now, Synthetic: true, DeptId: plan.DeptId,
		}); err != nil {
			return pathres.PlanActionResult{}, err
		}
		if err := appendOutbox(tx, eventType, "CarePlan", plan.ID, map[string]any{
			"careClientId": plan.CareClientID, "enrollmentId": plan.EnrollmentID,
			"planInstanceId": plan.ID, "fromStatus": from, "toStatus": to,
			"reason": reason, "synthetic": true,
		}, now, strings.TrimSpace(key), plan.DeptId); err != nil {
			return pathres.PlanActionResult{}, err
		}
		if to == pathmodel.EnrollmentActive {
			if err := s.openDueTasks(tx, plan, now); err != nil {
				return pathres.PlanActionResult{}, err
			}
		}
		return pathres.PlanActionResult{PlanInstanceID: plan.ID, EnrollmentID: plan.EnrollmentID, Status: to, Version: req.ExpectedVersion + 1}, nil
	})
}

func (s *CarePathService) ReconcilePlanTasks(ctx context.Context, planID uint) error {
	if !s.syntheticFixturesEnabled() {
		return pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "P1-04 测试计划能力未启用")
	}
	var plan pathmodel.PlanInstance
	if err := s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", planID, true).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pathmodel.NewDomainError(pathmodel.CodeResourceNotFound, "测试计划不存在")
		}
		return err
	}
	identity, hasIdentity := datascope.FromContext(ctx)
	if !hasIdentity || identity == nil || !identity.IsSystem {
		_, client, err := s.decisionAndClient(ctx, plan.CareClientID, false)
		if err != nil {
			return err
		}
		ctx = withDepartment(ctx, client.DeptId)
	}
	return s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockQuery(tx).Where("id = ? AND synthetic = ?", planID, true).First(&plan).Error; err != nil {
			return err
		}
		if plan.Status != pathmodel.EnrollmentActive {
			return nil
		}
		return s.openDueTasks(tx, plan, s.now())
	})
}

func (s *CarePathService) openDueTasks(tx *gorm.DB, plan pathmodel.PlanInstance, now time.Time) error {
	var tasks []pathmodel.TaskInstance
	if err := lockQuery(tx).Where("plan_instance_id = ? AND synthetic = ? AND execution_status = ? AND open_at <= ?", plan.ID, true, pathmodel.ExecutionScheduled, now).
		Order("sort ASC, id ASC").Find(&tasks).Error; err != nil {
		return err
	}
	for _, task := range tasks {
		result := tx.Model(&pathmodel.TaskInstance{}).Where("id = ? AND execution_status = ?", task.ID, pathmodel.ExecutionScheduled).
			Updates(map[string]any{"execution_status": pathmodel.ExecutionOpen, "opened_at": now, "version": gorm.Expr("version + 1")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}
		taskID := task.ID
		if err := appendDomainEvent(tx, pathmodel.CarePathEvent{
			EventType: pathmodel.EventTaskOpened, CareClientID: plan.CareClientID,
			EnrollmentID: plan.EnrollmentID, PlanInstanceID: plan.ID, TaskInstanceID: &taskID,
			ActorID: actorID(tx.Statement.Context), Source: pathmodel.EventSourceSystem,
			FromStatus: pathmodel.ExecutionScheduled, ToStatus: pathmodel.ExecutionOpen,
			OccurredAt: now, Synthetic: true, DeptId: plan.DeptId,
		}); err != nil {
			return err
		}
		if err := appendOutbox(tx, pathmodel.EventTaskOpened, "CareTask", task.ID, map[string]any{
			"careClientId": plan.CareClientID, "enrollmentId": plan.EnrollmentID,
			"planInstanceId": plan.ID, "taskInstanceId": task.ID, "dayCode": task.DayCode,
			"openAt": task.OpenAt, "openedAt": now, "synthetic": true,
		}, now, fmt.Sprintf("%d:%d:%s", plan.ID, task.ID, task.OpenAt.UTC().Format(time.RFC3339Nano)), plan.DeptId); err != nil {
			return err
		}
	}
	return nil
}

func loadPlanResult(db *gorm.DB, planID uint) (pathres.PlanInstanceResult, error) {
	var plan pathmodel.PlanInstance
	if err := db.Where("id = ? AND synthetic = ?", planID, true).First(&plan).Error; err != nil {
		return pathres.PlanInstanceResult{}, err
	}
	var tasks []pathmodel.TaskInstance
	if err := db.Where("plan_instance_id = ? AND synthetic = ?", plan.ID, true).Order("sort ASC, id ASC").Find(&tasks).Error; err != nil {
		return pathres.PlanInstanceResult{}, err
	}
	ids := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return pathres.PlanInstanceResult{
		EnrollmentID: plan.EnrollmentID, PlanInstanceID: plan.ID, CareClientID: plan.CareClientID,
		AnchorAt: plan.AnchorAt, Status: plan.Status, TaskIDs: ids, Version: plan.Version,
	}, nil
}

func seconds(value int64) time.Duration { return time.Duration(value) * time.Second }
