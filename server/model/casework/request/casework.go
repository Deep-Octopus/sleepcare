package request

import "time"

type AcknowledgeCase struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Result          string `json:"result" binding:"required,max=1000"`
}

type HandlingRecord struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	ActionType      string `json:"actionType" binding:"required,oneof=CONTACT HANDLING"`
	Result          string `json:"result" binding:"required,max=4000"`
	NextAction      string `json:"nextAction" binding:"omitempty,max=1000"`
	NextStatus      string `json:"nextStatus" binding:"omitempty"`
}

type EscalateCase struct {
	ExpectedVersion  uint       `json:"expectedVersion" binding:"required"`
	TargetAssigneeID uint       `json:"targetAssigneeId" binding:"required"`
	Reason           string     `json:"reason" binding:"required,max=2000"`
	DueAt            *time.Time `json:"dueAt"`
}

type CloseCase struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	CloseReason     string `json:"closeReason" binding:"required,max=2000"`
}

type ReopenCase struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Reason          string `json:"reason" binding:"required,max=2000"`
}
