package supervision

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type DailySummaryVersion struct {
	global.GVA_MODEL
	OrganizationID          uint           `json:"organizationId" gorm:"uniqueIndex:idx_daily_summary_version,priority:1;index;not null"`
	BusinessDate            time.Time      `json:"businessDate" gorm:"type:date;uniqueIndex:idx_daily_summary_version,priority:2;index;not null"`
	Version                 uint           `json:"version" gorm:"uniqueIndex:idx_daily_summary_version,priority:3;not null"`
	MetricDefinitionVersion string         `json:"metricDefinitionVersion" gorm:"type:varchar(32);not null"`
	GenerationType          string         `json:"generationType" gorm:"type:varchar(32);index;not null;default:LEGACY"`
	GeneratedAt             time.Time      `json:"generatedAt" gorm:"index;not null"`
	SourceCutoffAt          *time.Time     `json:"sourceCutoffAt" gorm:"index"`
	PreviousVersionID       *uint          `json:"previousVersionId" gorm:"index"`
	CorrectionReason        string         `json:"correctionReason" gorm:"type:varchar(1000)"`
	SourceDigest            string         `json:"-" gorm:"type:char(64)"`
	RevisionChangesJSON     datatypes.JSON `json:"-" gorm:"type:json" swaggertype:"array,object"`
	FocusCasesChanged       bool           `json:"focusCasesChanged" gorm:"not null;default:false"`
	CommandActorID          uint           `json:"-" gorm:"index"`
	CommandOperation        string         `json:"-" gorm:"type:varchar(80)"`
	CommandKeyDigest        *string        `json:"-" gorm:"type:char(64);uniqueIndex"`
	CommandRequestHash      string         `json:"-" gorm:"type:char(64)"`
	ServedClients           int64          `json:"servedClients" gorm:"not null;default:0"`
	DueTasks                int64          `json:"dueTasks" gorm:"not null;default:0"`
	SubmittedTasks          int64          `json:"submittedTasks" gorm:"not null;default:0"`
	OverdueTasks            int64          `json:"overdueTasks" gorm:"not null;default:0"`
	DeliveryIssues          int64          `json:"deliveryIssues" gorm:"not null;default:0"`
	OpenAttentionCases      int64          `json:"openAttentionCases" gorm:"not null;default:0"`
	ResolvedAttentionCases  int64          `json:"resolvedAttentionCases" gorm:"not null;default:0"`
	ConsultationsOpened     int64          `json:"consultationsOpened" gorm:"not null;default:0"`
	ConsultationsClosed     int64          `json:"consultationsClosed" gorm:"not null;default:0"`
	OpenConsultations       int64          `json:"openConsultations" gorm:"not null;default:0"`
	OpenTodos               int64          `json:"openTodos" gorm:"not null;default:0"`
	ReviewRequired          int64          `json:"reviewRequired" gorm:"not null;default:0"`
	FocusCasesJSON          datatypes.JSON `json:"-" gorm:"type:json;not null" swaggertype:"array,object"`
	Synthetic               bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                  uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy               uint           `json:"createdBy" gorm:"column:created_by;index"`
}

func (DailySummaryVersion) TableName() string { return "daily_summary_versions" }

type SupervisorGuidance struct {
	global.GVA_MODEL
	AttentionCaseID       uint      `json:"attentionCaseId" gorm:"index;not null"`
	CaseActionID          uint      `json:"caseActionId" gorm:"uniqueIndex;not null"`
	ActionType            string    `json:"actionType" gorm:"type:varchar(24);index;not null"`
	Guidance              string    `json:"guidance" gorm:"type:text;not null"`
	ActorID               uint      `json:"actorId" gorm:"uniqueIndex:idx_supervision_command,priority:1;index;not null"`
	ResponsibleAssigneeID uint      `json:"responsibleAssigneeId" gorm:"index;not null"`
	DueAt                 time.Time `json:"dueAt" gorm:"index;not null"`
	CaseVersionBefore     uint      `json:"caseVersionBefore" gorm:"not null"`
	CaseVersionAfter      uint      `json:"caseVersionAfter" gorm:"not null"`
	OccurredAt            time.Time `json:"occurredAt" gorm:"index;not null"`
	Operation             string    `json:"-" gorm:"type:varchar(64);uniqueIndex:idx_supervision_command,priority:2;not null"`
	CommandKeyDigest      string    `json:"-" gorm:"type:char(64);uniqueIndex:idx_supervision_command,priority:3;not null"`
	RequestHash           string    `json:"-" gorm:"type:char(64);not null"`
	Synthetic             bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (SupervisorGuidance) TableName() string { return "supervisor_guidance" }
