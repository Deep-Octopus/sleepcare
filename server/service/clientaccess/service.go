package clientaccess

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	clientreq "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/request"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	"gorm.io/gorm"
)

type ClientAccessService struct {
	DB                       *gorm.DB
	Now                      func() time.Time
	SessionTTL               time.Duration
	SyntheticFixturesEnabled *bool
}

func (s *ClientAccessService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *ClientAccessService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return global.GVA_CONFIG.Care.Now()
}

func (s *ClientAccessService) sessionTTL() time.Duration {
	if s.SessionTTL > 0 {
		return s.SessionTTL
	}
	minutes := global.GVA_CONFIG.Care.ClientAccess.SessionTTLMinutes
	if minutes <= 0 {
		minutes = 480
	}
	return time.Duration(minutes) * time.Minute
}

func (s *ClientAccessService) syntheticFixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}

type Service interface {
	Redeem(context.Context, string) (clientres.RedeemResult, string, error)
	Authenticate(context.Context, string) (SessionIdentity, error)
	ListTasks(context.Context, clientreq.TaskSearch) ([]clientres.TaskSummary, int64, error)
	GetTask(context.Context, uint) (clientres.TaskDetail, error)
	GetQuestionnaire(context.Context, uint) (clientres.Questionnaire, error)
	RecordInteraction(context.Context, uint, string, clientreq.RecordInteraction) (clientres.InteractionResult, error)
	SaveDraft(context.Context, uint, string, clientreq.SaveDraft) (clientres.DraftResult, error)
	SubmitTask(context.Context, uint, string, clientreq.SubmitTask) (clientres.SubmitResult, error)
}

var _ Service = (*ClientAccessService)(nil)

type ServiceGroup struct {
	ClientAccessService
}
