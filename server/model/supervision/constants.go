package supervision

const (
	SummaryTypeRealtimePreview   = "REALTIME_PREVIEW"
	SummaryTypeVersionedSnapshot = "VERSIONED_SNAPSHOT"
	MetricDefinitionVersionV1    = "P1-08-v1"

	GuidanceActionGuidance  = "GUIDANCE"
	GuidanceActionIntervene = "INTERVENE"

	ReviewStatusPending    = "PENDING"
	ReviewStatusGuided     = "GUIDED"
	ReviewStatusIntervened = "INTERVENED"
	ReviewStatusCompleted  = "COMPLETED"

	SatisfactionPolicyStatusPublished = "PUBLISHED"
	SatisfactionTriggerConsultation   = "CONSULTATION_CLOSED"
	SatisfactionSourceConsultation    = "CONSULTATION"
	SatisfactionAnonymousStaff        = "STAFF_ANONYMOUS_SYSTEM_LINKED"

	SatisfactionRequestPending   = "PENDING"
	SatisfactionRequestSubmitted = "SUBMITTED"
	SatisfactionRequestExpired   = "EXPIRED"

	SatisfactionFollowUpOpen     = "OPEN"
	SatisfactionFollowUpInReview = "IN_REVIEW"
	SatisfactionFollowUpResolved = "RESOLVED"

	SatisfactionFollowUpActionAcknowledge = "ACKNOWLEDGE"
	SatisfactionFollowUpActionResolve     = "RESOLVE"

	EventSatisfactionRequested            = "SatisfactionRequested"
	EventSatisfactionExpired              = "SatisfactionExpired"
	EventSatisfactionSubmitted            = "SatisfactionSubmitted"
	EventSatisfactionFollowUpOpened       = "SatisfactionFollowUpOpened"
	EventSatisfactionFollowUpAcknowledged = "SatisfactionFollowUpAcknowledged"
	EventSatisfactionFollowUpResolved     = "SatisfactionFollowUpResolved"
)
