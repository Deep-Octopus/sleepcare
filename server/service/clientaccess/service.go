package clientaccess

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	clientreq "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/request"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
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
	CreateConsultation(context.Context, string, caseworkreq.CreateConsultation) (caseworkres.ConsultationActionResult, error)
	ListConsultations(context.Context, caseworkreq.ClientConsultationSearch) ([]caseworkres.ClientConsultationSummary, int64, error)
	GetConsultation(context.Context, uint) (caseworkres.ClientConsultationDetail, error)
	AddConsultationMessage(context.Context, uint, string, caseworkreq.AddClientConsultationMessage) (caseworkres.ConsultationActionResult, error)
	ListSatisfactionRequests(context.Context, supervisionreq.ClientSatisfactionSearch) ([]supervisionres.ClientSatisfactionSummary, int64, error)
	GetSatisfactionRequest(context.Context, uint) (supervisionres.ClientSatisfactionDetail, error)
	SubmitSatisfactionResponse(context.Context, uint, string, supervisionreq.SubmitSatisfactionResponse) (supervisionres.SubmitSatisfactionResult, error)
}

var _ Service = (*ClientAccessService)(nil)

type ServiceGroup struct {
	ClientAccessService
}
