package initialize

import (
	"context"
	"strconv"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPhaseOneAccessControlReconcilesExactMatrix(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthority{},
		&system.SysAuthorityMenu{}, &system.SysAuthorityBtn{}, &adapter.CasbinRule{},
	).WithContext(datascope.WithSystem(context.Background()))
	seedPhaseOneAuthorityRows(t, db)
	require.NoError(t, ensureCareClientMetadata(db))
	require.NoError(t, ensureQuestionnaireMetadata(db))
	require.NoError(t, ensureCarePathMetadata(db))
	require.NoError(t, ensureCaseWorkMetadata(db, true))
	require.NoError(t, ensureConsultationMetadata(db, true))
	require.NoError(t, ensureSupervisionMetadata(db, true))
	require.NoError(t, ensureNotificationMetadata(db, true))
	seedStalePhaseOneGrants(t, db)

	require.NoError(t, ensurePhaseOneAccessControl(db, true))
	firstCounts := phaseOneAccessCounts(t, db)
	require.NoError(t, ensurePhaseOneAccessControl(db, true))
	require.Equal(t, firstCounts, phaseOneAccessCounts(t, db))

	menus := loadPhaseOneMenus(t, db)
	assertPhaseOneMenuStructure(t, db, menus)
	assertPhaseOneRoleSettings(t, db, menus)
	assertPhaseOneMenuMatrix(t, db, menus)
	assertPhaseOneButtonMatrix(t, db, menus)
	assertPhaseOneAPIMatrix(t, db)
	assertAdministratorHasNoCareGrants(t, db, menus)
}

func TestPhaseOneAccessControlWithoutFixturesOnlyMaintainsMetadata(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthorityMenu{},
		&system.SysAuthorityBtn{}, &adapter.CasbinRule{},
	).WithContext(datascope.WithSystem(context.Background()))
	require.NoError(t, ensurePhaseOneAccessControl(db, false))

	menus := loadPhaseOneMenus(t, db)
	assertPhaseOneMenuStructure(t, db, menus)
	var grants int64
	require.NoError(t, db.Model(&system.SysAuthorityMenu{}).Count(&grants).Error)
	require.Zero(t, grants)
	require.NoError(t, db.Model(&system.SysAuthorityBtn{}).Count(&grants).Error)
	require.Zero(t, grants)
	require.NoError(t, db.Model(&adapter.CasbinRule{}).Count(&grants).Error)
	require.Zero(t, grants)
}

func seedPhaseOneAuthorityRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	parentID := uint(888)
	authorities := []system.SysAuthority{
		{AuthorityId: syntheticStewardRole, AuthorityName: "管家测试角色", ParentId: &parentID},
		{AuthorityId: syntheticClinicianRole, AuthorityName: "医护测试角色", ParentId: &parentID},
		{AuthorityId: syntheticSupervisorRole, AuthorityName: "上级测试角色", ParentId: &parentID},
		{AuthorityId: phaseOneContentAdminRole, AuthorityName: "内容测试角色", ParentId: &parentID},
		{AuthorityId: 888, AuthorityName: "系统管理角色", ParentId: new(uint)},
	}
	require.NoError(t, db.Create(&authorities).Error)
}

func seedStalePhaseOneGrants(t *testing.T, db *gorm.DB) {
	t.Helper()
	var root, clientMenu system.SysBaseMenu
	require.NoError(t, db.Where("name = ?", "SleepCare").First(&root).Error)
	require.NoError(t, db.Where("name = ?", "CareClients").First(&clientMenu).Error)
	var viewButton system.SysBaseMenuBtn
	require.NoError(t, db.Where("sys_base_menu_id = ? AND name = ?", clientMenu.ID, "viewDetail").First(&viewButton).Error)
	staleMenus := []system.SysAuthorityMenu{
		{MenuId: strconv.FormatUint(uint64(clientMenu.ID), 10), AuthorityId: strconv.FormatUint(uint64(phaseOneContentAdminRole), 10)},
		{MenuId: strconv.FormatUint(uint64(root.ID), 10), AuthorityId: "888"},
	}
	require.NoError(t, db.Create(&staleMenus).Error)
	staleButtons := []system.SysAuthorityBtn{
		{AuthorityId: phaseOneContentAdminRole, SysMenuID: clientMenu.ID, SysBaseMenuBtnID: viewButton.ID},
		{AuthorityId: 888, SysMenuID: clientMenu.ID, SysBaseMenuBtnID: viewButton.ID},
	}
	require.NoError(t, db.Create(&staleButtons).Error)
	stalePolicies := []adapter.CasbinRule{
		{Ptype: "p", V0: strconv.FormatUint(uint64(phaseOneContentAdminRole), 10), V1: "/care/tasks", V2: "GET"},
		{Ptype: "p", V0: strconv.FormatUint(uint64(syntheticStewardRole), 10), V1: "/system/getSystemConfig", V2: "POST"},
		{Ptype: "p", V0: "888", V1: "/care/clients", V2: "GET"},
	}
	require.NoError(t, db.Create(&stalePolicies).Error)
}

type phaseOneCounts struct {
	Menus        int64
	Buttons      int64
	MenuGrants   int64
	ButtonGrants int64
	Policies     int64
}

func phaseOneAccessCounts(t *testing.T, db *gorm.DB) phaseOneCounts {
	t.Helper()
	counts := phaseOneCounts{}
	require.NoError(t, db.Model(&system.SysBaseMenu{}).Count(&counts.Menus).Error)
	require.NoError(t, db.Model(&system.SysBaseMenuBtn{}).Count(&counts.Buttons).Error)
	require.NoError(t, db.Model(&system.SysAuthorityMenu{}).Count(&counts.MenuGrants).Error)
	require.NoError(t, db.Model(&system.SysAuthorityBtn{}).Count(&counts.ButtonGrants).Error)
	require.NoError(t, db.Model(&adapter.CasbinRule{}).Count(&counts.Policies).Error)
	return counts
}

func loadPhaseOneMenus(t *testing.T, db *gorm.DB) map[string]system.SysBaseMenu {
	t.Helper()
	menus := make(map[string]system.SysBaseMenu, len(phaseOneMenuSpecs))
	for _, spec := range phaseOneMenuSpecs {
		var menu system.SysBaseMenu
		require.NoError(t, db.Where("name = ?", spec.Name).First(&menu).Error)
		menus[spec.Name] = menu
	}
	return menus
}

func assertPhaseOneMenuStructure(t *testing.T, db *gorm.DB, menus map[string]system.SysBaseMenu) {
	t.Helper()
	for _, spec := range phaseOneMenuSpecs {
		menu := menus[spec.Name]
		require.Equal(t, spec.Path, menu.Path, spec.Name)
		require.Equal(t, spec.Component, menu.Component, spec.Name)
		require.Equal(t, spec.Hidden, menu.Hidden, spec.Name)
		require.Equal(t, spec.ActiveName, menu.Meta.ActiveName, spec.Name)
		if spec.ParentName == "" {
			require.Zero(t, menu.ParentId, spec.Name)
		} else {
			require.Equal(t, menus[spec.ParentName].ID, menu.ParentId, spec.Name)
		}
	}
	for _, name := range []string{
		"CareClientDetail",
		"CareTaskDetail",
		"CareAttentionCaseDetail",
		"CareConsultationDetail",
		"CareReviewDetail",
	} {
		require.True(t, menus[name].Hidden, name)
		require.NotEmpty(t, menus[name].Meta.ActiveName, name)
	}
}

func assertPhaseOneRoleSettings(t *testing.T, db *gorm.DB, menus map[string]system.SysBaseMenu) {
	t.Helper()
	for _, setting := range phaseOneRoleSettings {
		var authority system.SysAuthority
		require.NoError(t, db.First(&authority, "authority_id = ?", setting.AuthorityID).Error)
		require.Equal(t, setting.DataScope, authority.DataScope)
		require.Equal(t, setting.DefaultRoute, authority.DefaultRouter)
		defaultMenu := menus[setting.DefaultRoute]
		require.False(t, defaultMenu.Hidden)
		var visibleChildren int64
		require.NoError(t, db.Model(&system.SysBaseMenu{}).
			Where("parent_id = ? AND hidden = ?", defaultMenu.ID, false).Count(&visibleChildren).Error)
		require.Zero(t, visibleChildren, "default route %s must be a visible leaf", setting.DefaultRoute)
		assertMenuGrant(t, db, setting.AuthorityID, defaultMenu.ID, true)
	}
}

func assertPhaseOneMenuMatrix(t *testing.T, db *gorm.DB, menus map[string]system.SysBaseMenu) {
	t.Helper()
	roleIDs, _ := phaseOneRoleIdentifiers()
	for _, roleID := range roleIDs {
		allowed := stringSet(phaseOneMenuAccess[roleID])
		for _, spec := range phaseOneMenuSpecs {
			_, expected := allowed[spec.Name]
			assertMenuGrant(t, db, roleID, menus[spec.Name].ID, expected)
		}
	}
}

func assertPhaseOneButtonMatrix(t *testing.T, db *gorm.DB, menus map[string]system.SysBaseMenu) {
	t.Helper()
	roleIDs, _ := phaseOneRoleIdentifiers()
	for _, roleID := range roleIDs {
		allowed := make(map[string]struct{})
		grants := append([]phaseOneButtonGrant(nil), phaseOneButtonAccess[roleID]...)
		grants = append(grants, phaseOneHiddenButtonGrants(roleID)...)
		for _, grant := range grants {
			allowed[grant.MenuName+"."+grant.ButtonName] = struct{}{}
		}
		for menuName, menu := range menus {
			var buttons []system.SysBaseMenuBtn
			require.NoError(t, db.Where("sys_base_menu_id = ?", menu.ID).Find(&buttons).Error)
			for _, button := range buttons {
				_, expected := allowed[menuName+"."+button.Name]
				assertButtonGrant(t, db, roleID, menu.ID, button.ID, expected)
			}
		}
	}
}

func assertPhaseOneAPIMatrix(t *testing.T, db *gorm.DB) {
	t.Helper()
	all := make(map[string]phaseOneAPIGrant)
	for _, roleGrants := range phaseOneAPIAccess {
		for _, grant := range roleGrants {
			all[grant.Method+" "+grant.Path] = grant
		}
	}
	roleIDs, _ := phaseOneRoleIdentifiers()
	for _, roleID := range roleIDs {
		allowed := make(map[string]struct{})
		for _, grant := range phaseOneAPIAccess[roleID] {
			allowed[grant.Method+" "+grant.Path] = struct{}{}
		}
		for key, grant := range all {
			_, expected := allowed[key]
			assertPolicy(t, db, roleID, grant.Path, grant.Method, expected)
		}
		for _, grant := range phaseOneShellAccess {
			assertPolicy(t, db, roleID, grant.Path, grant.Method, true)
		}
	}
	assertPolicy(t, db, phaseOneContentAdminRole, "/care/tasks", "GET", false)
	assertPolicy(t, db, syntheticStewardRole, "/system/getSystemConfig", "POST", false)
}

func assertAdministratorHasNoCareGrants(t *testing.T, db *gorm.DB, menus map[string]system.SysBaseMenu) {
	t.Helper()
	for _, menu := range menus {
		assertMenuGrant(t, db, 888, menu.ID, false)
	}
	var buttonCount int64
	require.NoError(t, db.Model(&system.SysAuthorityBtn{}).Where("authority_id = ?", 888).Count(&buttonCount).Error)
	require.Zero(t, buttonCount)
	var policyCount int64
	require.NoError(t, db.Model(&adapter.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 LIKE ?", "p", "888", "/care/%").Count(&policyCount).Error)
	require.Zero(t, policyCount)
}

func assertMenuGrant(t *testing.T, db *gorm.DB, roleID, menuID uint, expected bool) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&system.SysAuthorityMenu{}).
		Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", strconv.FormatUint(uint64(roleID), 10), strconv.FormatUint(uint64(menuID), 10)).
		Count(&count).Error)
	require.Equal(t, expected, count == 1, "menu grant role=%d menu=%d count=%d", roleID, menuID, count)
}

func assertButtonGrant(t *testing.T, db *gorm.DB, roleID, menuID, buttonID uint, expected bool) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&system.SysAuthorityBtn{}).
		Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", roleID, menuID, buttonID).
		Count(&count).Error)
	require.Equal(t, expected, count == 1, "button grant role=%d menu=%d button=%d count=%d", roleID, menuID, buttonID, count)
}

func assertPolicy(t *testing.T, db *gorm.DB, roleID uint, path, method string, expected bool) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&adapter.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", strconv.FormatUint(uint64(roleID), 10), path, method).
		Count(&count).Error)
	require.Equal(t, expected, count == 1, "policy role=%d %s %s count=%d", roleID, method, path, count)
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
