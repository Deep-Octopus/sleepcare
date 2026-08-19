package response

import "time"

type ConsultationActionResult struct {
	ConsultationID uint      `json:"consultationId"`
	InteractionID  uint      `json:"interactionId"`
	Status         string    `json:"status"`
	Version        uint      `json:"version"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type ConsultationSummary struct {
	ID                uint       `json:"id"`
	CareClientID      uint       `json:"careClientId"`
	ClientDisplayCode string     `json:"clientDisplayCode"`
	ClientDisplayName string     `json:"clientDisplayName"`
	Subject           string     `json:"subject"`
	Source            string     `json:"source"`
	Urgency           string     `json:"urgency"`
	Status            string     `json:"status"`
	AssigneeID        *uint      `json:"assigneeId"`
	AssigneeRole      string     `json:"assigneeRole"`
	AssigneeName      string     `json:"assigneeName"`
	OpenedAt          time.Time  `json:"openedAt"`
	FirstRespondedAt  *time.Time `json:"firstRespondedAt"`
	ResolvedAt        *time.Time `json:"resolvedAt"`
	ClosedAt          *time.Time `json:"closedAt"`
	Version           uint       `json:"version"`
}

type ConsultationInteraction struct {
	ID         uint      `json:"id"`
	ActionType string    `json:"actionType"`
	ActorType  string    `json:"actorType"`
	ActorRole  string    `json:"actorRole"`
	ActorName  string    `json:"actorName"`
	Content    string    `json:"content"`
	Reason     *string   `json:"reason"`
	FromStatus string    `json:"fromStatus"`
	ToStatus   string    `json:"toStatus"`
	TargetRole string    `json:"targetRole"`
	OccurredAt time.Time `json:"occurredAt"`
}

type ConsultationDetail struct {
	ConsultationSummary
	InitialQuestion string                    `json:"initialQuestion"`
	Resolution      *string                   `json:"resolution"`
	FollowUpPlan    *string                   `json:"followUpPlan"`
	CloseReason     *string                   `json:"closeReason"`
	Interactions    []ConsultationInteraction `json:"interactions"`
}

type ConsultationAssigneeOption struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"displayName"`
	RoleType    string `json:"roleType"`
}

type ClientConsultationSummary struct {
	ID               uint       `json:"id"`
	Subject          string     `json:"subject"`
	Urgency          string     `json:"urgency"`
	Status           string     `json:"status"`
	OpenedAt         time.Time  `json:"openedAt"`
	FirstRespondedAt *time.Time `json:"firstRespondedAt"`
	ResolvedAt       *time.Time `json:"resolvedAt"`
	ClosedAt         *time.Time `json:"closedAt"`
	Version          uint       `json:"version"`
}

type ClientConsultationInteraction struct {
	ID         uint      `json:"id"`
	SenderType string    `json:"senderType"`
	Content    string    `json:"content"`
	OccurredAt time.Time `json:"occurredAt"`
}

type ClientConsultationDetail struct {
	ClientConsultationSummary
	InitialQuestion string                          `json:"initialQuestion"`
	Resolution      *string                         `json:"resolution"`
	FollowUpPlan    *string                         `json:"followUpPlan"`
	Interactions    []ClientConsultationInteraction `json:"interactions"`
}
