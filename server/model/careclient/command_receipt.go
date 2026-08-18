package careclient

import "github.com/flipped-aurora/gin-vue-admin/server/global"

// CareClientCommandReceipt provides durable idempotency for aggregate writes.
type CareClientCommandReceipt struct {
	global.GVA_MODEL
	Operation      string `json:"operation" gorm:"type:varchar(64);uniqueIndex:idx_care_receipt_actor_op_key,priority:2;not null"`
	ActorID        uint   `json:"actorId" gorm:"uniqueIndex:idx_care_receipt_actor_op_key,priority:1;not null"`
	IdempotencyKey string `json:"idempotencyKey" gorm:"type:varchar(128);uniqueIndex:idx_care_receipt_actor_op_key,priority:3;not null"`
	RequestHash    string `json:"requestHash" gorm:"type:char(64);not null"`
	ResultJSON     string `json:"-" gorm:"type:text;not null"`
	DeptId         uint   `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint   `json:"createdBy" gorm:"column:created_by;index"`
}

func (CareClientCommandReceipt) TableName() string { return "care_client_command_receipts" }
