package careclient

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// ConsentRecord records a grant or withdrawal without overwriting prior facts.
type ConsentRecord struct {
	global.GVA_MODEL
	CareClientID uint      `json:"careClientId" gorm:"index;not null"`
	ConsentType  string    `json:"consentType" gorm:"type:varchar(64);index;not null"`
	Action       string    `json:"action" gorm:"type:varchar(16);not null"`
	TextVersion  string    `json:"textVersion" gorm:"type:varchar(80);not null"`
	OccurredAt   time.Time `json:"occurredAt" gorm:"index;not null"`
	Source       string    `json:"source" gorm:"type:varchar(32);not null"`
	Reason       string    `json:"reason" gorm:"type:varchar(1000)"`
	RecordedBy   uint      `json:"recordedBy" gorm:"index;not null"`
	Synthetic    bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId       uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy    uint      `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy    uint      `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy    uint      `json:"-" gorm:"column:deleted_by"`
}

func (ConsentRecord) TableName() string { return "care_consent_records" }
