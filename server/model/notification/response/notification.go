package response

import "time"

type DeliveryEvent struct {
	ID          uint      `json:"id"`
	EventType   string    `json:"eventType"`
	FromStatus  string    `json:"fromStatus"`
	ToStatus    string    `json:"toStatus"`
	OccurredAt  time.Time `json:"occurredAt"`
	FailureCode string    `json:"failureCode"`
}

type NotificationAttempt struct {
	ID                    uint            `json:"id"`
	NotificationRequestID uint            `json:"notificationRequestId"`
	TaskID                uint            `json:"taskId"`
	CareClientID          uint            `json:"careClientId"`
	CareClientDisplayCode string          `json:"careClientDisplayCode"`
	CareClientDisplayName string          `json:"careClientDisplayName"`
	AttemptNo             int             `json:"attemptNo"`
	RetryOfAttemptID      *uint           `json:"retryOfAttemptId"`
	Channel               string          `json:"channel"`
	Status                string          `json:"status"`
	RequestedAt           time.Time       `json:"requestedAt"`
	SubmittedAt           *time.Time      `json:"submittedAt"`
	AcceptedAt            *time.Time      `json:"acceptedAt"`
	DeliveredAt           *time.Time      `json:"deliveredAt"`
	FinalizedAt           *time.Time      `json:"finalizedAt"`
	FailureCode           string          `json:"failureCode"`
	Version               uint            `json:"version"`
	Events                []DeliveryEvent `json:"events"`
}
