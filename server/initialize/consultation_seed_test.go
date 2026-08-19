package initialize

import (
	"context"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/stretchr/testify/require"
)

func TestEnsureConsultationMetadataIsIdempotentAndRoleScoped(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{},
		&system.SysAuthorityMenu{}, &system.SysAuthorityBtn{}, &adapter.CasbinRule{},
	).WithContext(datascope.WithSystem(context.Background()))
	require.NoError(t, ensureConsultationMetadata(db, true))
	require.NoError(t, ensureConsultationMetadata(db, true))

	var apiCount int64
	require.NoError(t, db.Model(&system.SysApi{}).
		Where("path LIKE ?", "/care/consultations%").Count(&apiCount).Error)
	require.Equal(t, int64(len(consultationAPIs)), apiCount)

	for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole} {
		assertCaseWorkPolicy(t, db, role, "/care/consultations", "GET", true)
		assertCaseWorkPolicy(t, db, role, "/care/consultations/:id/assignee-options", "GET", true)
	}
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/consultations/:id/assign", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/consultations/:id/reopen", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/consultations/:id/assign", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/consultations/:id/escalate", "POST", false)
	assertCaseWorkPolicy(t, db, uint(888), "/care/consultations", "GET", false)

	var executionMenu, listMenu, detailMenu system.SysBaseMenu
	require.NoError(t, db.Where("name = ?", "CareExecution").First(&executionMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareConsultations").First(&listMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareConsultationDetail").First(&detailMenu).Error)
	require.Equal(t, executionMenu.ID, listMenu.ParentId)
	require.Equal(t, executionMenu.ID, detailMenu.ParentId)
	require.True(t, detailMenu.Hidden)
	require.Equal(t, "CareConsultations", detailMenu.Meta.ActiveName)

	assertCaseWorkButton(t, db, listMenu.ID, "viewDetail", syntheticStewardRole, true)
	assertCaseWorkButton(t, db, listMenu.ID, "assign", syntheticStewardRole, false)
	assertCaseWorkButton(t, db, listMenu.ID, "escalate", syntheticClinicianRole, true)
	assertCaseWorkButton(t, db, listMenu.ID, "reopen", syntheticClinicianRole, false)
	assertCaseWorkButton(t, db, listMenu.ID, "assign", syntheticSupervisorRole, true)
	assertCaseWorkButton(t, db, listMenu.ID, "escalate", syntheticSupervisorRole, false)
	assertCaseWorkButton(t, db, listMenu.ID, "reopen", syntheticSupervisorRole, true)
}
