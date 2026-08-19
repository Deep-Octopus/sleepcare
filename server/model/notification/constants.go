package notification

const (
	ChannelDemo             = "DEMO"
	ChannelProviderContract = "PROVIDER_CONTRACT"

	ProviderModeDisabled     = "DISABLED"
	ProviderModeContractTest = "CONTRACT_TEST"
	ProviderUsageTestOnly    = "TEST_ONLY"

	QuotaBucketRate = "RATE"
	QuotaBucketCost = "COST"

	DispatchReservationReserved  = "RESERVED"
	DispatchReservationSubmitted = "SUBMITTED"
	DispatchReservationFinalized = "FINALIZED"

	AttemptStatusPending             = "PENDING"
	AttemptStatusSubmittedToProvider = "SUBMITTED_TO_PROVIDER"
	AttemptStatusAccepted            = "ACCEPTED"
	AttemptStatusDelivered           = "DELIVERED"
	AttemptStatusFailed              = "FAILED"
	AttemptStatusUnknown             = "UNKNOWN"

	EventNotificationRequested = "NotificationRequested"
	EventNotificationSubmitted = "NotificationSubmittedToProvider"
	EventNotificationAccepted  = "NotificationAccepted"
	EventNotificationDelivered = "NotificationDelivered"
	EventNotificationFailed    = "NotificationFailed"
	EventNotificationUnknown   = "NotificationUnknown"

	TodoSourceNotificationRequest = "NOTIFICATION_REQUEST"

	DemoFailureCode = "DEMO_REJECTED"
	DemoUnknownCode = "RECEIPT_TIMEOUT"

	ProviderFailureRejected    = "PROVIDER_REJECTED"
	ProviderFailureUnavailable = "DELIVERY_UNAVAILABLE"
)

func IsAttemptStatus(value string) bool {
	switch value {
	case AttemptStatusPending,
		AttemptStatusSubmittedToProvider,
		AttemptStatusAccepted,
		AttemptStatusDelivered,
		AttemptStatusFailed,
		AttemptStatusUnknown:
		return true
	default:
		return false
	}
}

func IsFinalAttemptStatus(value string) bool {
	switch value {
	case AttemptStatusDelivered, AttemptStatusFailed, AttemptStatusUnknown:
		return true
	default:
		return false
	}
}

func CanTransitionAttempt(from, to string) bool {
	switch from {
	case AttemptStatusPending:
		return to == AttemptStatusSubmittedToProvider
	case AttemptStatusSubmittedToProvider:
		return to == AttemptStatusAccepted
	case AttemptStatusAccepted:
		return IsFinalAttemptStatus(to)
	default:
		return false
	}
}

func EventTypeForStatus(status string) string {
	switch status {
	case AttemptStatusPending:
		return EventNotificationRequested
	case AttemptStatusSubmittedToProvider:
		return EventNotificationSubmitted
	case AttemptStatusAccepted:
		return EventNotificationAccepted
	case AttemptStatusDelivered:
		return EventNotificationDelivered
	case AttemptStatusFailed:
		return EventNotificationFailed
	case AttemptStatusUnknown:
		return EventNotificationUnknown
	default:
		return ""
	}
}

func IsProviderCallbackStatus(value string) bool {
	switch value {
	case AttemptStatusAccepted,
		AttemptStatusDelivered,
		AttemptStatusFailed,
		AttemptStatusUnknown:
		return true
	default:
		return false
	}
}

func IsNormalizedProviderFailureCode(value string) bool {
	switch value {
	case ProviderFailureRejected, ProviderFailureUnavailable, DemoUnknownCode:
		return true
	default:
		return false
	}
}
