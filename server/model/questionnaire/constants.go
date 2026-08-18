package questionnaire

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

	QuestionTypeSingleChoice   = "SINGLE_CHOICE"
	QuestionTypeMultipleChoice = "MULTIPLE_CHOICE"
	QuestionTypeText           = "TEXT"
	QuestionTypeNumber         = "NUMBER"
	QuestionTypeDate           = "DATE"
	QuestionTypeBoolean        = "BOOLEAN"

	SubmissionSourceClientSelf    = "CLIENT_SELF"
	SubmissionSourceStaffAssisted = "STAFF_ASSISTED"
	ActorKindClient               = "CLIENT"
	ActorKindStaff                = "STAFF"

	RuleOperatorEquals   = "EQUALS"
	RuleOperatorContains = "CONTAINS"

	EventTaskAnswerSubmitted = "TaskAnswerSubmitted"
	EventRuleHitRecorded     = "RuleHitRecorded"
)

var questionTypes = map[string]struct{}{
	QuestionTypeSingleChoice:   {},
	QuestionTypeMultipleChoice: {},
	QuestionTypeText:           {},
	QuestionTypeNumber:         {},
	QuestionTypeDate:           {},
	QuestionTypeBoolean:        {},
}

func IsQuestionType(value string) bool {
	_, ok := questionTypes[value]
	return ok
}

func CanTransitionLifecycle(from, to string) bool {
	allowed := map[string]string{
		LifecycleDraft:     LifecycleInReview,
		LifecycleInReview:  LifecycleApproved,
		LifecycleApproved:  LifecyclePublished,
		LifecyclePublished: LifecycleDisabled,
	}
	return allowed[from] == to
}
