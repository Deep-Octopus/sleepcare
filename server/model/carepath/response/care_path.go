package response

import "time"

type ReviewRecord struct {
	ReviewType            string     `json:"reviewType"`
	ReviewedBy            uint       `json:"reviewedBy"`
	ReviewedAt            *time.Time `json:"reviewedAt"`
	FormalMedicalApproval bool       `json:"formalMedicalApproval"`
	Note                  string     `json:"note"`
}

type PlanTaskDefinition struct {
	ID                     uint   `json:"id"`
	DayCode                string `json:"dayCode"`
	Title                  string `json:"title"`
	Sort                   int    `json:"sort"`
	ExecutionRole          string `json:"executionRole"`
	OpenOffsetSeconds      int64  `json:"openOffsetSeconds"`
	DueOffsetSeconds       int64  `json:"dueOffsetSeconds"`
	ExpiresOffsetSeconds   *int64 `json:"expiresOffsetSeconds"`
	QuestionnaireVersionID *uint  `json:"questionnaireVersionId"`
	BoundRuleVersionIDs    []uint `json:"boundRuleVersionIds"`
	ReviewRequired         bool   `json:"reviewRequired"`
	ReviewRole             string `json:"reviewRole"`
	NotificationPolicy     string `json:"notificationPolicy"`
}

type PlanVersionSummary struct {
	ID                   uint         `json:"id"`
	PathCode             string       `json:"pathCode"`
	Code                 string       `json:"code"`
	Version              string       `json:"version"`
	Title                string       `json:"title"`
	Purpose              string       `json:"purpose"`
	LifecycleStatus      string       `json:"lifecycleStatus"`
	UsageScope           string       `json:"usageScope"`
	Synthetic            bool         `json:"synthetic"`
	ProductionEnabled    bool         `json:"productionEnabled"`
	ReviewRecord         ReviewRecord `json:"reviewRecord"`
	AnchorDefinition     string       `json:"anchorDefinition"`
	LateSubmissionPolicy string       `json:"lateSubmissionPolicy"`
	PauseStrategy        string       `json:"pauseStrategy"`
	TaskCount            int64        `json:"taskCount"`
	PublishedAt          *time.Time   `json:"publishedAt"`
	DefinitionHash       string       `json:"definitionHash"`
}

type PlanVersionDetail struct {
	PlanVersionSummary
	PathDefinitionVersionID uint                 `json:"pathDefinitionVersionId"`
	DefinitionSchemaVersion string               `json:"definitionSchemaVersion"`
	Tasks                   []PlanTaskDefinition `json:"tasks"`
}

type PreviewTask struct {
	PlanTaskDefinition
	OpenAt    time.Time  `json:"openAt"`
	DueAt     time.Time  `json:"dueAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type PlanPreview struct {
	PreviewID             string        `json:"previewId"`
	CareClientID          uint          `json:"careClientId"`
	PlanTemplateVersionID uint          `json:"planTemplateVersionId"`
	AnchorAt              time.Time     `json:"anchorAt"`
	ExpiresAt             time.Time     `json:"expiresAt"`
	Tasks                 []PreviewTask `json:"tasks"`
}

type PlanInstanceResult struct {
	EnrollmentID   uint      `json:"enrollmentId"`
	PlanInstanceID uint      `json:"planInstanceId"`
	CareClientID   uint      `json:"careClientId"`
	AnchorAt       time.Time `json:"anchorAt"`
	Status         string    `json:"status"`
	TaskIDs        []uint    `json:"taskIds"`
	Version        uint      `json:"version"`
}

type PlanActionResult struct {
	PlanInstanceID uint   `json:"planInstanceId"`
	EnrollmentID   uint   `json:"enrollmentId"`
	Status         string `json:"status"`
	Version        uint   `json:"version"`
}

type TaskActionResult struct {
	ResourceID uint      `json:"resourceId"`
	ActionID   uint      `json:"actionId"`
	Status     string    `json:"status"`
	Version    uint      `json:"version"`
	OccurredAt time.Time `json:"occurredAt"`
}

type TimelineEvent struct {
	EventType  string    `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
	Source     string    `json:"source"`
	Summary    string    `json:"summary"`
}

type TaskSummary struct {
	ID                    uint       `json:"id"`
	PlanInstanceID        uint       `json:"planInstanceId"`
	CareClientID          uint       `json:"careClientId"`
	CareClientDisplayCode string     `json:"careClientDisplayCode"`
	CareClientDisplayName string     `json:"careClientDisplayName"`
	PlanStatus            string     `json:"planStatus"`
	Title                 string     `json:"title"`
	DayCode               string     `json:"dayCode"`
	ExecutionRole         string     `json:"executionRole"`
	ExecutionStatus       string     `json:"executionStatus"`
	TimingStatus          string     `json:"timingStatus"`
	ReviewStatus          string     `json:"reviewStatus"`
	OpenAt                time.Time  `json:"openAt"`
	DueAt                 time.Time  `json:"dueAt"`
	ExpiresAt             *time.Time `json:"expiresAt"`
	SubmittedAt           *time.Time `json:"submittedAt"`
	Version               uint       `json:"version"`
}

type TaskDetail struct {
	TaskSummary
	QuestionnaireVersionID *uint           `json:"questionnaireVersionId"`
	RuleVersionIDs         []uint          `json:"ruleVersionIds"`
	LateSubmissionPolicy   string          `json:"lateSubmissionPolicy"`
	NotificationPolicy     string          `json:"notificationPolicy"`
	ReviewRole             string          `json:"reviewRole"`
	Timeline               []TimelineEvent `json:"timeline"`
}

type PlanInstanceSummary struct {
	ID                    uint            `json:"id"`
	EnrollmentID          uint            `json:"enrollmentId"`
	CareClientID          uint            `json:"careClientId"`
	PlanTemplateVersionID uint            `json:"planTemplateVersionId"`
	TemplateTitle         string          `json:"templateTitle"`
	PathCode              string          `json:"pathCode"`
	AnchorAt              time.Time       `json:"anchorAt"`
	Status                string          `json:"status"`
	PauseStrategy         string          `json:"pauseStrategy"`
	PausedAt              *time.Time      `json:"pausedAt"`
	Version               uint            `json:"version"`
	Synthetic             bool            `json:"synthetic"`
	Tasks                 []TaskSummary   `json:"tasks"`
	Timeline              []TimelineEvent `json:"timeline"`
}
