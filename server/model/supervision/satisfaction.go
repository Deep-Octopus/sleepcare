package supervision

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// SatisfactionPolicy is an immutable policy version used to project a
// service-close fact into one evaluation request.
type SatisfactionPolicy struct {
	global.GVA_MODEL
	Code              string    `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_satisfaction_policy_version,priority:1;not null"`
	Version           uint      `json:"version" gorm:"uniqueIndex:idx_satisfaction_policy_version,priority:2;not null"`
	TriggerType       string    `json:"triggerType" gorm:"type:varchar(32);index;not null"`
	AnonymityMode     string    `json:"anonymityMode" gorm:"type:varchar(48);not null"`
	ValidForHours     uint      `json:"validForHours" gorm:"not null"`
	LowScoreThreshold uint8     `json:"lowScoreThreshold" gorm:"not null"`
	EffectiveFrom     time.Time `json:"effectiveFrom" gorm:"index;not null"`
	Status            string    `json:"status" gorm:"type:varchar(24);index;not null"`
	Synthetic         bool      `json:"synthetic" gorm:"index;not null;default:false"`
}

func (SatisfactionPolicy) TableName() string { return "satisfaction_policies" }

// SatisfactionRequest links one immutable service-close event to an
// evaluation window. Client and staff DTOs intentionally omit the internal
// linkage fields needed for system-level quality review.
type SatisfactionRequest struct {
	global.GVA_MODEL
	SourceType          string     `json:"sourceType" gorm:"type:varchar(32);uniqueIndex:idx_satisfaction_source_event,priority:1;index;not null"`
	SourceID            uint       `json:"sourceId" gorm:"index;not null"`
	SourceEventID       uint       `json:"sourceEventId" gorm:"uniqueIndex:idx_satisfaction_source_event,priority:2;index;not null"`
	CareClientID        uint       `json:"careClientId" gorm:"index;not null"`
	OrganizationID      uint       `json:"organizationId" gorm:"index;not null"`
	ServiceAssigneeID   *uint      `json:"serviceAssigneeId" gorm:"index"`
	ServiceAssigneeRole string     `json:"serviceAssigneeRole" gorm:"type:varchar(32);index"`
	PolicyID            uint       `json:"policyId" gorm:"index;not null"`
	PolicyCode          string     `json:"policyCode" gorm:"type:varchar(64);not null"`
	PolicyVersion       uint       `json:"policyVersion" gorm:"not null"`
	AnonymityMode       string     `json:"anonymityMode" gorm:"type:varchar(48);not null"`
	LowScoreThreshold   uint8      `json:"lowScoreThreshold" gorm:"not null"`
	Status              string     `json:"status" gorm:"type:varchar(24);index;not null"`
	InvitedAt           time.Time  `json:"invitedAt" gorm:"index;not null"`
	ExpiresAt           time.Time  `json:"expiresAt" gorm:"index;not null"`
	SubmittedAt         *time.Time `json:"submittedAt" gorm:"index"`
	Version             uint       `json:"version" gorm:"not null;default:1"`
	Synthetic           bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId              uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy           uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy           uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy           uint       `json:"-" gorm:"column:deleted_by"`
}

func (SatisfactionRequest) TableName() string { return "satisfaction_requests" }

// SatisfactionResponse is append-only and unique per request.
type SatisfactionResponse struct {
	global.GVA_MODEL
	RequestID        uint      `json:"requestId" gorm:"uniqueIndex;not null"`
	Rating           uint8     `json:"rating" gorm:"index;not null"`
	Comment          string    `json:"comment" gorm:"type:text"`
	SubmittedAt      time.Time `json:"submittedAt" gorm:"index;not null"`
	CommandKeyDigest string    `json:"-" gorm:"type:char(64);not null"`
	RequestHash      string    `json:"-" gorm:"type:char(64);not null"`
	Synthetic        bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId           uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy        uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (SatisfactionResponse) TableName() string { return "satisfaction_responses" }

// SatisfactionFollowUp is the operational aggregate created for a low score.
// It supports quality review but is not a personnel-performance conclusion.
type SatisfactionFollowUp struct {
	global.GVA_MODEL
	RequestID         uint       `json:"requestId" gorm:"uniqueIndex;not null"`
	ResponseID        uint       `json:"responseId" gorm:"uniqueIndex;not null"`
	OrganizationID    uint       `json:"organizationId" gorm:"index;not null"`
	AssigneeID        *uint      `json:"assigneeId" gorm:"index"`
	Status            string     `json:"status" gorm:"type:varchar(24);index;not null"`
	OpenedAt          time.Time  `json:"openedAt" gorm:"index;not null"`
	AcknowledgedAt    *time.Time `json:"acknowledgedAt" gorm:"index"`
	ResolvedAt        *time.Time `json:"resolvedAt" gorm:"index"`
	Resolution        string     `json:"resolution" gorm:"type:text"`
	ImprovementAction string     `json:"improvementAction" gorm:"type:text"`
	Version           uint       `json:"version" gorm:"not null;default:1"`
	Synthetic         bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId            uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy         uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy         uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy         uint       `json:"-" gorm:"column:deleted_by"`
}

func (SatisfactionFollowUp) TableName() string { return "satisfaction_follow_ups" }

// SatisfactionFollowUpAction is append-only and also stores the command
// fingerprint needed for idempotent replay.
type SatisfactionFollowUpAction struct {
	global.GVA_MODEL
	FollowUpID             uint      `json:"followUpId" gorm:"index;not null"`
	ActionType             string    `json:"actionType" gorm:"type:varchar(24);index;not null"`
	ActorID                uint      `json:"actorId" gorm:"uniqueIndex:idx_satisfaction_followup_command,priority:1;index;not null"`
	ActorRole              string    `json:"actorRole" gorm:"type:varchar(32);not null"`
	Content                string    `json:"content" gorm:"type:text;not null"`
	ImprovementAction      string    `json:"improvementAction" gorm:"type:text"`
	UsageBoundaryConfirmed bool      `json:"usageBoundaryConfirmed" gorm:"not null;default:false"`
	FromStatus             string    `json:"fromStatus" gorm:"type:varchar(24);not null"`
	ToStatus               string    `json:"toStatus" gorm:"type:varchar(24);not null"`
	VersionBefore          uint      `json:"versionBefore" gorm:"not null"`
	VersionAfter           uint      `json:"versionAfter" gorm:"not null"`
	OccurredAt             time.Time `json:"occurredAt" gorm:"index;not null"`
	Operation              string    `json:"-" gorm:"type:varchar(64);uniqueIndex:idx_satisfaction_followup_command,priority:2;not null"`
	CommandKeyDigest       string    `json:"-" gorm:"type:char(64);uniqueIndex:idx_satisfaction_followup_command,priority:3;not null"`
	RequestHash            string    `json:"-" gorm:"type:char(64);not null"`
	Synthetic              bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                 uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy              uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (SatisfactionFollowUpAction) TableName() string { return "satisfaction_follow_up_actions" }
