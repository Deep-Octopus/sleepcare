package initialize

import (
	"context"
	"fmt"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnsureCaseWorkMetadataIsIdempotentAndRoleScoped(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthority{},
		&system.SysAuthorityMenu{}, &system.SysAuthorityBtn{}, &adapter.CasbinRule{},
	).WithContext(datascope.WithSystem(context.Background()))
	require.NoError(t, ensureCaseWorkMetadata(db, true))
	require.NoError(t, ensureCaseWorkMetadata(db, true))

	var apiCount int64
	require.NoError(t, db.Model(&system.SysApi{}).Where("path LIKE ?", "/care/attention-cases%").Count(&apiCount).Error)
	require.Equal(t, int64(7), apiCount)
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/workbench", "GET", true)
	assertCaseWorkPolicy(t, db, uint(888), "/care/workbench", "GET", false)

	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/attention-cases", "GET", true)
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/attention-cases/:id/acknowledge", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/attention-cases/:id/close", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/attention-cases/:id/close", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/attention-cases/:id/reopen", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/attention-cases/:id/acknowledge", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/attention-cases/:id/reopen", "POST", true)

	var executionMenu, taskMenu, caseMenu, workbenchMenu system.SysBaseMenu
	require.NoError(t, db.Where("name = ?", "CareExecution").First(&executionMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareTasks").First(&taskMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareAttentionCases").First(&caseMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareWorkbench").First(&workbenchMenu).Error)
	require.Equal(t, executionMenu.ID, taskMenu.ParentId)
	require.Equal(t, executionMenu.ID, caseMenu.ParentId)

	assertCaseWorkButton(t, db, workbenchMenu.ID, "viewDetail", syntheticStewardRole, true)
	assertCaseWorkButton(t, db, caseMenu.ID, "recordContact", syntheticStewardRole, true)
	assertCaseWorkButton(t, db, caseMenu.ID, "recordHandling", syntheticStewardRole, false)
	assertCaseWorkButton(t, db, caseMenu.ID, "recordContact", syntheticClinicianRole, false)
	assertCaseWorkButton(t, db, caseMenu.ID, "recordHandling", syntheticClinicianRole, true)
	assertCaseWorkButton(t, db, caseMenu.ID, "close", syntheticClinicianRole, true)
	assertCaseWorkButton(t, db, caseMenu.ID, "close", syntheticStewardRole, false)
	assertCaseWorkButton(t, db, caseMenu.ID, "reopen", syntheticSupervisorRole, true)
}

func assertCaseWorkPolicy(t *testing.T, db *gorm.DB, role uint, path, method string, want bool) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&adapter.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", fmt.Sprint(role), path, method).
		Count(&count).Error)
	if want {
		require.Equal(t, int64(1), count)
		return
	}
	require.Zero(t, count)
}

func assertCaseWorkButton(t *testing.T, db *gorm.DB, menuID uint, name string, role uint, want bool) {
	t.Helper()
	var button system.SysBaseMenuBtn
	require.NoError(t, db.Where("sys_base_menu_id = ? AND name = ?", menuID, name).First(&button).Error)
	var count int64
	require.NoError(t, db.Model(&system.SysAuthorityBtn{}).
		Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", role, menuID, button.ID).
		Count(&count).Error)
	if want {
		require.Equal(t, int64(1), count)
		return
	}
	require.Zero(t, count)
}
