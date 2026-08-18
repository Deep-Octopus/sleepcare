package casework

const (
	CaseStatusPendingAck           = "PENDING_ACK"
	CaseStatusAcknowledged         = "ACKNOWLEDGED"
	CaseStatusHandling             = "HANDLING"
	CaseStatusWaitingClient        = "WAITING_CLIENT"
	CaseStatusWaitingCollaboration = "WAITING_COLLABORATION"
	CaseStatusWaitingSupervisor    = "WAITING_SUPERVISOR"
	CaseStatusResolved             = "RESOLVED"
	CaseStatusClosed               = "CLOSED"

	CaseSourceRuleHit = "RULE_HIT"

	CaseActionAcknowledge = "ACKNOWLEDGE"
	CaseActionContact     = "CONTACT"
	CaseActionHandling    = "HANDLING"
	CaseActionEscalate    = "ESCALATE"
	CaseActionGuidance    = "GUIDANCE"
	CaseActionIntervene   = "INTERVENE"
	CaseActionResolve     = "RESOLVE"
	CaseActionClose       = "CLOSE"
	CaseActionReopen      = "REOPEN"

	ActionSourceSystem = "SYSTEM"
	ActionSourceStaff  = "STAFF"

	TodoCategoryContentAttention = "CONTENT_ATTENTION"
	TodoSourceAttentionCase      = "ATTENTION_CASE"
	TodoStatusOpen               = "OPEN"
	TodoStatusCompleted          = "COMPLETED"
	TodoStatusSuperseded         = "SUPERSEDED"
	TodoActiveSlot               = "ACTIVE"

	EventAttentionCaseOpened       = "AttentionCaseOpened"
	EventAttentionCaseAcknowledged = "AttentionCaseAcknowledged"
	EventAttentionHandlingRecorded = "AttentionHandlingRecorded"
	EventAttentionCaseEscalated    = "AttentionCaseEscalated"
	EventAttentionCaseResolved     = "AttentionCaseResolved"
	EventAttentionCaseClosed       = "AttentionCaseClosed"
	EventAttentionCaseReopened     = "AttentionCaseReopened"
)
