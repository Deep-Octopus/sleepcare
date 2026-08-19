package casework

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// Consultation is a client-originated service request. It records workflow
// state only and must not be treated as a diagnosis or emergency channel.
type Consultation struct {
	global.GVA_MODEL
	CareClientID     uint       `json:"careClientId" gorm:"index;not null"`
	Source           string     `json:"source" gorm:"type:varchar(24);index;not null"`
	Subject          string     `json:"subject" gorm:"type:varchar(120);not null"`
	InitialQuestion  string     `json:"initialQuestion" gorm:"type:text;not null"`
	Urgency          string     `json:"urgency" gorm:"type:varchar(24);index;not null"`
	Status           string     `json:"status" gorm:"type:varchar(32);index;not null"`
	AssigneeID       *uint      `json:"assigneeId" gorm:"index"`
	AssigneeRole     string     `json:"assigneeRole" gorm:"type:varchar(32);index"`
	OpenedAt         time.Time  `json:"openedAt" gorm:"index;not null"`
	FirstRespondedAt *time.Time `json:"firstRespondedAt" gorm:"index"`
	ResolvedAt       *time.Time `json:"resolvedAt" gorm:"index"`
	ClosedAt         *time.Time `json:"closedAt" gorm:"index"`
	Resolution       string     `json:"resolution" gorm:"type:text"`
	FollowUpPlan     string     `json:"followUpPlan" gorm:"type:text"`
	CloseReason      string     `json:"closeReason" gorm:"type:text"`
	Version          uint       `json:"version" gorm:"not null;default:1"`
	Synthetic        bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId           uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy        uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy        uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy        uint       `json:"-" gorm:"column:deleted_by"`
}

func (Consultation) TableName() string { return "consultations" }

// ConsultationInteraction is append-only. Client-facing queries only expose
// rows explicitly marked ClientVisible.
type ConsultationInteraction struct {
	global.GVA_MODEL
	ConsultationID   uint      `json:"consultationId" gorm:"index;not null"`
	ActionType       string    `json:"actionType" gorm:"type:varchar(32);index;not null"`
	ActorType        string    `json:"actorType" gorm:"type:varchar(24);index;not null"`
	ActorID          uint      `json:"actorId" gorm:"index;not null"`
	ActorRole        string    `json:"actorRole" gorm:"type:varchar(32);index"`
	Content          string    `json:"content" gorm:"type:text"`
	Reason           string    `json:"reason" gorm:"type:text"`
	FromStatus       string    `json:"fromStatus" gorm:"type:varchar(32)"`
	ToStatus         string    `json:"toStatus" gorm:"type:varchar(32)"`
	TargetAssigneeID *uint     `json:"targetAssigneeId" gorm:"index"`
	TargetRole       string    `json:"targetRole" gorm:"type:varchar(32)"`
	ClientVisible    bool      `json:"clientVisible" gorm:"index;not null;default:false"`
	OccurredAt       time.Time `json:"occurredAt" gorm:"index;not null"`
	CommandKeyDigest string    `json:"-" gorm:"type:char(64);index;not null"`
	Synthetic        bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId           uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy        uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (ConsultationInteraction) TableName() string { return "consultation_interactions" }
