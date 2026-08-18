package outbox

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Event is the shared transactional outbox record used by care domain modules.
// It carries ownership columns because event payloads belong to the same data
// scope as their aggregate.
type Event struct {
	global.GVA_MODEL
	EventID        string         `json:"eventId" gorm:"type:char(36);uniqueIndex;not null"`
	EventType      string         `json:"eventType" gorm:"type:varchar(64);index;not null"`
	PayloadVersion string         `json:"payloadVersion" gorm:"type:varchar(16);not null"`
	AggregateType  string         `json:"aggregateType" gorm:"type:varchar(64);index;not null"`
	AggregateID    string         `json:"aggregateId" gorm:"type:varchar(64);index;not null"`
	PayloadJSON    datatypes.JSON `json:"payload" gorm:"type:json;not null" swaggertype:"object"`
	OccurredAt     time.Time      `json:"occurredAt" gorm:"index;not null"`
	CorrelationID  string         `json:"correlationId" gorm:"type:varchar(64);index"`
	CausationID    string         `json:"causationId" gorm:"type:varchar(64);index"`
	Synthetic      bool           `json:"synthetic" gorm:"index;not null;default:false"`
	DeptId         uint           `json:"deptId" gorm:"column:dept_id;index;not null"`
	CreatedBy      uint           `json:"createdBy" gorm:"column:created_by;index"`
}

func (Event) TableName() string { return "outbox_events" }

type AppendInput struct {
	EventType     string
	AggregateType string
	AggregateID   uint
	Payload       any
	OccurredAt    time.Time
	CorrelationID string
	CausationID   string
	Synthetic     bool
	DeptID        uint
	CreatedBy     uint
}

// Append writes an event on the caller's transaction. The caller owns the
// transaction boundary so state and event can never commit independently.
func Append(tx *gorm.DB, input AppendInput) error {
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return err
	}
	event := Event{
		EventID: uuid.NewString(), EventType: input.EventType, PayloadVersion: "v1",
		AggregateType: input.AggregateType, AggregateID: strconv.FormatUint(uint64(input.AggregateID), 10),
		PayloadJSON: datatypes.JSON(payload), OccurredAt: input.OccurredAt,
		CorrelationID: input.CorrelationID, CausationID: input.CausationID, Synthetic: input.Synthetic,
		DeptId: input.DeptID, CreatedBy: input.CreatedBy,
	}
	return tx.Create(&event).Error
}
