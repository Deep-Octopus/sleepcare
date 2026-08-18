package carepath

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

// Definition tables are global immutable configuration and intentionally do
// not carry row-level ownership columns.
type PathDefinitionVersion struct {
	global.GVA_MODEL
	Code              string     `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_care_path_code_version,priority:1;not null"`
	Version           string     `json:"version" gorm:"type:varchar(32);uniqueIndex:idx_care_path_code_version,priority:2;not null"`
	Title             string     `json:"title" gorm:"type:varchar(160);not null"`
	Purpose           string     `json:"purpose" gorm:"type:varchar(500);not null"`
	Status            string     `json:"status" gorm:"type:varchar(24);index;not null"`
	UsageScope        string     `json:"usageScope" gorm:"type:varchar(24);index;not null"`
	Synthetic         bool       `json:"synthetic" gorm:"index;not null;default:false"`
	ProductionEnabled bool       `json:"productionEnabled" gorm:"not null;default:false"`
	ReviewType        string     `json:"reviewType" gorm:"type:varchar(24);not null"`
	ReviewedBy        uint       `json:"reviewedBy" gorm:"index"`
	ReviewedAt        *time.Time `json:"reviewedAt"`
	ReviewNote        string     `json:"reviewNote" gorm:"type:varchar(500)"`
	PublishedAt       *time.Time `json:"publishedAt" gorm:"index"`
	DefinitionHash    string     `json:"definitionHash" gorm:"type:char(64);not null"`
	RowVersion        uint       `json:"rowVersion" gorm:"not null;default:1"`
}

func (PathDefinitionVersion) TableName() string { return "care_path_definition_versions" }

type PlanTemplateVersion struct {
	global.GVA_MODEL
	PathDefinitionVersionID uint       `json:"pathDefinitionVersionId" gorm:"index;not null"`
	Code                    string     `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_care_plan_code_version,priority:1;not null"`
	Version                 string     `json:"version" gorm:"type:varchar(32);uniqueIndex:idx_care_plan_code_version,priority:2;not null"`
	Title                   string     `json:"title" gorm:"type:varchar(160);not null"`
	Purpose                 string     `json:"purpose" gorm:"type:varchar(500);not null"`
	Status                  string     `json:"status" gorm:"type:varchar(24);index;not null"`
	UsageScope              string     `json:"usageScope" gorm:"type:varchar(24);index;not null"`
	Synthetic               bool       `json:"synthetic" gorm:"index;not null;default:false"`
	ProductionEnabled       bool       `json:"productionEnabled" gorm:"not null;default:false"`
	ReviewType              string     `json:"reviewType" gorm:"type:varchar(24);not null"`
	ReviewedBy              uint       `json:"reviewedBy" gorm:"index"`
	ReviewedAt              *time.Time `json:"reviewedAt"`
	ReviewNote              string     `json:"reviewNote" gorm:"type:varchar(500)"`
	PublishedAt             *time.Time `json:"publishedAt" gorm:"index"`
	AnchorDefinition        string     `json:"anchorDefinition" gorm:"type:varchar(80);not null"`
	LateSubmissionPolicy    string     `json:"lateSubmissionPolicy" gorm:"type:varchar(32);not null"`
	PauseStrategy           string     `json:"pauseStrategy" gorm:"type:varchar(32);not null"`
	DefinitionSchemaVersion string     `json:"definitionSchemaVersion" gorm:"type:varchar(16);not null;default:v1"`
	DefinitionHash          string     `json:"definitionHash" gorm:"type:char(64);not null"`
	RowVersion              uint       `json:"rowVersion" gorm:"not null;default:1"`
}

func (PlanTemplateVersion) TableName() string { return "care_plan_template_versions" }

type PlanTaskDefinition struct {
	global.GVA_MODEL
	PlanTemplateVersionID   uint           `json:"planTemplateVersionId" gorm:"uniqueIndex:idx_care_plan_task_day,priority:1;index;not null"`
	DayCode                 string         `json:"dayCode" gorm:"type:varchar(16);uniqueIndex:idx_care_plan_task_day,priority:2;not null"`
	Title                   string         `json:"title" gorm:"type:varchar(160);not null"`
	Sort                    int            `json:"sort" gorm:"index;not null"`
	ExecutionRole           string         `json:"executionRole" gorm:"type:varchar(32);index;not null"`
	OpenOffsetSeconds       int64          `json:"openOffsetSeconds" gorm:"not null"`
	DueOffsetSeconds        int64          `json:"dueOffsetSeconds" gorm:"not null"`
	ExpiresOffsetSeconds    *int64         `json:"expiresOffsetSeconds"`
	QuestionnaireVersionID  *uint          `json:"questionnaireVersionId" gorm:"index"`
	BoundRuleVersionIDsJSON datatypes.JSON `json:"boundRuleVersionIds" gorm:"type:json;not null" swaggertype:"array,integer"`
	ReviewRequired          bool           `json:"reviewRequired" gorm:"not null;default:false"`
	ReviewRole              string         `json:"reviewRole" gorm:"type:varchar(32)"`
	NotificationPolicy      string         `json:"notificationPolicy" gorm:"type:varchar(32);not null"`
}

func (PlanTaskDefinition) TableName() string { return "care_plan_task_definitions" }

type PlanTaskDependency struct {
	global.GVA_MODEL
	PlanTemplateVersionID        uint   `json:"planTemplateVersionId" gorm:"index;not null"`
	TaskDefinitionID             uint   `json:"taskDefinitionId" gorm:"uniqueIndex:idx_care_plan_dependency,priority:1;not null"`
	PrerequisiteTaskDefinitionID uint   `json:"prerequisiteTaskDefinitionId" gorm:"uniqueIndex:idx_care_plan_dependency,priority:2;not null"`
	Policy                       string `json:"policy" gorm:"type:varchar(32);not null"`
}

func (PlanTaskDependency) TableName() string { return "care_plan_task_dependencies" }

// Runtime tables are owned by the care client's team and therefore carry the
// standard DataScope ownership fields.
type Enrollment struct {
	global.GVA_MODEL
	CareClientID            uint       `json:"careClientId" gorm:"uniqueIndex:idx_active_care_path,priority:1;index;not null"`
	PathDefinitionVersionID uint       `json:"pathDefinitionVersionId" gorm:"index;not null"`
	PathCode                string     `json:"pathCode" gorm:"type:varchar(64);uniqueIndex:idx_active_care_path,priority:2;not null"`
	ActiveSlot              *string    `json:"activeSlot" gorm:"type:varchar(64);uniqueIndex:idx_active_care_path,priority:3"`
	Status                  string     `json:"status" gorm:"type:varchar(24);index;not null"`
	StartedAt               *time.Time `json:"startedAt"`
	EndedAt                 *time.Time `json:"endedAt"`
	Version                 uint       `json:"version" gorm:"not null;default:1"`
	Synthetic               bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                  uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy               uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy               uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy               uint       `json:"-" gorm:"column:deleted_by"`
}

func (Enrollment) TableName() string { return "care_path_enrollments" }

type PlanPreview struct {
	global.GVA_MODEL
	PreviewID              string     `json:"previewId" gorm:"type:varchar(120);uniqueIndex;not null"`
	CareClientID           uint       `json:"careClientId" gorm:"index;not null"`
	PlanTemplateVersionID  uint       `json:"planTemplateVersionId" gorm:"index;not null"`
	AnchorAt               time.Time  `json:"anchorAt" gorm:"index;not null"`
	ExpiresAt              time.Time  `json:"expiresAt" gorm:"index;not null"`
	TemplateDefinitionHash string     `json:"templateDefinitionHash" gorm:"type:char(64);not null"`
	ConsumedAt             *time.Time `json:"consumedAt"`
	PlanInstanceID         *uint      `json:"planInstanceId" gorm:"index"`
	Synthetic              bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                 uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy              uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy              uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy              uint       `json:"-" gorm:"column:deleted_by"`
}

func (PlanPreview) TableName() string { return "care_plan_previews" }

type PlanInstance struct {
	global.GVA_MODEL
	EnrollmentID          uint       `json:"enrollmentId" gorm:"uniqueIndex;not null"`
	CareClientID          uint       `json:"careClientId" gorm:"index;not null"`
	PlanTemplateVersionID uint       `json:"planTemplateVersionId" gorm:"index;not null"`
	PreviewID             uint       `json:"previewId" gorm:"uniqueIndex;not null"`
	AnchorAt              time.Time  `json:"anchorAt" gorm:"index;not null"`
	Status                string     `json:"status" gorm:"type:varchar(24);index;not null"`
	PauseStrategy         string     `json:"pauseStrategy" gorm:"type:varchar(32);not null"`
	PausedAt              *time.Time `json:"pausedAt"`
	Version               uint       `json:"version" gorm:"not null;default:1"`
	Synthetic             bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy             uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy             uint       `json:"-" gorm:"column:deleted_by"`
}

func (PlanInstance) TableName() string { return "care_plan_instances" }

type TaskInstance struct {
	global.GVA_MODEL
	PlanInstanceID          uint           `json:"planInstanceId" gorm:"uniqueIndex:idx_care_task_plan_definition,priority:1;index;not null"`
	CareClientID            uint           `json:"careClientId" gorm:"index;not null"`
	TaskDefinitionID        uint           `json:"taskDefinitionId" gorm:"uniqueIndex:idx_care_task_plan_definition,priority:2;not null"`
	DayCode                 string         `json:"dayCode" gorm:"type:varchar(16);index;not null"`
	Title                   string         `json:"title" gorm:"type:varchar(160);not null"`
	Sort                    int            `json:"sort" gorm:"index;not null"`
	ExecutionRole           string         `json:"executionRole" gorm:"type:varchar(32);index;not null"`
	ExecutionStatus         string         `json:"executionStatus" gorm:"type:varchar(24);index;not null"`
	ReviewStatus            string         `json:"reviewStatus" gorm:"type:varchar(24);index;not null"`
	ReviewRole              string         `json:"reviewRole" gorm:"type:varchar(32)"`
	OpenAt                  time.Time      `json:"openAt" gorm:"index;not null"`
	DueAt                   time.Time      `json:"dueAt" gorm:"index;not null"`
	ExpiresAt               *time.Time     `json:"expiresAt" gorm:"index"`
	QuestionnaireVersionID  *uint          `json:"questionnaireVersionId" gorm:"index"`
	BoundRuleVersionIDsJSON datatypes.JSON `json:"boundRuleVersionIds" gorm:"type:json;not null" swaggertype:"array,integer"`
	LateSubmissionPolicy    string         `json:"lateSubmissionPolicy" gorm:"type:varchar(32);not null"`
	NotificationPolicy      string         `json:"notificationPolicy" gorm:"type:varchar(32);not null"`
	OpenedAt                *time.Time     `json:"openedAt"`
	SubmittedAt             *time.Time     `json:"submittedAt"`
	Version                 uint           `json:"version" gorm:"not null;default:1"`
	Synthetic               bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                  uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy               uint           `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy               uint           `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy               uint           `json:"-" gorm:"column:deleted_by"`
}

func (TaskInstance) TableName() string { return "care_task_instances" }

func (t TaskInstance) TimingStatus(now time.Time) string {
	if now.Before(t.OpenAt) {
		return TimingNotOpen
	}
	if t.ExpiresAt != nil && !now.Before(*t.ExpiresAt) {
		return TimingExpired
	}
	if !now.Before(t.DueAt) {
		if t.LateSubmissionPolicy == LateSubmissionDeny {
			return TimingExpired
		}
		return TimingOverdue
	}
	return TimingWithinWindow
}

type CarePathEvent struct {
	global.GVA_MODEL
	EventID        string    `json:"eventId" gorm:"type:char(36);uniqueIndex;not null"`
	EventType      string    `json:"eventType" gorm:"type:varchar(64);index;not null"`
	CareClientID   uint      `json:"careClientId" gorm:"index;not null"`
	EnrollmentID   uint      `json:"enrollmentId" gorm:"index;not null"`
	PlanInstanceID uint      `json:"planInstanceId" gorm:"index;not null"`
	TaskInstanceID *uint     `json:"taskInstanceId" gorm:"index"`
	ActorID        uint      `json:"actorId" gorm:"index;not null"`
	Source         string    `json:"source" gorm:"type:varchar(32);not null"`
	Reason         string    `json:"reason" gorm:"type:varchar(1000)"`
	FromStatus     string    `json:"fromStatus" gorm:"type:varchar(24)"`
	ToStatus       string    `json:"toStatus" gorm:"type:varchar(24)"`
	OccurredAt     time.Time `json:"occurredAt" gorm:"index;not null"`
	Synthetic      bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId         uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (CarePathEvent) TableName() string { return "care_path_events" }

type CommandReceipt struct {
	global.GVA_MODEL
	ActorID        uint   `json:"actorId" gorm:"uniqueIndex:idx_care_path_receipt,priority:1;not null"`
	Operation      string `json:"operation" gorm:"type:varchar(64);uniqueIndex:idx_care_path_receipt,priority:2;not null"`
	IdempotencyKey string `json:"idempotencyKey" gorm:"type:varchar(128);uniqueIndex:idx_care_path_receipt,priority:3;not null"`
	RequestHash    string `json:"requestHash" gorm:"type:char(64);not null"`
	ResultJSON     string `json:"-" gorm:"type:text;not null"`
	DeptId         uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint   `json:"createdBy" gorm:"column:created_by;index"`
}

func (CommandReceipt) TableName() string { return "care_path_command_receipts" }
