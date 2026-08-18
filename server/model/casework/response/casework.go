package response

import "time"

type ActionResult struct {
	ResourceID uint      `json:"resourceId"`
	ActionID   uint      `json:"actionId"`
	Status     string    `json:"status"`
	Version    uint      `json:"version"`
	OccurredAt time.Time `json:"occurredAt"`
}

type AttentionCaseSummary struct {
	ID             uint       `json:"id"`
	CareClientID   uint       `json:"careClientId"`
	TaskID         uint       `json:"taskId"`
	Status         string     `json:"status"`
	AttentionLevel string     `json:"attentionLevel"`
	ReasonSummary  string     `json:"reasonSummary"`
	AssigneeID     *uint      `json:"assigneeId"`
	OpenedAt       time.Time  `json:"openedAt"`
	DueAt          *time.Time `json:"dueAt"`
	Version        uint       `json:"version"`
}

type RuleHitSummary struct {
	ID             uint      `json:"id"`
	RuleVersionID  uint      `json:"ruleVersionId"`
	ReasonSnapshot string    `json:"reasonSnapshot"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type CaseAction struct {
	ID         uint      `json:"id"`
	ActionType string    `json:"actionType"`
	ActorRole  string    `json:"actorRole"`
	Result     string    `json:"result"`
	Reason     *string   `json:"reason"`
	OccurredAt time.Time `json:"occurredAt"`
}

type AttentionCaseDetail struct {
	AttentionCaseSummary
	RuleHits       []RuleHitSummary `json:"ruleHits"`
	Actions        []CaseAction     `json:"actions"`
	HandlingResult *string          `json:"handlingResult"`
	CloseReason    *string          `json:"closeReason"`
}
