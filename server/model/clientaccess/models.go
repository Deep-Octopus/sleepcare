package clientaccess

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

// CareClientAccount is the client-facing security principal. It is kept
// separate from the employee user model and has no staff-token semantics.
type CareClientAccount struct {
	global.GVA_MODEL
	CareClientID uint   `json:"careClientId" gorm:"uniqueIndex;not null"`
	Status       string `json:"status" gorm:"type:varchar(24);index;not null"`
	Version      uint   `json:"version" gorm:"not null;default:1"`
	Synthetic    bool   `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId       uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy    uint   `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy    uint   `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy    uint   `json:"-" gorm:"column:deleted_by"`
}

func (CareClientAccount) TableName() string { return "care_client_accounts" }

// ClientAccessGrant stores only a digest of the one-time bearer value. The
// task scope is frozen at issue time and copied into the resulting session.
type ClientAccessGrant struct {
	global.GVA_MODEL
	AccountID          uint           `json:"accountId" gorm:"index;not null"`
	CareClientID       uint           `json:"careClientId" gorm:"index;not null"`
	TokenDigest        string         `json:"-" gorm:"type:char(64);uniqueIndex;not null"`
	AllowedTaskIDsJSON datatypes.JSON `json:"allowedTaskIds" gorm:"type:json;not null" swaggertype:"array,integer"`
	Status             string         `json:"status" gorm:"type:varchar(24);index;not null"`
	ExpiresAt          time.Time      `json:"expiresAt" gorm:"index;not null"`
	RedeemedAt         *time.Time     `json:"redeemedAt"`
	RevokedAt          *time.Time     `json:"revokedAt"`
	Synthetic          bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId             uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy          uint           `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy          uint           `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy          uint           `json:"-" gorm:"column:deleted_by"`
}

func (ClientAccessGrant) TableName() string { return "client_access_grants" }

type ClientSession struct {
	global.GVA_MODEL
	SessionID          string         `json:"sessionId" gorm:"type:char(36);uniqueIndex;not null"`
	AccountID          uint           `json:"accountId" gorm:"index;not null"`
	GrantID            uint           `json:"grantId" gorm:"uniqueIndex;not null"`
	CareClientID       uint           `json:"careClientId" gorm:"index;not null"`
	TokenDigest        string         `json:"-" gorm:"type:char(64);uniqueIndex;not null"`
	AllowedTaskIDsJSON datatypes.JSON `json:"allowedTaskIds" gorm:"type:json;not null" swaggertype:"array,integer"`
	Status             string         `json:"status" gorm:"type:varchar(24);index;not null"`
	ExpiresAt          time.Time      `json:"expiresAt" gorm:"index;not null"`
	RevokedAt          *time.Time     `json:"revokedAt"`
	Synthetic          bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId             uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy          uint           `json:"createdBy" gorm:"column:created_by;index"`
	UpdatedBy          uint           `json:"updatedBy" gorm:"column:updated_by"`
	DeletedBy          uint           `json:"-" gorm:"column:deleted_by"`
}

func (ClientSession) TableName() string { return "client_sessions" }

// ClientTaskCommandReceipt is a technical replay record for client mutations.
// It is not a domain aggregate and contains no raw bearer or idempotency key.
type ClientTaskCommandReceipt struct {
	global.GVA_MODEL
	CareClientID uint   `json:"careClientId" gorm:"uniqueIndex:idx_client_task_receipt,priority:1;not null"`
	Operation    string `json:"operation" gorm:"type:varchar(96);uniqueIndex:idx_client_task_receipt,priority:2;not null"`
	KeyDigest    string `json:"-" gorm:"type:char(64);uniqueIndex:idx_client_task_receipt,priority:3;not null"`
	RequestHash  string `json:"requestHash" gorm:"type:char(64);not null"`
	ResultJSON   string `json:"-" gorm:"type:text;not null"`
	DeptId       uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy    uint   `json:"createdBy" gorm:"column:created_by;index"`
}

func (ClientTaskCommandReceipt) TableName() string { return "client_task_command_receipts" }
