package notification

import (
	"context"
	"time"

	careconfig "github.com/flipped-aurora/gin-vue-admin/server/config"
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

func (SystemClock) Now() time.Time { return global.GVA_CONFIG.Care.Now() }

type FixedClock struct{ Time time.Time }

func (c FixedClock) Now() time.Time { return c.Time }

type Service interface {
	ListDeliveries(context.Context, notificationreq.DeliverySearch) ([]notificationres.NotificationAttempt, int64, error)
	GetProviderReadiness(context.Context) (notificationres.ProviderReadiness, error)
	Resend(context.Context, uint, string, notificationreq.Resend) (caseworkres.ActionResult, error)
	ApplyProviderCallback(context.Context, string, []byte, notificationreq.ProviderCallbackSignature) (caseworkres.ActionResult, error)
}

type NotificationService struct {
	DB                       *gorm.DB
	Clock                    Clock
	Adapter                  NotificationPort
	ProviderConfig           *careconfig.NotificationProvider
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
	clock := s.Clock
	if clock == nil {
		clock = SystemClock{}
	}
	return DemoNotificationAdapter{Outcome: "UNKNOWN", Clock: clock}
}

func (s *NotificationService) fixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}

func (s *NotificationService) providerConfig() careconfig.NotificationProvider {
	if s.ProviderConfig != nil {
		return *s.ProviderConfig
	}
	return global.GVA_CONFIG.Care.NotificationProvider
}

type ServiceGroup struct {
	NotificationService
}
