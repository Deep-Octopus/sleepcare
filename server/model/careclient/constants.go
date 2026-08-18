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

	OrgUnitTypeOrganization = "ORGANIZATION"
	OrgUnitTypeTeam         = "TEAM"

	AuthorityRoleCareSteward  = "CARE_STEWARD"
	AuthorityRoleClinician    = "CLINICIAN"
	AuthorityRoleSupervisor   = "SUPERVISOR"
	AuthorityRoleContentAdmin = "CONTENT_ADMIN"

	DataLevelBasic     = "BASIC"
	DataLevelSensitive = "SENSITIVE"
)
