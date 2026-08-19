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
		Or("path = ?", "/care/operations-dashboard").
		Or("path LIKE ?", "/care/reviews%").
		Or("path LIKE ?", "/care/satisfaction-%").
		Count(&apiCount).Error)
	require.Equal(t, int64(len(supervisionAPIs)), apiCount)
	for _, api := range supervisionAPIs {
		assertCaseWorkPolicy(t, db, syntheticSupervisorRole, api.Path, api.Method, true)
		assertCaseWorkPolicy(t, db, syntheticClinicianRole, api.Path, api.Method, false)
		assertCaseWorkPolicy(t, db, syntheticStewardRole, api.Path, api.Method, false)
	}

	var root, center, dailyMenu, reviewMenu, satisfactionMenu system.SysBaseMenu
	require.NoError(t, db.Where("name = ?", "SleepCare").First(&root).Error)
	require.NoError(t, db.Where("name = ?", "CareSupervision").First(&center).Error)
	require.NoError(t, db.Where("name = ?", "CareDailySummaries").First(&dailyMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareReviewQueue").First(&reviewMenu).Error)
	require.NoError(t, db.Where("name = ?", "CareSatisfaction").First(&satisfactionMenu).Error)
	require.Equal(t, root.ID, center.ParentId)
	require.Equal(t, center.ID, dailyMenu.ParentId)
	require.Equal(t, center.ID, reviewMenu.ParentId)
	require.Equal(t, center.ID, satisfactionMenu.ParentId)

	for _, name := range []string{"viewDetail", "revise"} {
		assertCaseWorkButton(t, db, dailyMenu.ID, name, syntheticSupervisorRole, true)
	}
	for _, name := range []string{"viewDetail", "guide", "discuss", "intervene"} {
		assertCaseWorkButton(t, db, reviewMenu.ID, name, syntheticSupervisorRole, true)
	}
	for _, name := range []string{"viewFollowUp", "acknowledgeFollowUp", "resolveFollowUp"} {
		assertCaseWorkButton(t, db, satisfactionMenu.ID, name, syntheticSupervisorRole, true)
		assertCaseWorkButton(t, db, satisfactionMenu.ID, name, syntheticClinicianRole, false)
		assertCaseWorkButton(t, db, satisfactionMenu.ID, name, syntheticStewardRole, false)
	}
}

func TestEnsureSatisfactionPolicyIsIdempotent(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t, &supervisionmodel.SatisfactionPolicy{}).
		WithContext(datascope.WithSystem(context.Background()))

	require.NoError(t, ensureSatisfactionPolicy(db))
	require.NoError(t, ensureSatisfactionPolicy(db))

	var policies []supervisionmodel.SatisfactionPolicy
	require.NoError(t, db.Order("id ASC").Find(&policies).Error)
	require.Len(t, policies, 1)
	require.Equal(t, "CONSULTATION-CLOSE-TEST", policies[0].Code)
	require.Equal(t, uint(1), policies[0].Version)
	require.Equal(t, uint(7*24), policies[0].ValidForHours)
	require.Equal(t, uint8(2), policies[0].LowScoreThreshold)
	require.Equal(t, supervisionmodel.SatisfactionAnonymousStaff, policies[0].AnonymityMode)
	require.True(t, policies[0].Synthetic)
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
	require.Equal(t, supervisionmodel.MetricDefinitionVersionV2, snapshots[0].MetricDefinitionVersion)
	require.Equal(t, supervisionmodel.SummaryGenerationScheduled, snapshots[0].GenerationType)
	require.True(t, snapshots[0].Synthetic)
}

func TestEnsureDailySummaryTimedTaskIsGatedAndIdempotent(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t, &system.SysTimedTask{}).
		WithContext(datascope.WithSystem(context.Background()))

	require.NoError(t, ensureDailySummaryTimedTask(db, false))
	var count int64
	require.NoError(t, db.Model(&system.SysTimedTask{}).Count(&count).Error)
	require.Zero(t, count)

	require.NoError(t, ensureDailySummaryTimedTask(db, true))
	require.NoError(t, ensureDailySummaryTimedTask(db, true))
	var scheduled system.SysTimedTask
	require.NoError(t, db.Where("name = ?", "CareDailySummary").First(&scheduled).Error)
	require.True(t, scheduled.Enabled)
	require.Equal(t, system.TimedTaskExecutorMethod, scheduled.ExecutorType)
	require.Equal(t, "GenerateCareDailySummaries", scheduled.MethodName)
	require.Equal(t, "CRON_TZ=Asia/Shanghai 10 0 * * *", scheduled.Spec)

	require.NoError(t, ensureDailySummaryTimedTask(db, false))
	require.NoError(t, db.First(&scheduled, scheduled.ID).Error)
	require.False(t, scheduled.Enabled)
}
