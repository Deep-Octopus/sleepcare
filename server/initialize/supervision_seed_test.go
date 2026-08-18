package initialize

import (
	"context"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"github.com/stretchr/testify/require"
)

func TestEnsureSupervisionMetadataIsIdempotentAndSupervisorOnly(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthority{},
		&system.SysAuthorityMenu{}, &system.SysAuthorityBtn{}, &adapter.CasbinRule{},
	).WithContext(datascope.WithSystem(context.Background()))
	require.NoError(t, db.Create(&system.SysBaseMenu{
		Path: "sleep-care", Name: "SleepCare", Component: "view/routerHolder.vue",
		Meta: system.Meta{Title: "睡眠照护"},
	}).Error)

	require.NoError(t, ensureSupervisionMetadata(db, true))
	require.NoError(t, ensureSupervisionMetadata(db, true))

	var apiCount int64
	require.NoError(t, db.Model(&system.SysApi{}).
		Where("path LIKE ?", "/care/daily-summaries%").
		Or("path LIKE ?", "/care/reviews%").
		Count(&apiCount).Error)
	require.Equal(t, int64(5), apiCount)
	for _, api := range supervisionAPIs {
		assertCaseWorkPolicy(t, db, syntheticSupervisorRole, api.Path, api.Method, true)
		assertCaseWorkPolicy(t, db, syntheticClinicianRole, api.Path, api.Method, false)
		assertCaseWorkPolicy(t, db, syntheticStewardRole, api.Path, api.Method, false)
	}

	var root, center, dailyMenu, reviewMenu system.SysBaseMenu
	require.NoError(t, db.Where("name = ?", "SleepCare").First(&root).Error)
	require.NoError(t, db.Where("name = ?", "CareSupervision").First(&center).Error)
	require.NoError(t, db.Where("name = ?", "CareDailySummaries").First(&dailyMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareReviewQueue").First(&reviewMenu).Error)
	require.Equal(t, root.ID, center.ParentId)
	require.Equal(t, center.ID, dailyMenu.ParentId)
	require.Equal(t, center.ID, reviewMenu.ParentId)

	assertCaseWorkButton(t, db, dailyMenu.ID, "viewDetail", syntheticSupervisorRole, true)
	for _, name := range []string{"viewDetail", "guide", "discuss", "intervene"} {
		assertCaseWorkButton(t, db, reviewMenu.ID, name, syntheticSupervisorRole, true)
	}
}

func TestEnsureInitialSupervisionSnapshotIsIdempotent(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&caremodel.CareClient{}, &supervisionmodel.DailySummaryVersion{},
	).WithContext(datascope.WithSystem(context.Background()))
	ctx := datascope.WithSystem(context.Background())

	require.NoError(t, ensureInitialSupervisionSnapshot(ctx, db))
	require.NoError(t, ensureInitialSupervisionSnapshot(ctx, db))

	var snapshots []supervisionmodel.DailySummaryVersion
	require.NoError(t, db.Order("id ASC").Find(&snapshots).Error)
	require.Len(t, snapshots, 1)
	require.Equal(t, uint(syntheticOrgAID), snapshots[0].OrganizationID)
	require.Equal(t, uint(1), snapshots[0].Version)
	require.Equal(t, supervisionmodel.MetricDefinitionVersionV1, snapshots[0].MetricDefinitionVersion)
	require.True(t, snapshots[0].Synthetic)
}
