package casework

import (
	"errors"
	"fmt"
	"strings"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"gorm.io/gorm"
)

type ClientConsultationIdentity struct {
	CareClientID uint
	DeptID       uint
	Synthetic    bool
}

func validConsultationStatus(value string) bool {
	switch value {
	case caseworkmodel.ConsultationStatusNew,
		caseworkmodel.ConsultationStatusWaitingAssignment,
		caseworkmodel.ConsultationStatusAssigned,
		caseworkmodel.ConsultationStatusHandling,
		caseworkmodel.ConsultationStatusWaitingClient,
		caseworkmodel.ConsultationStatusWaitingCollaboration,
		caseworkmodel.ConsultationStatusResolved,
		caseworkmodel.ConsultationStatusClosed:
		return true
	default:
		return false
	}
}

func validConsultationUrgency(value string) bool {
	return value == caseworkmodel.ConsultationUrgencyRoutine || value == caseworkmodel.ConsultationUrgencyExpedited
}

func activeCareAssignment(
	db *gorm.DB,
	careClientID uint,
	assigneeID uint,
	role string,
	now time.Time,
) (caremodel.CareAssignment, error) {
	var assignment caremodel.CareAssignment
	err := db.Where(
		"care_client_id = ? AND assignee_id = ? AND role_type = ? AND cancelled_at IS NULL AND valid_from <= ?",
		careClientID,
		assigneeID,
		role,
		now,
	).Where("valid_until IS NULL OR valid_until > ?", now).
		Order("valid_from DESC, id DESC").First(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return assignment, caseworkmodel.NewForbiddenError(
			caseworkmodel.CodeConsultationAssigneeRequired,
			"目标人员不是当前有效责任人员",
		)
	}
	return assignment, err
}

func currentCareAssignment(
	db *gorm.DB,
	careClientID uint,
	role string,
	now time.Time,
) (caremodel.CareAssignment, error) {
	var assignment caremodel.CareAssignment
	err := db.Where(
		"care_client_id = ? AND role_type = ? AND cancelled_at IS NULL AND valid_from <= ?",
		careClientID,
		role,
		now,
	).Where("valid_until IS NULL OR valid_until > ?", now).
		Order("valid_from DESC, id DESC").First(&assignment).Error
	return assignment, err
}

func createConsultationTodo(db *gorm.DB, consultation caseworkmodel.Consultation, now time.Time) error {
	if consultation.AssigneeID == nil || *consultation.AssigneeID == 0 || consultation.AssigneeRole == "" {
		return caseworkmodel.NewDomainError(
			caseworkmodel.CodeConsultationAssigneeRequired,
			"咨询缺少当前责任人",
		)
	}
	active := caseworkmodel.TodoActiveSlot
	todo := caseworkmodel.TodoItem{
		Category:     caseworkmodel.TodoCategoryConsultation,
		SourceType:   caseworkmodel.TodoSourceConsultation,
		SourceID:     consultation.ID,
		ActiveSlot:   &active,
		CareClientID: consultation.CareClientID,
		AssigneeID:   *consultation.AssigneeID,
		AssigneeRole: consultation.AssigneeRole,
		Status:       caseworkmodel.TodoStatusOpen,
		OpenedAt:     now,
		Version:      1,
		Synthetic:    consultation.Synthetic,
	}
	return db.Create(&todo).Error
}

func supersedeConsultationTodo(db *gorm.DB, consultationID uint, now time.Time, note string) error {
	return db.Model(&caseworkmodel.TodoItem{}).
		Where(
			"source_type = ? AND source_id = ? AND active_slot = ?",
			caseworkmodel.TodoSourceConsultation,
			consultationID,
			caseworkmodel.TodoActiveSlot,
		).
		Updates(map[string]any{
			"status":          caseworkmodel.TodoStatusSuperseded,
			"active_slot":     nil,
			"completed_at":    now,
			"completion_note": note,
			"version":         gorm.Expr("version + 1"),
		}).Error
}

func completeConsultationTodo(db *gorm.DB, consultationID uint, now time.Time, note string) error {
	return db.Model(&caseworkmodel.TodoItem{}).
		Where(
			"source_type = ? AND source_id = ? AND active_slot = ?",
			caseworkmodel.TodoSourceConsultation,
			consultationID,
			caseworkmodel.TodoActiveSlot,
		).
		Updates(map[string]any{
			"status":          caseworkmodel.TodoStatusCompleted,
			"active_slot":     nil,
			"completed_at":    now,
			"completion_note": note,
			"version":         gorm.Expr("version + 1"),
		}).Error
}

func appendConsultationEvent(
	db *gorm.DB,
	consultation caseworkmodel.Consultation,
	interaction caseworkmodel.ConsultationInteraction,
	eventType string,
) error {
	return platformoutbox.Append(db, platformoutbox.AppendInput{
		EventType:     eventType,
		AggregateType: "Consultation",
		AggregateID:   consultation.ID,
		Payload: map[string]any{
			"consultationId": consultation.ID,
			"careClientId":   consultation.CareClientID,
			"interactionId":  interaction.ID,
			"actionType":     interaction.ActionType,
			"actorType":      interaction.ActorType,
			"actorId":        interaction.ActorID,
			"actorRole":      interaction.ActorRole,
			"fromStatus":     interaction.FromStatus,
			"toStatus":       interaction.ToStatus,
			"synthetic":      consultation.Synthetic,
		},
		OccurredAt:  interaction.OccurredAt,
		CausationID: fmt.Sprintf("consultation-interaction:%d", interaction.ID),
		Synthetic:   consultation.Synthetic,
	})
}

func consultationActionResult(
	consultation caseworkmodel.Consultation,
	interaction caseworkmodel.ConsultationInteraction,
) caseworkres.ConsultationActionResult {
	return caseworkres.ConsultationActionResult{
		ConsultationID: consultation.ID,
		InteractionID:  interaction.ID,
		Status:         consultation.Status,
		Version:        consultation.Version,
		OccurredAt:     interaction.OccurredAt,
	}
}

func trimmed(value string) string {
	return strings.TrimSpace(value)
}
