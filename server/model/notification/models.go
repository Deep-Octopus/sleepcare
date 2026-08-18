package notification

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// NotificationRequest is the stable delivery intent for one task. Attempts
// are append-only children so retry history can never overwrite prior facts.
type NotificationRequest struct {
	global.GVA_MODEL
	TaskID       uint      `json:"taskId" gorm:"uniqueIndex;not null"`
	CareClientID uint      `json:"careClientId" gorm:"index;not null"`
	Channel      string    `json:"channel" gorm:"type:varchar(24);not null"`
	RequestedAt  time.Time `json:"requestedAt" gorm:"index;not null"`
	Synthetic    bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId       uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy    uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (NotificationRequest) TableName() string { return "notification_requests" }

type NotificationAttempt struct {
	global.GVA_MODEL
	NotificationRequestID uint       `json:"notificationRequestId" gorm:"uniqueIndex:idx_notification_attempt_no,priority:1;index;not null"`
	AttemptNo             int        `json:"attemptNo" gorm:"uniqueIndex:idx_notification_attempt_no,priority:2;not null"`
	TaskID                uint       `json:"taskId" gorm:"index;not null"`
	CareClientID          uint       `json:"careClientId" gorm:"index;not null"`
	RetryOfAttemptID      *uint      `json:"retryOfAttemptId" gorm:"index"`
	Channel               string     `json:"channel" gorm:"type:varchar(24);not null"`
	Status                string     `json:"status" gorm:"type:varchar(32);index;not null"`
	RequestedAt           time.Time  `json:"requestedAt" gorm:"index;not null"`
	SubmittedAt           *time.Time `json:"submittedAt"`
	AcceptedAt            *time.Time `json:"acceptedAt"`
	DeliveredAt           *time.Time `json:"deliveredAt"`
	FinalizedAt           *time.Time `json:"finalizedAt" gorm:"index"`
	FailureCode           string     `json:"failureCode" gorm:"type:varchar(64)"`
	ResendReason          string     `json:"resendReason" gorm:"type:varchar(1000)"`
	Version               uint       `json:"version" gorm:"not null;default:1"`
	ActorID               uint       `json:"actorId" gorm:"uniqueIndex:idx_notification_command,priority:1;index;not null"`
	Operation             string     `json:"-" gorm:"type:varchar(80);uniqueIndex:idx_notification_command,priority:2;not null"`
	CommandKeyDigest      string     `json:"-" gorm:"type:char(64);uniqueIndex:idx_notification_command,priority:3;not null"`
	RequestHash           string     `json:"-" gorm:"type:char(64);not null"`
	Synthetic             bool       `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint       `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint       `json:"createdBy" gorm:"column:created_by;index"`
}

func (NotificationAttempt) TableName() string { return "notification_attempts" }

type DeliveryEvent struct {
	global.GVA_MODEL
	EventID               string    `json:"eventId" gorm:"type:char(36);uniqueIndex;not null"`
	EventKey              string    `json:"-" gorm:"type:varchar(120);uniqueIndex:idx_delivery_event_key,priority:2;not null"`
	NotificationRequestID uint      `json:"notificationRequestId" gorm:"index;not null"`
	NotificationAttemptID uint      `json:"notificationAttemptId" gorm:"uniqueIndex:idx_delivery_event_key,priority:1;index;not null"`
	EventType             string    `json:"eventType" gorm:"type:varchar(64);index;not null"`
	FromStatus            string    `json:"fromStatus" gorm:"type:varchar(32)"`
	ToStatus              string    `json:"toStatus" gorm:"type:varchar(32);index;not null"`
	OccurredAt            time.Time `json:"occurredAt" gorm:"index;not null"`
	FailureCode           string    `json:"failureCode" gorm:"type:varchar(64)"`
	AdapterReferenceHash  string    `json:"adapterReferenceHash" gorm:"type:char(64)"`
	Synthetic             bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (DeliveryEvent) TableName() string { return "notification_delivery_events" }
