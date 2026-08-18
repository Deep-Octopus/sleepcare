package initialize

import (
	"context"
	"testing"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestEnsureCareClientSyntheticFixturesIsIdempotentAndDoesNotGrantAdmin(t *testing.T) {
	db := testutil.NewMemoryDB(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthority{}, &system.SysDepartment{},
		&system.SysUser{}, &system.SysUserAuthority{}, &system.SysUserDepartment{}, &system.SysAuthorityMenu{}, &system.SysAuthorityBtn{},
		&adapter.CasbinRule{},
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.ConsentRecord{}, &caremodel.CareOrgUnitProfile{}, &caremodel.CareAuthorityProfile{}, &caremodel.CareClientCommandReceipt{},
	)
	db = db.WithContext(datascope.WithSystem(context.Background()))
	for i := 0; i < 2; i++ {
		if err := ensureCareClientMetadata(db); err != nil {
			t.Fatal(err)
		}
		if err := ensureCareClientSyntheticFixtures(db, "local-synthetic-password"); err != nil {
			t.Fatal(err)
		}
	}

	assertCount := func(model any, want int64) {
		t.Helper()
		var got int64
		if err := db.Model(model).Count(&got).Error; err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%T count=%d, want %d", model, got, want)
		}
	}
	assertCount(&caremodel.CareClient{}, 3)
	assertCount(&caremodel.CareAssignment{}, 4)
	assertCount(&caremodel.ConsentRecord{}, 1)
	assertCount(&caremodel.CareOrgUnitProfile{}, 4)
	assertCount(&caremodel.CareAuthorityProfile{}, 4)

	notificationFinalAt := time.Date(2026, time.August, 18, 8, 56, 0, 0, time.FixedZone("CST", 8*60*60))
	var activeScenarioBStewards int64
	if err := db.Model(&caremodel.CareAssignment{}).
		Where("care_client_id = ? AND role_type = ? AND valid_from <= ? AND cancelled_at IS NULL", 20002, caremodel.AssignmentRoleCareSteward, notificationFinalAt).
		Where("valid_until IS NULL OR valid_until > ?", notificationFinalAt).
		Count(&activeScenarioBStewards).Error; err != nil {
		t.Fatal(err)
	}
	if activeScenarioBStewards != 1 {
		t.Fatalf("scenario B needs one active steward at the failure instant, got %d", activeScenarioBStewards)
	}

	var contentProfile caremodel.CareAuthorityProfile
	if err := db.Where("authority_id = ?", phaseOneContentAdminRole).First(&contentProfile).Error; err != nil {
		t.Fatal(err)
	}
	if contentProfile.RoleType != caremodel.AuthorityRoleContentAdmin || !contentProfile.Active || !contentProfile.Synthetic {
		t.Fatalf("content administrator profile differs: %+v", contentProfile)
	}
	var contentUser system.SysUser
	if err := db.First(&contentUser, phaseOneContentAdminAID).Error; err != nil {
		t.Fatal(err)
	}
	if contentUser.AuthorityId != phaseOneContentAdminRole || contentUser.Username != "test_content_admin_a" {
		t.Fatalf("content administrator fixture differs: %+v", contentUser)
	}

	var adminMenus int64
	if err := db.Model(&system.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", "888").Count(&adminMenus).Error; err != nil {
		t.Fatal(err)
	}
	if adminMenus != 0 {
		t.Fatalf("admin received %d care menu grants", adminMenus)
	}
	var adminPolicies int64
	if err := db.Model(&adapter.CasbinRule{}).Where("v0 = ? AND v1 LIKE ?", "888", "/care/%").Count(&adminPolicies).Error; err != nil {
		t.Fatal(err)
	}
	if adminPolicies != 0 {
		t.Fatalf("admin received %d care API policies", adminPolicies)
	}
}
