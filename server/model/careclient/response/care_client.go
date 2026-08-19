package response

import "time"

type CareAssignmentSummary struct {
	ID                   uint       `json:"id"`
	RoleType             string     `json:"roleType"`
	AssigneeID           uint       `json:"assigneeId"`
	AssigneeDisplayName  string     `json:"assigneeDisplayName"`
	TeamID               uint       `json:"teamId"`
	TeamName             string     `json:"teamName"`
	Status               string     `json:"status"`
	ValidFrom            time.Time  `json:"validFrom"`
	ValidUntil           *time.Time `json:"validUntil"`
	ReplacesAssignmentID *uint      `json:"replacesAssignmentId"`
	Reason               string     `json:"reason"`
	EndReason            string     `json:"endReason"`
}

type ConsentRecordSummary struct {
	ID          uint      `json:"id"`
	ConsentType string    `json:"consentType"`
	Action      string    `json:"action"`
	TextVersion string    `json:"textVersion"`
	OccurredAt  time.Time `json:"occurredAt"`
	Source      string    `json:"source"`
	Reason      string    `json:"reason"`
	RecordedBy  uint      `json:"recordedBy"`
}

type CareClientSummary struct {
	ID                 uint                    `json:"id"`
	DisplayCode        string                  `json:"displayCode"`
	DisplayName        string                  `json:"displayName"`
	ContactMobile      string                  `json:"contactMobile"`
	ServiceReason      string                  `json:"serviceReason"`
	ServicePackageCode string                  `json:"servicePackageCode"`
	OrganizationID     uint                    `json:"organizationId"`
	OrganizationName   string                  `json:"organizationName"`
	TeamID             *uint                   `json:"teamId"`
	TeamName           string                  `json:"teamName"`
	Status             string                  `json:"status"`
	SensitivityLevel   string                  `json:"sensitivityLevel"`
	Synthetic          bool                    `json:"synthetic"`
	Version            uint                    `json:"version"`
	CurrentAssignments []CareAssignmentSummary `json:"currentAssignments"`
}

type CareClientDetail struct {
	CareClientSummary
	Assignments    []CareAssignmentSummary `json:"assignments"`
	ConsentStatus  string                  `json:"consentStatus"`
	ConsentRecords []ConsentRecordSummary  `json:"consentRecords"`
}

type ActionResult struct {
	CareClientID uint `json:"careClientId"`
	ResourceID   uint `json:"resourceId"`
	Version      uint `json:"version"`
}

type OrgUnitOption struct {
	DepartmentID   uint   `json:"departmentId"`
	OrganizationID uint   `json:"organizationId"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	UnitType       string `json:"unitType"`
}

type AssigneeOption struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"displayName"`
	RoleType    string `json:"roleType"`
	TeamID      uint   `json:"teamId"`
}

type ClientOptions struct {
	OrgUnits  []OrgUnitOption  `json:"orgUnits"`
	Assignees []AssigneeOption `json:"assignees"`
}

type ConsentPolicyRequirement struct {
	ConsentType      string `json:"consentType"`
	PolicyVersion    string `json:"policyVersion"`
	ContentReviewed  bool   `json:"contentReviewed"`
	RecordingEnabled bool   `json:"recordingEnabled"`
}

type GovernanceReviewGate struct {
	Key      string `json:"key"`
	Reviewed bool   `json:"reviewed"`
}

type DataGovernanceReadiness struct {
	Mode                      string                     `json:"mode"`
	UsageScope                string                     `json:"usageScope"`
	RealDataEnabled           bool                       `json:"realDataEnabled"`
	FormalConsentEnabled      bool                       `json:"formalConsentEnabled"`
	LifecycleExecutionEnabled bool                       `json:"lifecycleExecutionEnabled"`
	RequestRecordingEnabled   bool                       `json:"requestRecordingEnabled"`
	ConsentRequirements       []ConsentPolicyRequirement `json:"consentRequirements"`
	ReviewGates               []GovernanceReviewGate     `json:"reviewGates"`
	BlockingItems             []string                   `json:"blockingItems"`
}

type DataLifecycleRequestSummary struct {
	ID                         uint      `json:"id"`
	CareClientID               uint      `json:"careClientId"`
	RequestType                string    `json:"requestType"`
	Status                     string    `json:"status"`
	RequestedAt                time.Time `json:"requestedAt"`
	Source                     string    `json:"source"`
	Reason                     string    `json:"reason"`
	IdentityVerificationStatus string    `json:"identityVerificationStatus"`
	GovernanceMode             string    `json:"governanceMode"`
	ExecutionAllowed           bool      `json:"executionAllowed"`
	RecordedAt                 time.Time `json:"recordedAt"`
}
