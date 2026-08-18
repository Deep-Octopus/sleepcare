package initialize

import (
	"context"
	"fmt"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	supervisionservice "github.com/flipped-aurora/gin-vue-admin/server/service/supervision"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

var supervisionAPIs = []system.SysApi{
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/daily-summaries", Description: "查询每日汇总"},
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/daily-summaries/:id", Description: "查询日报版本详情"},
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/reviews", Description: "查询上级复核队列"},
	{ApiGroup: "督导中心", Method: "POST", Path: "/care/reviews/:id/guidance", Description: "追加上级指导"},
	{ApiGroup: "督导中心", Method: "POST", Path: "/care/reviews/:id/intervene", Description: "记录上级介入"},
}

func EnsureSupervisionData() error {
	if global.GVA_DB == nil {
		return nil
	}
	ctx := datascope.WithSystem(context.Background())
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureSupervisionMetadata(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	return ensureInitialSupervisionSnapshot(ctx, db)
}

func ensureSupervisionMetadata(db *gorm.DB, grantPolicies bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range supervisionAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).
				Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure supervision API %s: %w", api.Path, err)
			}
		}
		root, center, dailyMenu, reviewMenu, dailyButtons, reviewButtons, err := ensureSupervisionMenus(tx)
		if err != nil {
			return err
		}
		if !grantPolicies {
			return nil
		}
		for _, menuID := range []uint{root.ID, center.ID, dailyMenu.ID, reviewMenu.ID} {
			link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(syntheticSupervisorRole)}
			if err = tx.Where(link).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
		for _, button := range append(dailyButtons, reviewButtons...) {
			if err = ensureAuthorityButton(tx, syntheticSupervisorRole, button.SysBaseMenuID, button.ID); err != nil {
				return err
			}
		}
		for _, api := range supervisionAPIs {
			policy := adapter.CasbinRule{
				Ptype: "p", V0: fmt.Sprint(syntheticSupervisorRole), V1: api.Path, V2: api.Method,
			}
			if err = tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureSupervisionMenus(tx *gorm.DB) (
	root system.SysBaseMenu,
	center system.SysBaseMenu,
	dailyMenu system.SysBaseMenu,
	reviewMenu system.SysBaseMenu,
	dailyButtons []system.SysBaseMenuBtn,
	reviewButtons []system.SysBaseMenuBtn,
	err error,
) {
	if err = tx.Where("name = ?", "SleepCare").First(&root).Error; err != nil {
		return
	}
	center = system.SysBaseMenu{
		ParentId: root.ID, Path: "care-supervision", Name: "CareSupervision",
		Component: "view/routerHolder.vue", Sort: 5,
		Meta: system.Meta{Title: "督导中心", Icon: "monitor-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &center); err != nil {
		return
	}
	dailyMenu = system.SysBaseMenu{
		ParentId: center.ID, Path: "daily-summaries", Name: "CareDailySummaries",
		Component: "view/sleep-care/daily-summaries/index.vue", Sort: 1,
		Meta: system.Meta{Title: "每日汇总", Icon: "version-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &dailyMenu); err != nil {
		return
	}
	reviewMenu = system.SysBaseMenu{
		ParentId: center.ID, Path: "review-queue", Name: "CareReviewQueue",
		Component: "view/sleep-care/review-queue/index.vue", Sort: 2,
		Meta: system.Meta{Title: "待复核事项", Icon: "error-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &reviewMenu); err != nil {
		return
	}
	dailyButtons = []system.SysBaseMenuBtn{
		{Name: "viewDetail", Desc: "查看日报版本详情", SysBaseMenuID: dailyMenu.ID},
	}
	reviewButtons = []system.SysBaseMenuBtn{
		{Name: "viewDetail", Desc: "查看待复核事项详情", SysBaseMenuID: reviewMenu.ID},
		{Name: "guide", Desc: "追加上级指导", SysBaseMenuID: reviewMenu.ID},
		{Name: "discuss", Desc: "要求流程讨论", SysBaseMenuID: reviewMenu.ID},
		{Name: "intervene", Desc: "记录上级直接介入", SysBaseMenuID: reviewMenu.ID},
	}
	for i := range dailyButtons {
		if err = tx.Where("name = ? AND sys_base_menu_id = ?", dailyButtons[i].Name, dailyMenu.ID).
			Attrs(dailyButtons[i]).FirstOrCreate(&dailyButtons[i]).Error; err != nil {
			return
		}
	}
	for i := range reviewButtons {
		if err = tx.Where("name = ? AND sys_base_menu_id = ?", reviewButtons[i].Name, reviewMenu.ID).
			Attrs(reviewButtons[i]).FirstOrCreate(&reviewButtons[i]).Error; err != nil {
			return
		}
	}
	return
}

func ensureInitialSupervisionSnapshot(ctx context.Context, db *gorm.DB) error {
	businessDate := time.Date(2026, time.August, 18, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	var count int64
	if err := db.Model(&supervisionmodel.DailySummaryVersion{}).
		Where("organization_id = ? AND business_date = ?", syntheticOrgAID, businessDate).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	fixedNow := time.Date(2026, time.August, 19, 0, 5, 0, 0, time.FixedZone("CST", 8*60*60))
	_, err := (&supervisionservice.SupervisionService{
		DB: db, Now: func() time.Time { return fixedNow },
	}).GenerateSnapshot(ctx, syntheticOrgAID, businessDate)
	return err
}
