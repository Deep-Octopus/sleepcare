package questionnaire

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/gorm"
)

type QuestionnaireService struct {
	DB                       *gorm.DB
	Now                      func() time.Time
	SyntheticFixturesEnabled *bool
}

func (s *QuestionnaireService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *QuestionnaireService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *QuestionnaireService) syntheticFixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}
