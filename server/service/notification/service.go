package notification

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	notificationres "github.com/flipped-aurora/gin-vue-admin/server/model/notification/response"
	"gorm.io/gorm"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type FixedClock struct{ Time time.Time }

func (c FixedClock) Now() time.Time { return c.Time }

type Service interface {
	ListDeliveries(context.Context, notificationreq.DeliverySearch) ([]notificationres.NotificationAttempt, int64, error)
	Resend(context.Context, uint, string, notificationreq.Resend) (caseworkres.ActionResult, error)
}

type NotificationService struct {
	DB                       *gorm.DB
	Clock                    Clock
	Adapter                  NotificationPort
	SyntheticFixturesEnabled *bool
}

var _ Service = (*NotificationService)(nil)

func (s *NotificationService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *NotificationService) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now()
	}
	return SystemClock{}.Now()
}

func (s *NotificationService) adapter() NotificationPort {
	if s.Adapter != nil {
		return s.Adapter
	}
	return DemoNotificationAdapter{Outcome: "UNKNOWN", Clock: s.Clock}
}

func (s *NotificationService) fixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}

type ServiceGroup struct {
	NotificationService
}
