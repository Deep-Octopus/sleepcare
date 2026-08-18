package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *NotificationService) CreateInitial(ctx context.Context, taskID uint, commandKey string) (caseworkres.ActionResult, error) {
	if !isSystemContext(ctx) {
		return caseworkres.ActionResult{}, notificationmodel.NewForbiddenError("初始通知只能由系统测试初始化创建")
	}
	if taskID == 0 || strings.TrimSpace(commandKey) == "" {
		return caseworkres.ActionResult{}, notificationmodel.NewDomainError(notificationmodel.CodeInvalidArgument, "任务ID和初始化键必填")
	}
	if !s.fixturesEnabled() {
		return caseworkres.ActionResult{}, notificationmodel.NewDomainError(notificationmodel.CodeOperationNotAllowed, "固定测试通知能力未启用")
	}
	var created notificationmodel.NotificationAttempt
	err := s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingRequest notificationmodel.NotificationRequest
		existingErr := tx.Where("task_id = ?", taskID).First(&existingRequest).Error
		if existingErr == nil {
			return tx.Where("notification_request_id = ? AND attempt_no = 1", existingRequest.ID).First(&created).Error
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		var task pathmodel.TaskInstance
		if err := tx.Where("id = ? AND synthetic = ?", taskID, true).First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notificationmodel.NewDomainError(notificationmodel.CodeResourceNotFound, "固定测试任务不存在")
			}
			return err
		}
		now := s.now()
		request := notificationmodel.NotificationRequest{
			TaskID: task.ID, CareClientID: task.CareClientID, Channel: notificationmodel.ChannelDemo,
			RequestedAt: now, Synthetic: task.Synthetic, DeptId: task.DeptId,
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
		created = notificationmodel.NotificationAttempt{
			NotificationRequestID: request.ID, AttemptNo: 1, TaskID: task.ID, CareClientID: task.CareClientID,
			Channel: notificationmodel.ChannelDemo, Status: notificationmodel.AttemptStatusPending,
			RequestedAt: now, Version: 1, ActorID: 0,
			Operation:        fmt.Sprintf("INITIAL_NOTIFICATION:%d", task.ID),
			CommandKeyDigest: digest(commandKey), RequestHash: digest(fmt.Sprint(task.ID)),
			Synthetic: task.Synthetic, DeptId: task.DeptId,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		event := notificationmodel.DeliveryEvent{
			EventID: uuid.NewString(), EventKey: "requested", NotificationRequestID: request.ID,
			NotificationAttemptID: created.ID, EventType: notificationmodel.EventNotificationRequested,
			ToStatus: notificationmodel.AttemptStatusPending, OccurredAt: now,
			Synthetic: task.Synthetic, DeptId: task.DeptId,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return appendOutbox(tx, created, event)
	})
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	if notificationmodel.IsFinalAttemptStatus(created.Status) {
		return s.currentAction(ctx, created.ID)
	}
	return s.dispatch(ctx, created, s.adapter())
}

func (s *NotificationService) Resend(ctx context.Context, sourceAttemptID uint, key string, req notificationreq.Resend) (caseworkres.ActionResult, error) {
	reason := strings.TrimSpace(req.Reason)
	if sourceAttemptID == 0 || req.ExpectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 1000 {
		return caseworkres.ActionResult{}, notificationmodel.NewDomainError(notificationmodel.CodeInvalidArgument, "attempt ID、expectedVersion 和不超过 1000 字符的原因必填")
	}
	if !s.fixturesEnabled() {
		return caseworkres.ActionResult{}, notificationmodel.NewDomainError(notificationmodel.CodeOperationNotAllowed, "固定测试通知能力未启用")
	}
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return caseworkres.ActionResult{}, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleCareSteward {
		return caseworkres.ActionResult{}, notificationmodel.NewForbiddenError("仅当前责任管家可创建补发尝试")
	}
	var visible notificationmodel.NotificationAttempt
	err = scopedAttemptQuery(s.db().WithContext(ctx), decision, s.now()).
		Select("notification_attempts.*").Where("notification_attempts.id = ?", sourceAttemptID).First(&visible).Error
	if err != nil {
		return caseworkres.ActionResult{}, notFoundAsForbidden(err, "通知尝试不存在或不在当前责任范围")
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 128 {
		return caseworkres.ActionResult{}, notificationmodel.NewDomainError(notificationmodel.CodeInvalidArgument, "Idempotency-Key 必填且不超过 128 字符")
	}
	hash, err := requestHash(req)
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	operation := operationForResend(sourceAttemptID)
	keyDigest := digest(key)
	commandCtx := withDepartment(ctx, visible.DeptId)
	var attempt notificationmodel.NotificationAttempt
	var replayed bool
	err = s.db().WithContext(commandCtx).Transaction(func(tx *gorm.DB) error {
		existingErr := tx.Where("actor_id = ? AND operation = ? AND command_key_digest = ?", decision.Identity.UserID, operation, keyDigest).
			First(&attempt).Error
		if existingErr == nil {
			if attempt.RequestHash != hash {
				return notificationmodel.NewDomainError(notificationmodel.CodeIdempotencyConflict, "幂等键已用于不同请求")
			}
			replayed = true
			return nil
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		var source notificationmodel.NotificationAttempt
		if err := locking(tx).Where("id = ?", sourceAttemptID).First(&source).Error; err != nil {
			return err
		}
		if source.CareClientID != visible.CareClientID || source.DeptId != visible.DeptId {
			return notificationmodel.NewForbiddenError("通知尝试不在当前责任范围")
		}
		if source.Version != req.ExpectedVersion {
			return notificationmodel.NewDomainError(notificationmodel.CodeVersionConflict, "通知尝试已被其他回执更新")
		}
		if source.Status != notificationmodel.AttemptStatusFailed && source.Status != notificationmodel.AttemptStatusUnknown {
			return notificationmodel.NewDomainError(notificationmodel.CodeOperationNotAllowed, "仅失败或未知终态可创建补发尝试")
		}
		if source.FinalizedAt == nil {
			return notificationmodel.NewDomainError(notificationmodel.CodeNotificationFinalized, "通知尝试终态事实不完整")
		}
		var request notificationmodel.NotificationRequest
		if err := locking(tx).Where("id = ?", source.NotificationRequestID).First(&request).Error; err != nil {
			return err
		}
		var maxAttemptNo int
		if err := tx.Model(&notificationmodel.NotificationAttempt{}).
			Where("notification_request_id = ?", request.ID).Select("COALESCE(MAX(attempt_no), 0)").Scan(&maxAttemptNo).Error; err != nil {
			return err
		}
		now := s.now()
		retryOf := source.ID
		attempt = notificationmodel.NotificationAttempt{
			NotificationRequestID: request.ID, AttemptNo: maxAttemptNo + 1,
			TaskID: source.TaskID, CareClientID: source.CareClientID, RetryOfAttemptID: &retryOf,
			Channel: notificationmodel.ChannelDemo, Status: notificationmodel.AttemptStatusPending,
			RequestedAt: now, ResendReason: reason, Version: 1,
			ActorID: decision.Identity.UserID, Operation: operation,
			CommandKeyDigest: keyDigest, RequestHash: hash,
			Synthetic: source.Synthetic, DeptId: source.DeptId, CreatedBy: decision.Identity.UserID,
		}
		if err := tx.Create(&attempt).Error; err != nil {
			if duplicateError(err) {
				return notificationmodel.NewDomainError(notificationmodel.CodeIdempotencyConflict, "补发请求发生并发冲突，请重试")
			}
			return err
		}
		event := notificationmodel.DeliveryEvent{
			EventID: uuid.NewString(), EventKey: "requested", NotificationRequestID: request.ID,
			NotificationAttemptID: attempt.ID, EventType: notificationmodel.EventNotificationRequested,
			ToStatus: notificationmodel.AttemptStatusPending, OccurredAt: now,
			Synthetic: attempt.Synthetic, DeptId: attempt.DeptId, CreatedBy: decision.Identity.UserID,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return appendOutbox(tx, attempt, event)
	})
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	if replayed && notificationmodel.IsFinalAttemptStatus(attempt.Status) {
		return s.currentAction(commandCtx, attempt.ID)
	}
	return s.dispatch(commandCtx, attempt, s.adapter())
}

func (s *NotificationService) dispatch(ctx context.Context, attempt notificationmodel.NotificationAttempt, adapter NotificationPort) (caseworkres.ActionResult, error) {
	receipts, err := adapter.Submit(ctx, SendCommand{
		NotificationRequestID: attempt.NotificationRequestID,
		NotificationAttemptID: attempt.ID,
		TaskID:                attempt.TaskID, AttemptNo: attempt.AttemptNo, RequestedAt: attempt.RequestedAt,
	})
	if err != nil {
		return caseworkres.ActionResult{}, err
	}
	var result caseworkres.ActionResult
	for _, receipt := range receipts {
		result, err = s.ApplyDeliveryReceipt(ctx, attempt.ID, receipt)
		if err != nil {
			return caseworkres.ActionResult{}, err
		}
	}
	if len(receipts) == 0 {
		return s.currentAction(ctx, attempt.ID)
	}
	return result, nil
}

func (s *NotificationService) ApplyDeliveryReceipt(ctx context.Context, attemptID uint, receipt DeliveryReceipt) (caseworkres.ActionResult, error) {
	receipt.EventKey = strings.TrimSpace(receipt.EventKey)
	receipt.Status = strings.ToUpper(strings.TrimSpace(receipt.Status))
	if attemptID == 0 || receipt.EventKey == "" || receipt.Status == "" ||
		receipt.Status == notificationmodel.AttemptStatusPending || !notificationmodel.IsAttemptStatus(receipt.Status) {
		return caseworkres.ActionResult{}, notificationmodel.NewDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执缺少事件键或包含无效状态")
	}
	var result caseworkres.ActionResult
	err := s.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempt notificationmodel.NotificationAttempt
		if err := locking(tx).Where("id = ?", attemptID).First(&attempt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notificationmodel.NewDomainError(notificationmodel.CodeResourceNotFound, "通知尝试不存在")
			}
			return err
		}
		var existing notificationmodel.DeliveryEvent
		existingErr := tx.Where("notification_attempt_id = ? AND event_key = ?", attempt.ID, receipt.EventKey).First(&existing).Error
		if existingErr == nil {
			result = actionResult(attempt, existing.ID, existing.OccurredAt)
			return nil
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return existingErr
		}
		if notificationmodel.IsFinalAttemptStatus(attempt.Status) {
			return notificationmodel.NewDomainError(notificationmodel.CodeNotificationFinalized, "终态通知尝试不可再应用新回执")
		}
		if !notificationmodel.CanTransitionAttempt(attempt.Status, receipt.Status) {
			return notificationmodel.NewDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执状态顺序无效")
		}
		occurredAt := receipt.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = s.now()
		}
		if occurredAt.Before(attempt.RequestedAt) {
			return notificationmodel.NewDomainError(notificationmodel.CodeDeliveryEventInvalid, "通知回执时间早于请求时间")
		}
		updates := map[string]any{"status": receipt.Status, "version": gorm.Expr("version + 1")}
		switch receipt.Status {
		case notificationmodel.AttemptStatusSubmittedToProvider:
			updates["submitted_at"] = occurredAt
		case notificationmodel.AttemptStatusAccepted:
			updates["accepted_at"] = occurredAt
		case notificationmodel.AttemptStatusDelivered:
			updates["delivered_at"] = occurredAt
			updates["finalized_at"] = occurredAt
			updates["failure_code"] = ""
		case notificationmodel.AttemptStatusFailed, notificationmodel.AttemptStatusUnknown:
			updates["finalized_at"] = occurredAt
			updates["failure_code"] = strings.TrimSpace(receipt.FailureCode)
		}
		updated := tx.Model(&notificationmodel.NotificationAttempt{}).
			Where("id = ? AND version = ? AND status = ?", attempt.ID, attempt.Version, attempt.Status).
			Updates(updates)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return notificationmodel.NewDomainError(notificationmodel.CodeVersionConflict, "通知尝试已被其他回执更新")
		}
		event := notificationmodel.DeliveryEvent{
			EventID: uuid.NewString(), EventKey: receipt.EventKey,
			NotificationRequestID: attempt.NotificationRequestID, NotificationAttemptID: attempt.ID,
			EventType:  notificationmodel.EventTypeForStatus(receipt.Status),
			FromStatus: attempt.Status, ToStatus: receipt.Status, OccurredAt: occurredAt,
			FailureCode: strings.TrimSpace(receipt.FailureCode), AdapterReferenceHash: digest(receipt.AdapterReference),
			Synthetic: attempt.Synthetic, DeptId: attempt.DeptId, CreatedBy: attempt.CreatedBy,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		attempt.Status = receipt.Status
		attempt.Version++
		if value, ok := updates["submitted_at"]; ok {
			timestamp := value.(time.Time)
			attempt.SubmittedAt = &timestamp
		}
		if value, ok := updates["accepted_at"]; ok {
			timestamp := value.(time.Time)
			attempt.AcceptedAt = &timestamp
		}
		if value, ok := updates["delivered_at"]; ok {
			timestamp := value.(time.Time)
			attempt.DeliveredAt = &timestamp
		}
		if value, ok := updates["finalized_at"]; ok {
			timestamp := value.(time.Time)
			attempt.FinalizedAt = &timestamp
		}
		attempt.FailureCode = strings.TrimSpace(receipt.FailureCode)
		if err := appendOutbox(tx, attempt, event); err != nil {
			return err
		}
		if notificationmodel.IsFinalAttemptStatus(receipt.Status) {
			var request notificationmodel.NotificationRequest
			if err := tx.Where("id = ?", attempt.NotificationRequestID).First(&request).Error; err != nil {
				return err
			}
			if receipt.Status == notificationmodel.AttemptStatusDelivered {
				if err := completeDeliveryTodo(tx, request.ID, occurredAt); err != nil {
					return err
				}
			} else if err := ensureDeliveryTodo(tx, request, occurredAt); err != nil {
				return err
			}
		}
		result = actionResult(attempt, event.ID, occurredAt)
		return nil
	})
	return result, err
}

func (s *NotificationService) currentAction(ctx context.Context, attemptID uint) (caseworkres.ActionResult, error) {
	var attempt notificationmodel.NotificationAttempt
	if err := s.db().WithContext(ctx).Where("id = ?", attemptID).First(&attempt).Error; err != nil {
		return caseworkres.ActionResult{}, err
	}
	var event notificationmodel.DeliveryEvent
	if err := s.db().WithContext(ctx).Where("notification_attempt_id = ?", attempt.ID).
		Order("occurred_at DESC, id DESC").First(&event).Error; err != nil {
		return caseworkres.ActionResult{}, err
	}
	return actionResult(attempt, event.ID, event.OccurredAt), nil
}
