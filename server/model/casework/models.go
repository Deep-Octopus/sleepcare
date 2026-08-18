package casework

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

type AttentionCase struct {
	global.GVA_MODEL
	CareClientID    uint       `json:"careClientId" gorm:"index;not null"`
	TaskID          uint       `json:"taskId" gorm:"index;not null"`
	SubmissionID    uint       `json:"submissionId" gorm:"index;not null"`
	SourceType      string     `json:"sourceType" gorm:"type:varchar(32);uniqueIndex:idx_attention_case_source,priority:1;not null"`
	SourceRuleHitID uint       `json:"sourceRuleHitId" gorm:"uniqueIndex:idx_attention_case_source,priority:2;index;not null"`
	DedupKey        string     `json:"dedupKey" gorm:"type:varchar(200);uniqueIndex:idx_attention_case_source,priority:3;not null"`
	Status          string     `json:"status" gorm:"type:varchar(32);index;not null"`
	AttentionLevel  string     `json:"attentionLevel" gorm:"type:varchar(64);index;not null"`
	ReasonSummary   string     `json:"reasonSummary" gorm:"type:varchar(500);not null"`
	AssigneeID      *uint      `json:"assigneeId" gorm:"index"`
	AssigneeRole    string     `json:"assigneeRole" gorm:"type:varchar(32);index"`
	OpenedAt        time.Time  `json:"openedAt" gorm:"index;not null"`
	DueAt           *time.Time `json:"dueAt" gorm:"index"`
	ResolvedAt      *time.Time `json:"resolvedAt" gorm:"index"`
	ClosedAt        *time.Time `json:"closedAt" gorm:"index"`
	HandlingResult  string     `json:"handlingResult" gorm:"type:text"`
	CloseReason     string     `json:"closeReason" gorm:"type:text"`
	Version         uint       `json:"version" gorm:"not null;default:1"`
	Synthetic       bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId          uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy       uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy       uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy       uint       `json:"-" gorm:"column:deleted_by"`
}

func (AttentionCase) TableName() string { return "attention_cases" }

type CaseAction struct {
	global.GVA_MODEL
	AttentionCaseID  uint       `json:"attentionCaseId" gorm:"index;not null"`
	ActionType       string     `json:"actionType" gorm:"type:varchar(32);index;not null"`
	ActorID          uint       `json:"actorId" gorm:"index;not null"`
	ActorRole        string     `json:"actorRole" gorm:"type:varchar(32);index;not null"`
	OrganizationID   uint       `json:"organizationId" gorm:"index;not null"`
	TeamID           uint       `json:"teamId" gorm:"index;not null"`
	Source           string     `json:"source" gorm:"type:varchar(32);not null"`
	Result           string     `json:"result" gorm:"type:text;not null"`
	Reason           string     `json:"reason" gorm:"type:text"`
	FromStatus       string     `json:"fromStatus" gorm:"type:varchar(32)"`
	ToStatus         string     `json:"toStatus" gorm:"type:varchar(32)"`
	TargetAssigneeID *uint      `json:"targetAssigneeId" gorm:"index"`
	TargetRole       string     `json:"targetRole" gorm:"type:varchar(32)"`
	DueAt            *time.Time `json:"dueAt"`
	OccurredAt       time.Time  `json:"occurredAt" gorm:"index;not null"`
	CommandKeyDigest string     `json:"-" gorm:"type:char(64);index;not null"`
	Synthetic        bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId           uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy        uint       `json:"createdBy" gorm:"column:created_by;index"`
}

func (CaseAction) TableName() string { return "case_actions" }

type TodoItem struct {
	global.GVA_MODEL
	Category       string     `json:"category" gorm:"type:varchar(48);index;not null"`
	SourceType     string     `json:"sourceType" gorm:"type:varchar(32);uniqueIndex:idx_todo_active,priority:1;not null"`
	SourceID       uint       `json:"sourceId" gorm:"uniqueIndex:idx_todo_active,priority:2;index;not null"`
	ActiveSlot     *string    `json:"-" gorm:"type:varchar(16);uniqueIndex:idx_todo_active,priority:3"`
	CareClientID   uint       `json:"careClientId" gorm:"index;not null"`
	AssigneeID     uint       `json:"assigneeId" gorm:"index;not null"`
	AssigneeRole   string     `json:"assigneeRole" gorm:"type:varchar(32);index;not null"`
	Status         string     `json:"status" gorm:"type:varchar(24);index;not null"`
	OpenedAt       time.Time  `json:"openedAt" gorm:"index;not null"`
	DueAt          *time.Time `json:"dueAt" gorm:"index"`
	CompletedAt    *time.Time `json:"completedAt" gorm:"index"`
	CompletionNote string     `json:"completionNote" gorm:"type:varchar(500)"`
	Version        uint       `json:"version" gorm:"not null;default:1"`
	Synthetic      bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId         uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy      uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy      uint       `json:"-" gorm:"column:deleted_by"`
}

func (TodoItem) TableName() string { return "todo_items" }

type CommandReceipt struct {
	global.GVA_MODEL
	ActorID     uint   `json:"actorId" gorm:"uniqueIndex:idx_case_receipt,priority:1;not null"`
	Operation   string `json:"operation" gorm:"type:varchar(64);uniqueIndex:idx_case_receipt,priority:2;not null"`
	KeyDigest   string `json:"-" gorm:"type:char(64);uniqueIndex:idx_case_receipt,priority:3;not null"`
	RequestHash string `json:"-" gorm:"type:char(64);not null"`
	ResultJSON  string `json:"-" gorm:"type:text;not null"`
	DeptId      uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy   uint   `json:"createdBy" gorm:"column:created_by;index"`
}

func (CommandReceipt) TableName() string { return "case_command_receipts" }
