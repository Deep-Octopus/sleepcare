package carepath

const (
	LifecycleDraft     = "DRAFT"
	LifecycleInReview  = "IN_REVIEW"
	LifecycleApproved  = "APPROVED"
	LifecyclePublished = "PUBLISHED"
	LifecycleDisabled  = "DISABLED"

	UsageScopeTestOnly = "TEST_ONLY"
	UsageScopeFormal   = "FORMAL"

	ReviewTypeEngineering = "ENGINEERING"
	ReviewTypeFormal      = "FORMAL"

	EnrollmentDraft        = "DRAFT"
	EnrollmentPendingStart = "PENDING_START"
	EnrollmentActive       = "ACTIVE"
	EnrollmentPaused       = "PAUSED"
	EnrollmentCompleted    = "COMPLETED"
	EnrollmentTerminated   = "TERMINATED"

	ExecutionRoleCareClient  = "CARE_CLIENT"
	ExecutionRoleCareSteward = "CARE_STEWARD"
	ExecutionRoleClinician   = "CLINICIAN"

	ExecutionScheduled  = "SCHEDULED"
	ExecutionOpen       = "OPEN"
	ExecutionInProgress = "IN_PROGRESS"
	ExecutionSubmitted  = "SUBMITTED"
	ExecutionCancelled  = "CANCELLED"

	TimingNotOpen      = "NOT_OPEN"
	TimingWithinWindow = "WITHIN_WINDOW"
	TimingOverdue      = "OVERDUE"
	TimingExpired      = "EXPIRED"

	ReviewNotReady    = "NOT_READY"
	ReviewNotRequired = "NOT_REQUIRED"
	ReviewPending     = "PENDING"
	ReviewReviewing   = "REVIEWING"
	ReviewReviewed    = "REVIEWED"
	ReviewReturned    = "RETURNED"

	LateSubmissionDeny                 = "DENY"
	LateSubmissionAllowUntilExpires    = "ALLOW_UNTIL_EXPIRES"
	LateSubmissionReviewRequired       = "REVIEW_REQUIRED"
	PauseStrategyKeepWindows           = "KEEP_WINDOWS"
	NotificationPolicyDisabled         = "DISABLED"
	AnchorFirstValidSyntheticDeviceUse = "FIRST_VALID_SYNTHETIC_DEVICE_USE_AT"

	EventPlanStarted         = "CarePlanStarted"
	EventPlanPaused          = "CarePlanPaused"
	EventPlanResumed         = "CarePlanResumed"
	EventTaskOpened          = "TaskOpened"
	EventClientTaskOpened    = "ClientTaskOpened"
	EventClientTaskConsented = "ClientTaskConsented"
	EventTaskAnswerStarted   = "TaskAnswerStarted"
	EventTaskAnswerSubmitted = "TaskAnswerSubmitted"

	EventSourceSystem      = "SYSTEM"
	EventSourceCareSteward = "CARE_STEWARD"
	EventSourceClinician   = "CLINICIAN"
	EventSourceSupervisor  = "SUPERVISOR"
	EventSourceClient      = "CLIENT"
)

func CanTransitionEnrollment(from, to string) bool {
	return (from == EnrollmentActive && to == EnrollmentPaused) ||
		(from == EnrollmentPaused && to == EnrollmentActive)
}
