package initialize

import (
	"context"
	"fmt"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	caseworkservice "github.com/flipped-aurora/gin-vue-admin/server/service/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

var caseWorkAPIs = []system.SysApi{
	{ApiGroup: "工作台", Method: "GET", Path: "/care/workbench", Description: "获取当前员工责任范围工作台"},
	{ApiGroup: "关注事项", Method: "GET", Path: "/care/attention-cases", Description: "获取授权范围关注事项列表"},
	{ApiGroup: "关注事项", Method: "GET", Path: "/care/attention-cases/:id", Description: "获取授权范围关注事项详情"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/acknowledge", Description: "确认关注事项"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/handling-records", Description: "追加关注事项处理记录"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/escalate", Description: "升级关注事项"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/close", Description: "关闭关注事项"},
	{ApiGroup: "关注事项", Method: "POST", Path: "/care/attention-cases/:id/reopen", Description: "重开关注事项"},
}

func EnsureCaseWorkData() error {
	if global.GVA_DB == nil {
		return nil
	}
	ctx := datascope.WithSystem(context.Background())
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureCaseWorkMetadata(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled); err != nil {
		return err
	}
	if err := ensureConsultationMetadata(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	enabled := true
	_, err := (&caseworkservice.CaseWorkService{DB: db, SyntheticFixturesEnabled: &enabled}).ReconcileRuleHits(ctx)
	return err
}

func ensureCaseWorkMetadata(db *gorm.DB, grantPolicies bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range caseWorkAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure attention case API %s: %w", api.Path, err)
			}
		}
		root, workbenchMenu, executionMenu, taskMenu, caseMenu, workbenchButtons, caseButtons, err := ensureCaseWorkMenus(tx)
		if err != nil {
			return err
		}
		if !grantPolicies {
			return nil
		}
		roles := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
		for _, role := range roles {
			for _, menuID := range []uint{root.ID, workbenchMenu.ID, executionMenu.ID, taskMenu.ID, caseMenu.ID} {
				link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(role)}
				if err = tx.Where(link).FirstOrCreate(&link).Error; err != nil {
					return err
				}
			}
		}
		for _, button := range workbenchButtons {
			for _, role := range roles {
				if err = ensureAuthorityButton(tx, role, workbenchMenu.ID, button.ID); err != nil {
					return err
				}
			}
		}
		buttonRoles := map[string][]uint{
			"viewDetail":     roles,
			"acknowledge":    {syntheticStewardRole, syntheticClinicianRole},
			"recordContact":  {syntheticStewardRole},
			"recordHandling": {syntheticClinicianRole},
			"escalate":       {syntheticStewardRole, syntheticClinicianRole},
			"close":          {syntheticClinicianRole, syntheticSupervisorRole},
			"reopen":         {syntheticSupervisorRole},
		}
		for _, button := range caseButtons {
			for _, role := range buttonRoles[button.Name] {
				if err = ensureAuthorityButton(tx, role, caseMenu.ID, button.ID); err != nil {
					return err
				}
			}
		}
		rolesByRoute := map[string][]uint{
			"GET /care/workbench":                             {syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole},
			"GET /care/attention-cases":                       {syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole},
			"GET /care/attention-cases/:id":                   {syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole},
			"POST /care/attention-cases/:id/acknowledge":      {syntheticStewardRole, syntheticClinicianRole},
			"POST /care/attention-cases/:id/handling-records": {syntheticStewardRole, syntheticClinicianRole},
			"POST /care/attention-cases/:id/escalate":         {syntheticStewardRole, syntheticClinicianRole},
			"POST /care/attention-cases/:id/close":            {syntheticClinicianRole, syntheticSupervisorRole},
			"POST /care/attention-cases/:id/reopen":           {syntheticSupervisorRole},
		}
		for _, api := range caseWorkAPIs {
			for _, role := range rolesByRoute[api.Method+" "+api.Path] {
				policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
				if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ensureCaseWorkMenus(tx *gorm.DB) (
	root system.SysBaseMenu,
	workbenchMenu system.SysBaseMenu,
	executionMenu system.SysBaseMenu,
	taskMenu system.SysBaseMenu,
	caseMenu system.SysBaseMenu,
	workbenchButtons []system.SysBaseMenuBtn,
	caseButtons []system.SysBaseMenuBtn,
	err error,
) {
	root = system.SysBaseMenu{
		Path: "sleep-care", Name: "SleepCare", Component: "view/routerHolder.vue", Sort: 20,
		Meta: system.Meta{Title: "睡眠康养随访", Icon: "user"},
	}
	if err = tx.Where("name = ?", root.Name).Attrs(root).FirstOrCreate(&root).Error; err != nil {
		return
	}
	workbenchMenu = system.SysBaseMenu{
		ParentId: root.ID, Path: "care-workbench", Name: "CareWorkbench",
		Component: "view/sleep-care/workbench/index.vue", Sort: 0,
		Meta: system.Meta{Title: "工作台", Icon: "monitor-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &workbenchMenu); err != nil {
		return
	}
	executionMenu = system.SysBaseMenu{
		ParentId: root.ID, Path: "care-execution", Name: "CareExecution",
		Component: "view/routerHolder.vue", Sort: 4,
		Meta: system.Meta{Title: "执行管理", Icon: "config-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &executionMenu); err != nil {
		return
	}
	taskMenu = system.SysBaseMenu{
		ParentId: executionMenu.ID, Path: "care-tasks", Name: "CareTasks",
		Component: "view/sleep-care/tasks/index.vue", Sort: 1,
		Meta: system.Meta{Title: "计划任务", Icon: "config-file-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &taskMenu); err != nil {
		return
	}
	caseMenu = system.SysBaseMenu{
		ParentId: executionMenu.ID, Path: "attention-cases", Name: "CareAttentionCases",
		Component: "view/sleep-care/attention-cases/index.vue", Sort: 2,
		Meta: system.Meta{Title: "关注事项", Icon: "error-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &caseMenu); err != nil {
		return
	}

	workbenchButtons = []system.SysBaseMenuBtn{
		{Name: "viewDetail", Desc: "查看工作台明细入口", SysBaseMenuID: workbenchMenu.ID},
	}
	caseButtons = []system.SysBaseMenuBtn{
		{Name: "viewDetail", Desc: "查看关注事项详情", SysBaseMenuID: caseMenu.ID},
		{Name: "acknowledge", Desc: "确认关注事项", SysBaseMenuID: caseMenu.ID},
		{Name: "recordContact", Desc: "记录非专业联系", SysBaseMenuID: caseMenu.ID},
		{Name: "recordHandling", Desc: "记录专业处置或请求复核", SysBaseMenuID: caseMenu.ID},
		{Name: "escalate", Desc: "升级关注事项", SysBaseMenuID: caseMenu.ID},
		{Name: "close", Desc: "关闭已解决事项", SysBaseMenuID: caseMenu.ID},
		{Name: "reopen", Desc: "重开已关闭事项", SysBaseMenuID: caseMenu.ID},
	}
	for i := range workbenchButtons {
		if err = tx.Where("name = ? AND sys_base_menu_id = ?", workbenchButtons[i].Name, workbenchMenu.ID).
			Attrs(workbenchButtons[i]).FirstOrCreate(&workbenchButtons[i]).Error; err != nil {
			return
		}
	}
	for i := range caseButtons {
		if err = tx.Where("name = ? AND sys_base_menu_id = ?", caseButtons[i].Name, caseMenu.ID).
			Attrs(caseButtons[i]).FirstOrCreate(&caseButtons[i]).Error; err != nil {
			return
		}
	}
	return
}

func ensureCaseWorkMenu(tx *gorm.DB, menu *system.SysBaseMenu) error {
	defaults := *menu
	if err := tx.Where("name = ?", menu.Name).Attrs(defaults).FirstOrCreate(menu).Error; err != nil {
		return err
	}
	return tx.Model(menu).Updates(map[string]any{
		"parent_id": defaults.ParentId,
		"path":      defaults.Path,
		"component": defaults.Component,
		"sort":      defaults.Sort,
		"title":     defaults.Meta.Title,
		"icon":      defaults.Meta.Icon,
	}).Error
}

func ensureAuthorityButton(tx *gorm.DB, role, menuID, buttonID uint) error {
	link := system.SysAuthorityBtn{AuthorityId: role, SysMenuID: menuID, SysBaseMenuBtnID: buttonID}
	return tx.Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", role, menuID, buttonID).
		FirstOrCreate(&link).Error
}
