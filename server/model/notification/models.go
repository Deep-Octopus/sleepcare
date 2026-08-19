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
	ProviderCode          string     `json:"providerCode" gorm:"type:varchar(64);index"`
	DispatchPolicyCode    string     `json:"dispatchPolicyCode" gorm:"type:varchar(80);index"`
	DispatchPolicyVersion int        `json:"dispatchPolicyVersion"`
	TemplateCode          string     `json:"templateCode" gorm:"type:varchar(128)"`
	ProviderMessageIDHash *string    `json:"-" gorm:"type:char(64);uniqueIndex"`
	EstimatedCostMinor    int64      `json:"estimatedCostMinor"`
	CostCurrency          string     `json:"costCurrency" gorm:"type:char(3)"`
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

// NotificationDispatchReservation is the immutable quota and cost hold for
// one provider-facing attempt. A unique attempt can consume capacity once.
type NotificationDispatchReservation struct {
	global.GVA_MODEL
	NotificationRequestID uint      `json:"notificationRequestId" gorm:"index;not null"`
	NotificationAttemptID uint      `json:"notificationAttemptId" gorm:"uniqueIndex;not null"`
	ProviderCode          string    `json:"providerCode" gorm:"type:varchar(64);index;not null"`
	PolicyCode            string    `json:"policyCode" gorm:"type:varchar(80);index;not null"`
	PolicyVersion         int       `json:"policyVersion" gorm:"not null"`
	TemplateCode          string    `json:"templateCode" gorm:"type:varchar(128);not null"`
	RateBucketID          uint      `json:"rateBucketId" gorm:"index;not null"`
	CostBucketID          uint      `json:"costBucketId" gorm:"index;not null"`
	EstimatedCostMinor    int64     `json:"estimatedCostMinor" gorm:"not null"`
	CostCurrency          string    `json:"costCurrency" gorm:"type:char(3);not null"`
	Status                string    `json:"status" gorm:"type:varchar(24);index;not null"`
	ReservedAt            time.Time `json:"reservedAt" gorm:"index;not null"`
	Synthetic             bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (NotificationDispatchReservation) TableName() string {
	return "notification_dispatch_reservations"
}

// NotificationQuotaBucket serializes rate and cost consumption for a policy
// version. LimitValue is a frozen snapshot and must not drift in-place.
type NotificationQuotaBucket struct {
	global.GVA_MODEL
	ProviderCode  string    `json:"providerCode" gorm:"type:varchar(64);uniqueIndex:idx_notification_quota_bucket,priority:1;not null"`
	PolicyCode    string    `json:"policyCode" gorm:"type:varchar(80);uniqueIndex:idx_notification_quota_bucket,priority:2;not null"`
	PolicyVersion int       `json:"policyVersion" gorm:"uniqueIndex:idx_notification_quota_bucket,priority:3;not null"`
	BucketKind    string    `json:"bucketKind" gorm:"type:varchar(16);uniqueIndex:idx_notification_quota_bucket,priority:4;not null"`
	BucketKey     string    `json:"bucketKey" gorm:"type:varchar(40);uniqueIndex:idx_notification_quota_bucket,priority:5;not null"`
	WindowStart   time.Time `json:"windowStart" gorm:"index;not null"`
	WindowEnd     time.Time `json:"windowEnd" gorm:"index;not null"`
	UsedValue     int64     `json:"usedValue" gorm:"not null;default:0"`
	LimitValue    int64     `json:"limitValue" gorm:"not null"`
	Version       uint      `json:"version" gorm:"not null;default:1"`
}

func (NotificationQuotaBucket) TableName() string { return "notification_quota_buckets" }

// NotificationProviderCallback stores only verified, digested callback
// evidence. Raw provider identifiers and payloads are intentionally omitted.
type NotificationProviderCallback struct {
	global.GVA_MODEL
	ProviderCode          string    `json:"providerCode" gorm:"type:varchar(64);uniqueIndex:idx_notification_callback_event,priority:1;uniqueIndex:idx_notification_callback_nonce,priority:1;not null"`
	EventKeyDigest        string    `json:"-" gorm:"type:char(64);uniqueIndex:idx_notification_callback_event,priority:2;not null"`
	NonceDigest           string    `json:"-" gorm:"type:char(64);uniqueIndex:idx_notification_callback_nonce,priority:2;not null"`
	PayloadDigest         string    `json:"-" gorm:"type:char(64);not null"`
	NotificationAttemptID uint      `json:"notificationAttemptId" gorm:"index;not null"`
	DeliveryEventID       uint      `json:"deliveryEventId" gorm:"index;not null"`
	Status                string    `json:"status" gorm:"type:varchar(32);index;not null"`
	OccurredAt            time.Time `json:"occurredAt" gorm:"index;not null"`
	ReceivedAt            time.Time `json:"receivedAt" gorm:"index;not null"`
	SignatureTimestamp    time.Time `json:"signatureTimestamp" gorm:"not null"`
	Synthetic             bool      `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId                uint      `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy             uint      `json:"createdBy" gorm:"column:created_by;index"`
}

func (NotificationProviderCallback) TableName() string {
	return "notification_provider_callbacks"
}
