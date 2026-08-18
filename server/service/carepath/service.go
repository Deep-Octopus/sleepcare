package carepath

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	carereq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	careres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
	questionnaireservice "github.com/flipped-aurora/gin-vue-admin/server/service/questionnaire"
	"gorm.io/gorm"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type FixedClock struct{ Time time.Time }

func (c FixedClock) Now() time.Time { return c.Time }

type QuestionnaireBindingValidator interface {
	ValidateFrozenBinding(ctx context.Context, questionnaireVersionID uint, ruleVersionIDs []uint, synthetic bool) error
}

// Service exposes the care-path module as a small cohesive interface. HTTP,
// persistence, clock and questionnaire validation details stay behind it.
type Service interface {
	ListPlanVersions(context.Context, carereq.PlanVersionSearch) ([]careres.PlanVersionSummary, int64, error)
	GetPlanVersion(context.Context, uint) (careres.PlanVersionDetail, error)
	PreviewPlan(context.Context, uint, string, carereq.PreviewPlan) (careres.PlanPreview, error)
	StartPlan(context.Context, uint, string, carereq.StartPlan) (careres.PlanInstanceResult, error)
	ListClientPlans(context.Context, uint) ([]careres.PlanInstanceSummary, error)
	PausePlan(context.Context, uint, string, carereq.PlanStateAction) (careres.PlanActionResult, error)
	ResumePlan(context.Context, uint, string, carereq.PlanStateAction) (careres.PlanActionResult, error)
	ListTasks(context.Context, carereq.TaskSearch) ([]careres.TaskSummary, int64, error)
	GetTask(context.Context, uint) (careres.TaskDetail, error)
	ReconcilePlanTasks(context.Context, uint) error
}

type CarePathService struct {
	DB                       *gorm.DB
	Clock                    Clock
	BindingValidator         QuestionnaireBindingValidator
	SyntheticFixturesEnabled *bool
	PreviewTTL               time.Duration
}

var _ Service = (*CarePathService)(nil)

func (s *CarePathService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *CarePathService) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now()
	}
	return SystemClock{}.Now()
}

func (s *CarePathService) bindingValidator() QuestionnaireBindingValidator {
	if s.BindingValidator != nil {
		return s.BindingValidator
	}
	return &questionnaireservice.QuestionnaireService{}
}

func (s *CarePathService) syntheticFixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}

func (s *CarePathService) previewTTL() time.Duration {
	if s.PreviewTTL > 0 {
		return s.PreviewTTL
	}
	return 30 * time.Minute
}

type ServiceGroup struct {
	CarePathService
}
