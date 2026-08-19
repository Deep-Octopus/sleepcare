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
	ProviderCode          string          `json:"providerCode"`
	DispatchPolicyCode    string          `json:"dispatchPolicyCode"`
	DispatchPolicyVersion int             `json:"dispatchPolicyVersion"`
	TemplateCode          string          `json:"templateCode"`
	EstimatedCostMinor    int64           `json:"estimatedCostMinor"`
	CostCurrency          string          `json:"costCurrency"`
	Version               uint            `json:"version"`
	Events                []DeliveryEvent `json:"events"`
}

type RetryBoundary struct {
	MaxAttemptsPerRequest int `json:"maxAttemptsPerRequest"`
}

type RateBoundary struct {
	WindowSeconds int   `json:"windowSeconds"`
	MaxDispatches int64 `json:"maxDispatches"`
}

type CostBoundary struct {
	Currency            string `json:"currency"`
	EstimatedCostMinor  int64  `json:"estimatedCostMinor"`
	DailyCostLimitMinor int64  `json:"dailyCostLimitMinor"`
}

type ProviderReadiness struct {
	Mode                           string        `json:"mode"`
	ProviderCode                   string        `json:"providerCode"`
	UsageScope                     string        `json:"usageScope"`
	ContractTestEnabled            bool          `json:"contractTestEnabled"`
	CallbackEndpointEnabled        bool          `json:"callbackEndpointEnabled"`
	NetworkDeliveryEnabled         bool          `json:"networkDeliveryEnabled"`
	FormalDeliveryEnabled          bool          `json:"formalDeliveryEnabled"`
	QualificationEvidenceReviewed  bool          `json:"qualificationEvidenceReviewed"`
	TemplateEvidenceReviewed       bool          `json:"templateEvidenceReviewed"`
	RequestSigningConfigured       bool          `json:"requestSigningConfigured"`
	CallbackVerificationConfigured bool          `json:"callbackVerificationConfigured"`
	ReceiptSemanticsReviewed       bool          `json:"receiptSemanticsReviewed"`
	RetryPolicyReviewed            bool          `json:"retryPolicyReviewed"`
	FallbackReviewed               bool          `json:"fallbackReviewed"`
	CostBoundaryReviewed           bool          `json:"costBoundaryReviewed"`
	RetryBoundary                  RetryBoundary `json:"retryBoundary"`
	RateBoundary                   RateBoundary  `json:"rateBoundary"`
	CostBoundary                   CostBoundary  `json:"costBoundary"`
	Blockers                       []string      `json:"blockers"`
}
