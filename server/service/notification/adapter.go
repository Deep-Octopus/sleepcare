package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
)

type SendCommand struct {
	NotificationRequestID uint
	NotificationAttemptID uint
	TaskID                uint
	CareClientID          uint
	DeptID                uint
	AttemptNo             int
	RequestedAt           time.Time
}

type DeliveryReceipt struct {
	EventKey         string
	Status           string
	OccurredAt       time.Time
	FailureCode      string
	AdapterReference string
}

type NotificationPort interface {
	Descriptor() AdapterDescriptor
	Submit(context.Context, SendCommand) ([]DeliveryReceipt, error)
}

// DemoNotificationAdapter is deterministic and performs no network I/O. It
// emits the same normalized lifecycle a provider adapter would produce.
type DemoNotificationAdapter struct {
	Outcome string
	Clock   Clock
}

func (DemoNotificationAdapter) Descriptor() AdapterDescriptor {
	return AdapterDescriptor{Channel: notificationmodel.ChannelDemo}
}

func (a DemoNotificationAdapter) Submit(_ context.Context, command SendCommand) ([]DeliveryReceipt, error) {
	outcome := strings.ToUpper(strings.TrimSpace(a.Outcome))
	if outcome == "" {
		outcome = notificationmodel.AttemptStatusUnknown
	}
	if !notificationmodel.IsFinalAttemptStatus(outcome) {
		return nil, notificationmodel.NewDomainError(notificationmodel.CodeInvalidArgument, "测试适配器结果必须为 DELIVERED、FAILED 或 UNKNOWN")
	}
	base := command.RequestedAt
	if base.IsZero() {
		if a.Clock != nil {
			base = a.Clock.Now()
		} else {
			base = time.Now()
		}
	}
	prefix := fmt.Sprintf("demo:%d", command.NotificationAttemptID)
	receipts := []DeliveryReceipt{
		{EventKey: prefix + ":submitted", Status: notificationmodel.AttemptStatusSubmittedToProvider, OccurredAt: base.Add(time.Second), AdapterReference: prefix + ":provider"},
		{EventKey: prefix + ":accepted", Status: notificationmodel.AttemptStatusAccepted, OccurredAt: base.Add(2 * time.Second), AdapterReference: prefix + ":accepted"},
		{EventKey: prefix + ":final", Status: outcome, OccurredAt: base.Add(3 * time.Second), AdapterReference: prefix + ":final"},
	}
	switch outcome {
	case notificationmodel.AttemptStatusFailed:
		receipts[2].FailureCode = notificationmodel.DemoFailureCode
	case notificationmodel.AttemptStatusUnknown:
		receipts[2].FailureCode = notificationmodel.DemoUnknownCode
	}
	return receipts, nil
}
