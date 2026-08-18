package careclient

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// CareAssignment is an append-oriented, time-bounded responsibility fact.
type CareAssignment struct {
	global.GVA_MODEL
	CareClientID         uint       `json:"careClientId" gorm:"index;not null"`
	OrganizationID       uint       `json:"organizationId" gorm:"index;not null"`
	TeamID               uint       `json:"teamId" gorm:"index;not null"`
	AssigneeID           uint       `json:"assigneeId" gorm:"index;not null"`
	RoleType             string     `json:"roleType" gorm:"type:varchar(32);index;not null"`
	ValidFrom            time.Time  `json:"validFrom" gorm:"index;not null"`
	ValidUntil           *time.Time `json:"validUntil" gorm:"index"`
	ReplacesAssignmentID *uint      `json:"replacesAssignmentId" gorm:"index"`
	Reason               string     `json:"reason" gorm:"type:varchar(1000);not null"`
	EndedAt              *time.Time `json:"endedAt"`
	EndReason            string     `json:"endReason" gorm:"type:varchar(1000)"`
	CancelledAt          *time.Time `json:"cancelledAt"`
	Synthetic            bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId               uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy            uint       `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy            uint       `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy            uint       `json:"-" gorm:"column:deleted_by"`
}

func (CareAssignment) TableName() string { return "care_assignments" }

func (a CareAssignment) EffectiveStatus(now time.Time) string {
	if a.CancelledAt != nil {
		return AssignmentStatusCancelled
	}
	if now.Before(a.ValidFrom) {
		return AssignmentStatusScheduled
	}
	if a.ValidUntil != nil && !now.Before(*a.ValidUntil) {
		return AssignmentStatusEnded
	}
	return AssignmentStatusActive
}
