package casework

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/gorm"
)

type CaseWorkService struct {
	DB                       *gorm.DB
	Now                      func() time.Time
	SyntheticFixturesEnabled *bool
}

func (s *CaseWorkService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *CaseWorkService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *CaseWorkService) syntheticFixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}

type ServiceGroup struct {
	CaseWorkService
}
