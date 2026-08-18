package carepath

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathreq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	pathres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *CarePathService) RecordTaskContact(ctx context.Context, taskID uint, key string, req pathreq.TaskContactRecord) (pathres.TaskActionResult, error) {
	channel := strings.ToUpper(strings.TrimSpace(req.Channel))
	resultText := strings.TrimSpace(req.Result)
	if taskID == 0 || req.ExpectedVersion == 0 || !pathmodel.IsContactChannel(channel) ||
		resultText == "" || utf8.RuneCountInString(resultText) > 2000 || req.OccurredAt.IsZero() {
		return pathres.TaskActionResult{}, pathmodel.NewDomainError(pathmodel.CodeInvalidArgument, "任务ID、expectedVersion、联系渠道、联系结果和发生时间必填")
	}
	if !s.syntheticFixturesEnabled() {
		return pathres.TaskActionResult{}, pathmodel.NewDomainError(pathmodel.CodeContentDisabled, "固定测试任务能力未启用")
	}
	var visible pathmodel.TaskInstance
	err := s.db().WithContext(ctx).Where("id = ? AND synthetic = ?", taskID, true).First(&visible).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return pathres.TaskActionResult{}, pathmodel.NewForbiddenError("任务不存在或不在当前数据范围")
	}
	if err != nil {
		return pathres.TaskActionResult{}, err
	}
	decision, client, err := s.decisionAndClient(ctx, visible.CareClientID, true)
	if err != nil {
		return pathres.TaskActionResult{}, err
	}
	commandCtx := withDepartment(ctx, client.DeptId)
	req.Channel = channel
	req.Result = resultText
	return runIdempotent(s, commandCtx, operation("TASK_CONTACT", taskID), key, req, func(tx *gorm.DB) (pathres.TaskActionResult, error) {
		var task pathmodel.TaskInstance
		if err := lockQuery(tx).Where("id = ? AND synthetic = ?", taskID, true).First(&task).Error; err != nil {
			return pathres.TaskActionResult{}, err
		}
		if task.CareClientID != visible.CareClientID {
			return pathres.TaskActionResult{}, pathmodel.NewForbiddenError("任务不在当前责任范围")
		}
		if task.Version != req.ExpectedVersion {
			return pathres.TaskActionResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "任务已被其他操作更新")
		}
		var plan pathmodel.PlanInstance
		if err := tx.Where("id = ?", task.PlanInstanceID).First(&plan).Error; err != nil {
			return pathres.TaskActionResult{}, err
		}
		updated := tx.Model(&pathmodel.TaskInstance{}).
			Where("id = ? AND version = ?", task.ID, req.ExpectedVersion).
			Update("version", gorm.Expr("version + 1"))
		if updated.Error != nil {
			return pathres.TaskActionResult{}, updated.Error
		}
		if updated.RowsAffected != 1 {
			return pathres.TaskActionResult{}, pathmodel.NewDomainError(pathmodel.CodeVersionConflict, "任务已被其他操作更新")
		}
		event := pathmodel.CarePathEvent{
			EventID:      uuid.NewString(),
			EventType:    pathmodel.EventTaskContactRecorded,
			CareClientID: task.CareClientID, EnrollmentID: plan.EnrollmentID, PlanInstanceID: task.PlanInstanceID,
			TaskInstanceID: &task.ID, ActorID: actorID(commandCtx), Source: sourceForRole(decision.RoleType),
			Channel: req.Channel, Reason: req.Result,
			FromStatus: task.ExecutionStatus, ToStatus: task.ExecutionStatus,
			OccurredAt: req.OccurredAt, Synthetic: task.Synthetic, DeptId: task.DeptId,
		}
		if err := appendDomainEvent(tx, event); err != nil {
			return pathres.TaskActionResult{}, err
		}
		var created pathmodel.CarePathEvent
		if err := tx.Where("event_id = ?", event.EventID).First(&created).Error; err != nil {
			return pathres.TaskActionResult{}, err
		}
		if err := appendOutbox(tx, pathmodel.EventTaskContactRecorded, "CareTask", task.ID, map[string]any{
			"careClientId":   task.CareClientID,
			"taskInstanceId": task.ID,
			"channel":        req.Channel,
			"result":         req.Result,
			"actorId":        actorID(commandCtx),
			"synthetic":      task.Synthetic,
		}, req.OccurredAt, strings.TrimSpace(key), task.DeptId); err != nil {
			return pathres.TaskActionResult{}, err
		}
		return pathres.TaskActionResult{
			ResourceID: task.ID, ActionID: created.ID, Status: task.ExecutionStatus,
			Version: req.ExpectedVersion + 1, OccurredAt: req.OccurredAt,
		}, nil
	})
}
