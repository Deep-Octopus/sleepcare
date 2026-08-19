package careclient

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// DataLifecycleRequest is an append-only record of a request that cannot be
// executed until the corresponding governance policy is approved.
type DataLifecycleRequest struct {
	global.GVA_MODEL
	OrganizationID             uint      `json:"organizationId" gorm:"index;not null"`
	CareClientID               uint      `json:"careClientId" gorm:"index;not null"`
	RequestType                string    `json:"requestType" gorm:"type:varchar(32);index;not null"`
	Status                     string    `json:"status" gorm:"type:varchar(32);index;not null"`
	RequestedAt                time.Time `json:"requestedAt" gorm:"index;not null"`
	Source                     string    `json:"source" gorm:"type:varchar(32);not null"`
	Reason                     string    `json:"reason" gorm:"type:varchar(1000);not null"`
	IdentityVerificationStatus string    `json:"identityVerificationStatus" gorm:"type:varchar(32);not null"`
	GovernanceMode             string    `json:"governanceMode" gorm:"type:varchar(32);not null"`
	PolicySnapshotDigest       string    `json:"-" gorm:"type:char(64);not null"`
	ExecutionAllowed           bool      `json:"executionAllowed" gorm:"not null;default:false"`
	Synthetic                  bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                     uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy                  uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (DataLifecycleRequest) TableName() string { return "care_data_lifecycle_requests" }
