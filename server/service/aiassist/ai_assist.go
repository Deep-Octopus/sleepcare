package aiassist

import (
	"context"

	careconfig "github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	aiassistres "github.com/flipped-aurora/gin-vue-admin/server/model/aiassist/response"
	"gorm.io/gorm"
)

type Service interface {
	GetShadowReadiness(context.Context) (aiassistres.ShadowReadiness, error)
}

type AIShadowService struct {
	DB     *gorm.DB
	Config *careconfig.AIShadow
}

var _ Service = (*AIShadowService)(nil)

func (s *AIShadowService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *AIShadowService) config() careconfig.AIShadow {
	if s.Config != nil {
		return *s.Config
	}
	return global.GVA_CONFIG.Care.AIShadow
}

type ServiceGroup struct {
	AIShadowService
}
