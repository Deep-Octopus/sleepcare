package initialize

import (
	"fmt"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
)

var consultationAPIs = []system.SysApi{
	{ApiGroup: "主动咨询", Method: "GET", Path: "/care/consultations", Description: "获取授权范围咨询列表"},
	{ApiGroup: "主动咨询", Method: "GET", Path: "/care/consultations/:id", Description: "获取授权范围咨询详情"},
	{ApiGroup: "主动咨询", Method: "GET", Path: "/care/consultations/:id/assignee-options", Description: "获取咨询可用责任人员"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/assign", Description: "分配待分配咨询"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/replies", Description: "公开回复咨询"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/transfer", Description: "转交咨询"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/escalate", Description: "升级咨询"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/resolve", Description: "记录咨询解决结果"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/close", Description: "关闭已解决咨询"},
	{ApiGroup: "主动咨询", Method: "POST", Path: "/care/consultations/:id/reopen", Description: "重开已关闭咨询"},
}

func ensureConsultationMetadata(db *gorm.DB, grantPolicies bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range consultationAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).
				Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure consultation API %s: %w", api.Path, err)
			}
		}
		root, execution, listMenu, detailMenu, buttons, detailButtons, err := ensureConsultationMenus(tx)
		if err != nil {
			return err
		}
		if !grantPolicies {
			return nil
		}
		roles := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
		for _, role := range roles {
			for _, menuID := range []uint{root.ID, execution.ID, listMenu.ID, detailMenu.ID} {
				link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(role)}
				if err = tx.Where(link).FirstOrCreate(&link).Error; err != nil {
					return err
				}
			}
		}
		buttonRoles := consultationButtonRoles()
		for _, button := range append(buttons, detailButtons...) {
			for _, role := range buttonRoles[button.Name] {
				if err = ensureAuthorityButton(tx, role, button.SysBaseMenuID, button.ID); err != nil {
					return err
				}
			}
		}
		rolesByRoute := consultationRolesByRoute()
		for _, api := range consultationAPIs {
			for _, role := range rolesByRoute[api.Method+" "+api.Path] {
				policy := adapter.CasbinRule{
					Ptype: "p",
					V0:    fmt.Sprint(role),
					V1:    api.Path,
					V2:    api.Method,
				}
				if err = tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ensureConsultationMenus(tx *gorm.DB) (
	root system.SysBaseMenu,
	execution system.SysBaseMenu,
	listMenu system.SysBaseMenu,
	detailMenu system.SysBaseMenu,
	buttons []system.SysBaseMenuBtn,
	detailButtons []system.SysBaseMenuBtn,
	err error,
) {
	root = system.SysBaseMenu{
		Path:      "sleep-care",
		Name:      "SleepCare",
		Component: "view/routerHolder.vue",
		Sort:      20,
		Meta:      system.Meta{Title: "睡眠康养随访", Icon: "customer-gva"},
	}
	if err = tx.Where("name = ?", root.Name).Attrs(root).FirstOrCreate(&root).Error; err != nil {
		return
	}
	execution = system.SysBaseMenu{
		ParentId:  root.ID,
		Path:      "care-execution",
		Name:      "CareExecution",
		Component: "view/routerHolder.vue",
		Sort:      3,
		Meta:      system.Meta{Title: "执行管理", Icon: "config-gva"},
	}
	if err = ensureCaseWorkMenu(tx, &execution); err != nil {
		return
	}
	listMenu = system.SysBaseMenu{
		ParentId:  execution.ID,
		Path:      "consultations",
		Name:      "CareConsultations",
		Component: "view/sleep-care/consultations/index.vue",
		Sort:      4,
		Meta:      system.Meta{Title: "主动咨询", Icon: "message"},
	}
	if err = ensureCaseWorkMenu(tx, &listMenu); err != nil {
		return
	}
	detailMenu = system.SysBaseMenu{
		ParentId:  execution.ID,
		Path:      "consultations/:id",
		Name:      "CareConsultationDetail",
		Hidden:    true,
		Component: "view/sleep-care/consultations/detail.vue",
		Sort:      92,
		Meta:      system.Meta{Title: "咨询详情", ActiveName: "CareConsultations"},
	}
	if err = ensureCaseWorkMenu(tx, &detailMenu); err != nil {
		return
	}
	buttonSpecs := []phaseOneButtonSpec{
		{Name: "viewDetail", Desc: "查看咨询详情"},
		{Name: "assign", Desc: "分配待分配咨询"},
		{Name: "reply", Desc: "公开回复咨询"},
		{Name: "transfer", Desc: "转交咨询"},
		{Name: "escalate", Desc: "升级咨询"},
		{Name: "resolve", Desc: "记录咨询解决结果"},
		{Name: "close", Desc: "关闭已解决咨询"},
		{Name: "reopen", Desc: "重开已关闭咨询"},
	}
	buttons, err = ensureConsultationButtons(tx, listMenu.ID, buttonSpecs)
	if err != nil {
		return
	}
	detailButtons, err = ensureConsultationButtons(tx, detailMenu.ID, buttonSpecs)
	return
}

func ensureConsultationButtons(
	tx *gorm.DB,
	menuID uint,
	specs []phaseOneButtonSpec,
) ([]system.SysBaseMenuBtn, error) {
	buttons := make([]system.SysBaseMenuBtn, 0, len(specs))
	for _, spec := range specs {
		button := system.SysBaseMenuBtn{Name: spec.Name, Desc: spec.Desc, SysBaseMenuID: menuID}
		if err := tx.Where("name = ? AND sys_base_menu_id = ?", spec.Name, menuID).
			Attrs(button).FirstOrCreate(&button).Error; err != nil {
			return nil, err
		}
		buttons = append(buttons, button)
	}
	return buttons, nil
}

func consultationButtonRoles() map[string][]uint {
	all := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
	return map[string][]uint{
		"viewDetail": all,
		"assign":     {syntheticSupervisorRole},
		"reply":      all,
		"transfer":   all,
		"escalate":   {syntheticStewardRole, syntheticClinicianRole},
		"resolve":    all,
		"close":      all,
		"reopen":     {syntheticSupervisorRole},
	}
}

func consultationRolesByRoute() map[string][]uint {
	all := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
	return map[string][]uint{
		"GET /care/consultations":                      all,
		"GET /care/consultations/:id":                  all,
		"GET /care/consultations/:id/assignee-options": all,
		"POST /care/consultations/:id/assign":          {syntheticSupervisorRole},
		"POST /care/consultations/:id/replies":         all,
		"POST /care/consultations/:id/transfer":        all,
		"POST /care/consultations/:id/escalate":        {syntheticStewardRole, syntheticClinicianRole},
		"POST /care/consultations/:id/resolve":         all,
		"POST /care/consultations/:id/close":           all,
		"POST /care/consultations/:id/reopen":          {syntheticSupervisorRole},
	}
}
