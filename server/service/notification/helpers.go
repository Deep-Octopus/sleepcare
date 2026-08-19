package notification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationres "github.com/flipped-aurora/gin-vue-admin/server/model/notification/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func locking(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func requestHash(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return digest(string(payload)), nil
}

func actorID(ctx context.Context) uint {
	if identity, ok := datascope.FromContext(ctx); ok && identity != nil {
		return identity.UserID
	}
	return 0
}

func withDepartment(ctx context.Context, deptID uint) context.Context {
	identity, ok := datascope.FromContext(ctx)
	if !ok || identity == nil {
		return ctx
	}
	copyIdentity := *identity
	copyIdentity.DeptID = deptID
	return datascope.WithIdentity(ctx, &copyIdentity)
}

func isSystemContext(ctx context.Context) bool {
	identity, ok := datascope.FromContext(ctx)
	return ok && identity != nil && identity.IsSystem
}

func normalizeAccessError(err error) error {
	var domainErr *caremodel.DomainError
	if errors.As(err, &domainErr) {
		return &notificationmodel.DomainError{Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus}
	}
	return err
}

func activeSteward(db *gorm.DB, careClientID uint, now time.Time) (caremodel.CareAssignment, error) {
	var assignment caremodel.CareAssignment
	err := db.Where("care_client_id = ? AND role_type = ? AND cancelled_at IS NULL AND valid_from <= ?", careClientID, caremodel.AssignmentRoleCareSteward, now).
		Where("valid_until IS NULL OR valid_until > ?", now).
		Order("valid_from DESC, id DESC").First(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return assignment, notificationmodel.NewDomainError(notificationmodel.CodeCareAssignmentRequired, "当前对象缺少有效责任管家")
	}
	return assignment, err
}

func ensureDeliveryTodo(tx *gorm.DB, request notificationmodel.NotificationRequest, now time.Time) error {
	assignment, err := activeSteward(tx, request.CareClientID, now)
	if err != nil {
		return err
	}
	active := caseworkmodel.TodoActiveSlot
	todo := caseworkmodel.TodoItem{
		Category:   caseworkmodel.TodoCategoryDeliveryIssue,
		SourceType: notificationmodel.TodoSourceNotificationRequest,
		SourceID:   request.ID, ActiveSlot: &active, CareClientID: request.CareClientID,
		AssigneeID: assignment.AssigneeID, AssigneeRole: assignment.RoleType,
		Status: caseworkmodel.TodoStatusOpen, OpenedAt: now, Version: 1,
		Synthetic: request.Synthetic, DeptId: request.DeptId,
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&todo).Error
}

func completeDeliveryTodo(tx *gorm.DB, requestID uint, now time.Time) error {
	return tx.Model(&caseworkmodel.TodoItem{}).
		Where("source_type = ? AND source_id = ? AND active_slot = ?", notificationmodel.TodoSourceNotificationRequest, requestID, caseworkmodel.TodoActiveSlot).
		Updates(map[string]any{
			"status":          caseworkmodel.TodoStatusCompleted,
			"active_slot":     nil,
			"completed_at":    now,
			"completion_note": "后续尝试已确认送达",
			"version":         gorm.Expr("version + 1"),
		}).Error
}

func appendOutbox(tx *gorm.DB, attempt notificationmodel.NotificationAttempt, event notificationmodel.DeliveryEvent) error {
	return platformoutbox.Append(tx, platformoutbox.AppendInput{
		EventType: event.EventType, AggregateType: "NotificationRequest", AggregateID: attempt.NotificationRequestID,
		Payload: map[string]any{
			"notificationRequestId": attempt.NotificationRequestID,
			"notificationAttemptId": attempt.ID,
			"taskId":                attempt.TaskID,
			"careClientId":          attempt.CareClientID,
			"attemptNo":             attempt.AttemptNo,
			"fromStatus":            event.FromStatus,
			"toStatus":              event.ToStatus,
			"failureCode":           event.FailureCode,
			"synthetic":             attempt.Synthetic,
		},
		OccurredAt: event.OccurredAt, CausationID: event.EventID,
		Synthetic: attempt.Synthetic, DeptID: attempt.DeptId, CreatedBy: attempt.CreatedBy,
	})
}

func actionResult(attempt notificationmodel.NotificationAttempt, eventID uint, occurredAt time.Time) caseworkres.ActionResult {
	return caseworkres.ActionResult{
		ResourceID: attempt.ID,
		ActionID:   eventID,
		Status:     attempt.Status,
		Version:    attempt.Version,
		OccurredAt: occurredAt,
	}
}

type attemptRow struct {
	notificationmodel.NotificationAttempt
	CareClientDisplayCode string
	CareClientDisplayName string
}

func loadAttemptResponse(db *gorm.DB, id uint) (notificationres.NotificationAttempt, error) {
	var row attemptRow
	err := db.Model(&notificationmodel.NotificationAttempt{}).
		Select("notification_attempts.*, care_clients.display_code AS care_client_display_code, care_clients.display_name AS care_client_display_name").
		Joins("JOIN care_clients ON care_clients.id = notification_attempts.care_client_id AND care_clients.deleted_at IS NULL").
		Where("notification_attempts.id = ?", id).First(&row).Error
	if err != nil {
		return notificationres.NotificationAttempt{}, err
	}
	var events []notificationmodel.DeliveryEvent
	if err = db.Where("notification_attempt_id = ?", id).Order("occurred_at ASC, id ASC").Find(&events).Error; err != nil {
		return notificationres.NotificationAttempt{}, err
	}
	return attemptResponse(row, events), nil
}

func attemptResponse(row attemptRow, events []notificationmodel.DeliveryEvent) notificationres.NotificationAttempt {
	eventRows := make([]notificationres.DeliveryEvent, 0, len(events))
	for _, event := range events {
		eventRows = append(eventRows, notificationres.DeliveryEvent{
			ID: event.ID, EventType: event.EventType, FromStatus: event.FromStatus,
			ToStatus: event.ToStatus, OccurredAt: event.OccurredAt, FailureCode: event.FailureCode,
		})
	}
	attempt := row.NotificationAttempt
	return notificationres.NotificationAttempt{
		ID: attempt.ID, NotificationRequestID: attempt.NotificationRequestID,
		TaskID: attempt.TaskID, CareClientID: attempt.CareClientID,
		CareClientDisplayCode: row.CareClientDisplayCode, CareClientDisplayName: row.CareClientDisplayName,
		AttemptNo: attempt.AttemptNo, RetryOfAttemptID: attempt.RetryOfAttemptID,
		Channel: attempt.Channel, Status: attempt.Status, RequestedAt: attempt.RequestedAt,
		SubmittedAt: attempt.SubmittedAt, AcceptedAt: attempt.AcceptedAt,
		DeliveredAt: attempt.DeliveredAt, FinalizedAt: attempt.FinalizedAt,
		FailureCode: attempt.FailureCode, ProviderCode: attempt.ProviderCode,
		DispatchPolicyCode: attempt.DispatchPolicyCode, DispatchPolicyVersion: attempt.DispatchPolicyVersion,
		TemplateCode: attempt.TemplateCode, EstimatedCostMinor: attempt.EstimatedCostMinor,
		CostCurrency: attempt.CostCurrency, Version: attempt.Version, Events: eventRows,
	}
}

func duplicateError(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique")
}

func operationForResend(attemptID uint) string {
	return fmt.Sprintf("RESEND_NOTIFICATION:%d", attemptID)
}

func scopedAttemptQuery(db *gorm.DB, decision *accesspolicy.CareClientDecision, now time.Time) *gorm.DB {
	query := db.Model(&notificationmodel.NotificationAttempt{}).
		Joins("JOIN care_clients ON care_clients.id = notification_attempts.care_client_id AND care_clients.deleted_at IS NULL")
	return decision.Scope(query, now)
}
