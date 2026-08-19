package careclient

const (
	ClientStatusActive   = "ACTIVE"
	ClientStatusInactive = "INACTIVE"

	SensitivityBasic           = "BASIC"
	SensitivitySensitive       = "SENSITIVE"
	SensitivityHighlySensitive = "HIGHLY_SENSITIVE"

	AssignmentRoleCareSteward = "CARE_STEWARD"
	AssignmentRoleClinician   = "CLINICIAN"

	AssignmentStatusScheduled = "SCHEDULED"
	AssignmentStatusActive    = "ACTIVE"
	AssignmentStatusEnded     = "ENDED"
	AssignmentStatusCancelled = "CANCELLED"

	ConsentTypeSyntheticTestParticipation = "SYNTHETIC_TEST_PARTICIPATION"
	ConsentActionGrant                    = "GRANT"
	ConsentActionWithdraw                 = "WITHDRAW"
	ConsentSourceClientSelf               = "CLIENT_SELF"
	ConsentSourceStaffRecorded            = "STAFF_RECORDED"

	ConsentStatusPending   = "PENDING"
	ConsentStatusGranted   = "GRANTED"
	ConsentStatusWithdrawn = "WITHDRAWN"

	DataGovernanceModeDisabled     = "DISABLED"
	DataGovernanceModeContractTest = "CONTRACT_TEST"
	DataGovernanceUsageTestOnly    = "TEST_ONLY"

	ConsentRequirementServiceNotice         = "SERVICE_NOTICE"
	ConsentRequirementPrivacyNotice         = "PRIVACY_NOTICE"
	ConsentRequirementNotification          = "NOTIFICATION_CONSENT"
	ConsentRequirementAIProcessing          = "AI_PROCESSING_CONSENT"
	LifecycleRequestAccessCopy              = "ACCESS_COPY"
	LifecycleRequestCorrection              = "CORRECTION"
	LifecycleRequestRestriction             = "RESTRICTION"
	LifecycleRequestErasure                 = "ERASURE"
	LifecycleRequestStatusPendingPolicy     = "PENDING_POLICY"
	LifecycleRequestSourceStaffRecorded     = "STAFF_RECORDED"
	IdentityVerificationStatusNotConfigured = "NOT_CONFIGURED"

	OrgUnitTypeOrganization = "ORGANIZATION"
	OrgUnitTypeTeam         = "TEAM"

	AuthorityRoleCareSteward  = "CARE_STEWARD"
	AuthorityRoleClinician    = "CLINICIAN"
	AuthorityRoleSupervisor   = "SUPERVISOR"
	AuthorityRoleContentAdmin = "CONTENT_ADMIN"

	DataLevelBasic     = "BASIC"
	DataLevelSensitive = "SENSITIVE"
)
