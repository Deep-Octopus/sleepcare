package supervision

import (
	"context"
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/accesspolicy"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	"gorm.io/gorm"
)

var summaryLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}()

type SupervisionService struct {
	DB                       *gorm.DB
	Now                      func() time.Time
	SyntheticFixturesEnabled *bool
}

func (s *SupervisionService) syntheticFixturesEnabled() bool {
	if s.SyntheticFixturesEnabled != nil {
		return *s.SyntheticFixturesEnabled
	}
	return global.GVA_CONFIG.Care.SyntheticFixturesEnabled
}

func (s *SupervisionService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	return global.GVA_DB
}

func (s *SupervisionService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return global.GVA_CONFIG.Care.Now()
}

func (s *SupervisionService) supervisorScope(ctx context.Context) (*accesspolicy.CareClientDecision, uint, error) {
	decision, err := accesspolicy.ResolveCareClient(ctx, s.db())
	if err != nil {
		return nil, 0, normalizeAccessError(err)
	}
	if decision.RoleType != caremodel.AuthorityRoleSupervisor {
		return nil, 0, supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "当前角色没有上级督导权限")
	}
	var unit caremodel.CareOrgUnitProfile
	err = s.db().WithContext(ctx).
		Where("department_id = ? AND active = ? AND synthetic = ?", decision.Identity.DeptID, true, true).
		First(&unit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "当前部门没有可用的督导范围")
	}
	if err != nil {
		return nil, 0, err
	}
	if unit.OrganizationID == 0 || !decision.CanAccessDepartment(unit.DepartmentID) {
		return nil, 0, supervisionmodel.NewForbiddenError(supervisionmodel.CodeReviewScopeDenied, "当前部门不在可管理范围内")
	}
	return decision, unit.OrganizationID, nil
}

func normalizeAccessError(err error) error {
	var domainErr *caremodel.DomainError
	if !errors.As(err, &domainErr) {
		return err
	}
	return &supervisionmodel.DomainError{
		Code: domainErr.Code, Message: domainErr.Message, HTTPStatus: domainErr.HTTPStatus,
	}
}

type ServiceGroup struct {
	SupervisionService
}
