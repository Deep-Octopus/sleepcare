package response

import "time"

type ClientSatisfactionSummary struct {
	ID            uint       `json:"id"`
	PublicCode    string     `json:"publicCode"`
	SourceType    string     `json:"sourceType"`
	Status        string     `json:"status"`
	AnonymityMode string     `json:"anonymityMode"`
	InvitedAt     time.Time  `json:"invitedAt"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	SubmittedAt   *time.Time `json:"submittedAt"`
	Version       uint       `json:"version"`
}

type ClientSatisfactionResponse struct {
	Rating      uint8     `json:"rating"`
	Comment     string    `json:"comment"`
	SubmittedAt time.Time `json:"submittedAt"`
}

type ClientSatisfactionDetail struct {
	ClientSatisfactionSummary
	Response *ClientSatisfactionResponse `json:"response"`
}

type SubmitSatisfactionResult struct {
	RequestID       uint      `json:"requestId"`
	ResponseID      uint      `json:"responseId"`
	Status          string    `json:"status"`
	Version         uint      `json:"version"`
	FollowUpCreated bool      `json:"followUpCreated"`
	SubmittedAt     time.Time `json:"submittedAt"`
}

type SatisfactionResponseItem struct {
	ID             uint      `json:"id"`
	PublicCode     string    `json:"publicCode"`
	Rating         uint8     `json:"rating"`
	Comment        string    `json:"comment"`
	AnonymityMode  string    `json:"anonymityMode"`
	SubmittedAt    time.Time `json:"submittedAt"`
	FollowUpID     *uint     `json:"followUpId"`
	FollowUpStatus string    `json:"followUpStatus"`
}

type SatisfactionFollowUpSummary struct {
	ID             uint       `json:"id"`
	PublicCode     string     `json:"publicCode"`
	Rating         uint8      `json:"rating"`
	Status         string     `json:"status"`
	AssigneeID     *uint      `json:"assigneeId"`
	AssigneeName   string     `json:"assigneeName"`
	OpenedAt       time.Time  `json:"openedAt"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt"`
	ResolvedAt     *time.Time `json:"resolvedAt"`
	Version        uint       `json:"version"`
}

type SatisfactionFollowUpAction struct {
	ID                     uint      `json:"id"`
	ActionType             string    `json:"actionType"`
	ActorName              string    `json:"actorName"`
	Content                string    `json:"content"`
	ImprovementAction      string    `json:"improvementAction"`
	UsageBoundaryConfirmed bool      `json:"usageBoundaryConfirmed"`
	FromStatus             string    `json:"fromStatus"`
	ToStatus               string    `json:"toStatus"`
	OccurredAt             time.Time `json:"occurredAt"`
}

type SatisfactionFollowUpDetail struct {
	SatisfactionFollowUpSummary
	Comment           string                       `json:"comment"`
	AnonymityMode     string                       `json:"anonymityMode"`
	SubmittedAt       time.Time                    `json:"submittedAt"`
	Resolution        string                       `json:"resolution"`
	ImprovementAction string                       `json:"improvementAction"`
	Actions           []SatisfactionFollowUpAction `json:"actions"`
}

type SatisfactionFollowUpActionResult struct {
	FollowUpID uint      `json:"followUpId"`
	ActionID   uint      `json:"actionId"`
	Status     string    `json:"status"`
	Version    uint      `json:"version"`
	OccurredAt time.Time `json:"occurredAt"`
}
