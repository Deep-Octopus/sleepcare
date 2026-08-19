package initialize

import (
	"context"
	"errors"
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
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/operations-dashboard", Description: "查询机构级运营概览"},
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/daily-summaries", Description: "查询每日汇总"},
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/daily-summaries/:id", Description: "查询日报版本详情"},
	{ApiGroup: "督导中心", Method: "POST", Path: "/care/daily-summaries/:id/revisions", Description: "追加日报修正版"},
	{ApiGroup: "督导中心", Method: "GET", Path: "/care/reviews", Description: "查询上级复核队列"},
	{ApiGroup: "督导中心", Method: "POST", Path: "/care/reviews/:id/guidance", Description: "追加上级指导"},
	{ApiGroup: "督导中心", Method: "POST", Path: "/care/reviews/:id/intervene", Description: "记录上级介入"},
	{ApiGroup: "服务评价", Method: "GET", Path: "/care/satisfaction-responses", Description: "查询匿名服务评价"},
	{ApiGroup: "服务评价", Method: "GET", Path: "/care/satisfaction-follow-ups", Description: "查询低分质量跟进"},
	{ApiGroup: "服务评价", Method: "GET", Path: "/care/satisfaction-follow-ups/:id", Description: "查询质量跟进详情"},
	{ApiGroup: "服务评价", Method: "POST", Path: "/care/satisfaction-follow-ups/:id/acknowledge", Description: "接收质量跟进"},
	{ApiGroup: "服务评价", Method: "POST", Path: "/care/satisfaction-follow-ups/:id/resolve", Description: "解决质量跟进"},
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
	if err := ensureDailySummaryTimedTask(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	if err := ensureSatisfactionPolicy(db); err != nil {
		return err
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
		root, center, dailyMenu, reviewMenu, satisfactionMenu, dailyButtons, reviewButtons, satisfactionButtons, err := ensureSupervisionMenus(tx)
		if err != nil {
			return err
		}
		if !grantPolicies {
			return nil
		}
		for _, menuID := range []uint{root.ID, center.ID, dailyMenu.ID, reviewMenu.ID, satisfactionMenu.ID} {
			link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(syntheticSupervisorRole)}
			if err = tx.Where(link).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
		buttons := append(append(dailyButtons, reviewButtons...), satisfactionButtons...)
		for _, button := range buttons {
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
	satisfactionMenu system.SysBaseMenu,
	dailyButtons []system.SysBaseMenuBtn,
	reviewButtons []system.SysBaseMenuBtn,
	satisfactionButtons []system.SysBaseMenuBtn,
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
	satisfactionMenu = system.SysBaseMenu{
		ParentId: center.ID, Path: "satisfaction", Name: "CareSatisfaction",
		Component: "view/sleep-care/satisfaction/index.vue", Sort: 3,
		Meta: system.Meta{Title: "服务评价", Icon: "star"},
	}
	if err = ensureCaseWorkMenu(tx, &satisfactionMenu); err != nil {
		return
	}
	dailyButtons = []system.SysBaseMenuBtn{
		{Name: "viewDetail", Desc: "查看日报版本详情", SysBaseMenuID: dailyMenu.ID},
		{Name: "revise", Desc: "根据修正记录追加日报版本", SysBaseMenuID: dailyMenu.ID},
	}
	reviewButtons = []system.SysBaseMenuBtn{
		{Name: "viewDetail", Desc: "查看待复核事项详情", SysBaseMenuID: reviewMenu.ID},
		{Name: "guide", Desc: "追加上级指导", SysBaseMenuID: reviewMenu.ID},
		{Name: "discuss", Desc: "要求流程讨论", SysBaseMenuID: reviewMenu.ID},
		{Name: "intervene", Desc: "记录上级直接介入", SysBaseMenuID: reviewMenu.ID},
	}
	satisfactionButtons = []system.SysBaseMenuBtn{
		{Name: "viewFollowUp", Desc: "查看质量跟进详情", SysBaseMenuID: satisfactionMenu.ID},
		{Name: "acknowledgeFollowUp", Desc: "接收质量跟进", SysBaseMenuID: satisfactionMenu.ID},
		{Name: "resolveFollowUp", Desc: "解决质量跟进", SysBaseMenuID: satisfactionMenu.ID},
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
	for i := range satisfactionButtons {
		if err = tx.Where("name = ? AND sys_base_menu_id = ?", satisfactionButtons[i].Name, satisfactionMenu.ID).
			Attrs(satisfactionButtons[i]).FirstOrCreate(&satisfactionButtons[i]).Error; err != nil {
			return
		}
	}
	return
}

func ensureSatisfactionPolicy(db *gorm.DB) error {
	expected := supervisionmodel.SatisfactionPolicy{
		Code:              "CONSULTATION-CLOSE-TEST",
		Version:           1,
		TriggerType:       supervisionmodel.SatisfactionTriggerConsultation,
		AnonymityMode:     supervisionmodel.SatisfactionAnonymousStaff,
		ValidForHours:     7 * 24,
		LowScoreThreshold: 2,
		EffectiveFrom:     time.Date(2026, time.August, 1, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
		Status:            supervisionmodel.SatisfactionPolicyStatusPublished,
		Synthetic:         true,
	}
	var actual supervisionmodel.SatisfactionPolicy
	if err := db.Where("code = ? AND version = ?", expected.Code, expected.Version).
		Attrs(expected).FirstOrCreate(&actual).Error; err != nil {
		return err
	}
	if actual.TriggerType != expected.TriggerType || actual.AnonymityMode != expected.AnonymityMode ||
		actual.ValidForHours != expected.ValidForHours || actual.LowScoreThreshold != expected.LowScoreThreshold ||
		!actual.EffectiveFrom.Equal(expected.EffectiveFrom) || actual.Status != expected.Status || !actual.Synthetic {
		return fmt.Errorf("satisfaction policy %s v%d differs", expected.Code, expected.Version)
	}
	return nil
}

func ensureInitialSupervisionSnapshot(ctx context.Context, db *gorm.DB) error {
	businessDate := time.Date(2026, time.August, 18, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	var count int64
	if err := db.Model(&supervisionmodel.DailySummaryVersion{}).
		Where("organization_id = ? AND business_date = ? AND metric_definition_version = ?",
			syntheticOrgAID, businessDate, supervisionmodel.MetricDefinitionVersionV2).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	fixedNow := time.Date(2026, time.August, 19, 0, 5, 0, 0, time.FixedZone("CST", 8*60*60))
	enabled := true
	_, _, err := (&supervisionservice.SupervisionService{
		DB: db, Now: func() time.Time { return fixedNow }, SyntheticFixturesEnabled: &enabled,
	}).EnsureScheduledSnapshot(ctx, syntheticOrgAID, businessDate)
	return err
}

func ensureDailySummaryTimedTask(db *gorm.DB, enabled bool) error {
	const taskName = "CareDailySummary"
	var existing system.SysTimedTask
	err := db.Where("name = ?", taskName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if !enabled {
			return nil
		}
		return db.Create(&system.SysTimedTask{
			Name:         taskName,
			Description:  "每日生成前一自然日的机构汇总；仅固定记录门禁开启时登记",
			Spec:         "CRON_TZ=Asia/Shanghai 10 0 * * *",
			ExecutorType: system.TimedTaskExecutorMethod,
			MethodName:   "GenerateCareDailySummaries",
			Enabled:      true,
		}).Error
	}
	if err != nil {
		return err
	}
	if existing.ExecutorType != system.TimedTaskExecutorMethod || existing.MethodName != "GenerateCareDailySummaries" {
		return fmt.Errorf("daily summary timed task executor differs from the registered method")
	}
	if !enabled && existing.Enabled {
		return db.Model(&system.SysTimedTask{}).Where("id = ?", existing.ID).Update("enabled", false).Error
	}
	return nil
}
