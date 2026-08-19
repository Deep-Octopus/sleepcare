package initialize

import (
	"context"
	"fmt"
	"strconv"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

type phaseOneRoleSetting struct {
	AuthorityID  uint
	DataScope    int
	DefaultRoute string
}

type phaseOneMenuSpec struct {
	Name       string
	ParentName string
	Path       string
	Component  string
	Title      string
	Icon       string
	Sort       int
	Hidden     bool
	ActiveName string
}

type phaseOneButtonSpec struct {
	Name string
	Desc string
}

type phaseOneButtonGrant struct {
	MenuName   string
	ButtonName string
}

type phaseOneAPIGrant struct {
	Method string
	Path   string
}

var phaseOneRoleSettings = []phaseOneRoleSetting{
	{AuthorityID: syntheticStewardRole, DataScope: datascope.ScopeDept, DefaultRoute: "CareWorkbench"},
	{AuthorityID: syntheticClinicianRole, DataScope: datascope.ScopeDept, DefaultRoute: "CareWorkbench"},
	{AuthorityID: syntheticSupervisorRole, DataScope: datascope.ScopeDeptAndChild, DefaultRoute: "CareDailySummaries"},
	{AuthorityID: phaseOneContentAdminRole, DataScope: datascope.ScopeSelf, DefaultRoute: "CareQuestionnaires"},
}

var phaseOneMenuSpecs = []phaseOneMenuSpec{
	{Name: "SleepCare", Path: "sleep-care", Component: "view/routerHolder.vue", Title: "睡眠康养随访", Icon: "customer-gva", Sort: 20},
	{Name: "CareWorkbench", ParentName: "SleepCare", Path: "care-workbench", Component: "view/sleep-care/workbench/index.vue", Title: "工作台", Icon: "monitor-gva", Sort: 1},
	{Name: "CareClients", ParentName: "SleepCare", Path: "care-clients", Component: "view/sleep-care/clients/index.vue", Title: "康养用户", Icon: "customer-gva", Sort: 2},
	{Name: "CareExecution", ParentName: "SleepCare", Path: "care-execution", Component: "view/routerHolder.vue", Title: "执行管理", Icon: "config-gva", Sort: 3},
	{Name: "CareTasks", ParentName: "CareExecution", Path: "care-tasks", Component: "view/sleep-care/tasks/index.vue", Title: "今日任务", Icon: "config-file-gva", Sort: 1},
	{Name: "CareAttentionCases", ParentName: "CareExecution", Path: "attention-cases", Component: "view/sleep-care/attention-cases/index.vue", Title: "关注事项", Icon: "error-gva", Sort: 2},
	{Name: "CareDeliveries", ParentName: "CareExecution", Path: "care-deliveries", Component: "view/sleep-care/deliveries/index.vue", Title: "通知状态", Icon: "version-gva", Sort: 3},
	{Name: "CareConsultations", ParentName: "CareExecution", Path: "consultations", Component: "view/sleep-care/consultations/index.vue", Title: "主动咨询", Icon: "message", Sort: 4},
	{Name: "CareContent", ParentName: "SleepCare", Path: "care-content", Component: "view/routerHolder.vue", Title: "内容管理", Icon: "file-code-2-gva", Sort: 4},
	{Name: "CareQuestionnaires", ParentName: "CareContent", Path: "questionnaire-versions", Component: "view/sleep-care/questionnaires/index.vue", Title: "问卷内容", Icon: "version-gva", Sort: 1},
	{Name: "CarePlans", ParentName: "CareContent", Path: "plan-versions", Component: "view/sleep-care/plans/index.vue", Title: "服务计划", Icon: "config-file-gva", Sort: 2},
	{Name: "CareSupervision", ParentName: "SleepCare", Path: "care-supervision", Component: "view/routerHolder.vue", Title: "督导中心", Icon: "monitor-gva", Sort: 5},
	{Name: "CareDailySummaries", ParentName: "CareSupervision", Path: "daily-summaries", Component: "view/sleep-care/daily-summaries/index.vue", Title: "每日汇总", Icon: "version-gva", Sort: 1},
	{Name: "CareReviewQueue", ParentName: "CareSupervision", Path: "review-queue", Component: "view/sleep-care/review-queue/index.vue", Title: "待复核事项", Icon: "error-gva", Sort: 2},
	{Name: "CareSatisfaction", ParentName: "CareSupervision", Path: "satisfaction", Component: "view/sleep-care/satisfaction/index.vue", Title: "服务评价", Icon: "star", Sort: 3},
	{Name: "CareClientDetail", ParentName: "SleepCare", Path: "care-clients/:id", Component: "view/sleep-care/clients/detail.vue", Title: "康养用户详情", Sort: 90, Hidden: true, ActiveName: "CareClients"},
	{Name: "CareTaskDetail", ParentName: "CareExecution", Path: "care-tasks/:id", Component: "view/sleep-care/tasks/detail.vue", Title: "任务详情", Sort: 90, Hidden: true, ActiveName: "CareTasks"},
	{Name: "CareAttentionCaseDetail", ParentName: "CareExecution", Path: "attention-cases/:id", Component: "view/sleep-care/attention-cases/detail.vue", Title: "关注事项详情", Sort: 91, Hidden: true, ActiveName: "CareAttentionCases"},
	{Name: "CareConsultationDetail", ParentName: "CareExecution", Path: "consultations/:id", Component: "view/sleep-care/consultations/detail.vue", Title: "咨询详情", Sort: 92, Hidden: true, ActiveName: "CareConsultations"},
	{Name: "CareReviewDetail", ParentName: "CareSupervision", Path: "review-queue/:id", Component: "view/sleep-care/review-queue/detail.vue", Title: "个案复核", Sort: 90, Hidden: true, ActiveName: "CareReviewQueue"},
}

var phaseOneHiddenButtonSpecs = map[string][]phaseOneButtonSpec{
	"CareClientDetail": {
		{Name: "viewDetail", Desc: "查看详情"},
		{Name: "createClient", Desc: "新建康养用户"},
		{Name: "maintainClient", Desc: "维护公开资料"},
		{Name: "assignCare", Desc: "记录责任关系"},
		{Name: "recordConsent", Desc: "记录参与状态"},
		{Name: "recordLifecycleRequest", Desc: "记录生命周期请求"},
		{Name: "startPlan", Desc: "预览并启动服务计划"},
		{Name: "pausePlan", Desc: "暂停服务计划"},
		{Name: "resumePlan", Desc: "恢复服务计划"},
	},
	"CareTaskDetail": {
		{Name: "viewDetail", Desc: "查看计划任务详情"},
		{Name: "recordContact", Desc: "追加人工联系记录"},
	},
	"CareAttentionCaseDetail": {
		{Name: "viewDetail", Desc: "查看关注事项详情"},
		{Name: "acknowledge", Desc: "确认关注事项"},
		{Name: "recordContact", Desc: "记录非专业联系"},
		{Name: "recordHandling", Desc: "记录专业处置或请求复核"},
		{Name: "escalate", Desc: "升级关注事项"},
		{Name: "close", Desc: "关闭已解决事项"},
		{Name: "reopen", Desc: "重开已关闭事项"},
	},
	"CareConsultationDetail": {
		{Name: "viewDetail", Desc: "查看咨询详情"},
		{Name: "assign", Desc: "分配待分配咨询"},
		{Name: "reply", Desc: "公开回复咨询"},
		{Name: "transfer", Desc: "转交咨询"},
		{Name: "escalate", Desc: "升级咨询"},
		{Name: "resolve", Desc: "记录咨询解决结果"},
		{Name: "close", Desc: "关闭已解决咨询"},
		{Name: "reopen", Desc: "重开已关闭咨询"},
	},
	"CareReviewDetail": {
		{Name: "viewDetail", Desc: "查看待复核事项详情"},
		{Name: "guide", Desc: "追加上级指导"},
		{Name: "discuss", Desc: "要求流程讨论"},
		{Name: "intervene", Desc: "记录上级直接介入"},
	},
}

var phaseOneMenuAccess = map[uint][]string{
	syntheticStewardRole: {
		"SleepCare", "CareWorkbench", "CareClients", "CareExecution", "CareTasks", "CareAttentionCases",
		"CareDeliveries", "CareConsultations", "CareContent", "CarePlans", "CareClientDetail", "CareTaskDetail",
		"CareAttentionCaseDetail", "CareConsultationDetail",
	},
	syntheticClinicianRole: {
		"SleepCare", "CareWorkbench", "CareClients", "CareExecution", "CareTasks", "CareAttentionCases",
		"CareDeliveries", "CareConsultations", "CareContent", "CareQuestionnaires", "CarePlans", "CareClientDetail",
		"CareTaskDetail", "CareAttentionCaseDetail", "CareConsultationDetail",
	},
	syntheticSupervisorRole: {
		"SleepCare", "CareWorkbench", "CareClients", "CareExecution", "CareTasks", "CareAttentionCases", "CareDeliveries",
		"CareContent", "CareQuestionnaires", "CarePlans", "CareSupervision", "CareDailySummaries", "CareReviewQueue",
		"CareConsultations", "CareSatisfaction", "CareClientDetail", "CareTaskDetail", "CareAttentionCaseDetail",
		"CareConsultationDetail", "CareReviewDetail",
	},
	phaseOneContentAdminRole: {"SleepCare", "CareContent", "CareQuestionnaires", "CarePlans"},
}

var phaseOneButtonAccess = map[uint][]phaseOneButtonGrant{
	syntheticStewardRole: {
		{MenuName: "CareWorkbench", ButtonName: "viewDetail"},
		{MenuName: "CareClients", ButtonName: "viewDetail"},
		{MenuName: "CareClients", ButtonName: "startPlan"},
		{MenuName: "CareClients", ButtonName: "pausePlan"},
		{MenuName: "CareClients", ButtonName: "resumePlan"},
		{MenuName: "CarePlans", ButtonName: "preview"},
		{MenuName: "CareTasks", ButtonName: "viewDetail"},
		{MenuName: "CareTasks", ButtonName: "recordContact"},
		{MenuName: "CareAttentionCases", ButtonName: "viewDetail"},
		{MenuName: "CareAttentionCases", ButtonName: "acknowledge"},
		{MenuName: "CareAttentionCases", ButtonName: "recordContact"},
		{MenuName: "CareAttentionCases", ButtonName: "escalate"},
		{MenuName: "CareDeliveries", ButtonName: "resendNotice"},
		{MenuName: "CareConsultations", ButtonName: "viewDetail"},
		{MenuName: "CareConsultations", ButtonName: "reply"},
		{MenuName: "CareConsultations", ButtonName: "transfer"},
		{MenuName: "CareConsultations", ButtonName: "escalate"},
		{MenuName: "CareConsultations", ButtonName: "resolve"},
		{MenuName: "CareConsultations", ButtonName: "close"},
	},
	syntheticClinicianRole: {
		{MenuName: "CareWorkbench", ButtonName: "viewDetail"},
		{MenuName: "CareClients", ButtonName: "viewDetail"},
		{MenuName: "CareClients", ButtonName: "startPlan"},
		{MenuName: "CareClients", ButtonName: "pausePlan"},
		{MenuName: "CareClients", ButtonName: "resumePlan"},
		{MenuName: "CareQuestionnaires", ButtonName: "preview"},
		{MenuName: "CarePlans", ButtonName: "preview"},
		{MenuName: "CareTasks", ButtonName: "viewDetail"},
		{MenuName: "CareTasks", ButtonName: "recordContact"},
		{MenuName: "CareAttentionCases", ButtonName: "viewDetail"},
		{MenuName: "CareAttentionCases", ButtonName: "acknowledge"},
		{MenuName: "CareAttentionCases", ButtonName: "recordHandling"},
		{MenuName: "CareAttentionCases", ButtonName: "escalate"},
		{MenuName: "CareConsultations", ButtonName: "viewDetail"},
		{MenuName: "CareConsultations", ButtonName: "reply"},
		{MenuName: "CareConsultations", ButtonName: "transfer"},
		{MenuName: "CareConsultations", ButtonName: "escalate"},
		{MenuName: "CareConsultations", ButtonName: "resolve"},
		{MenuName: "CareConsultations", ButtonName: "close"},
	},
	syntheticSupervisorRole: {
		{MenuName: "CareWorkbench", ButtonName: "viewDetail"},
		{MenuName: "CareClients", ButtonName: "viewDetail"},
		{MenuName: "CareClients", ButtonName: "createClient"},
		{MenuName: "CareClients", ButtonName: "maintainClient"},
		{MenuName: "CareClients", ButtonName: "assignCare"},
		{MenuName: "CareClients", ButtonName: "recordConsent"},
		{MenuName: "CareClients", ButtonName: "recordLifecycleRequest"},
		{MenuName: "CareQuestionnaires", ButtonName: "preview"},
		{MenuName: "CarePlans", ButtonName: "preview"},
		{MenuName: "CareTasks", ButtonName: "viewDetail"},
		{MenuName: "CareAttentionCases", ButtonName: "viewDetail"},
		{MenuName: "CareAttentionCases", ButtonName: "close"},
		{MenuName: "CareAttentionCases", ButtonName: "reopen"},
		{MenuName: "CareConsultations", ButtonName: "viewDetail"},
		{MenuName: "CareConsultations", ButtonName: "assign"},
		{MenuName: "CareConsultations", ButtonName: "reply"},
		{MenuName: "CareConsultations", ButtonName: "transfer"},
		{MenuName: "CareConsultations", ButtonName: "resolve"},
		{MenuName: "CareConsultations", ButtonName: "close"},
		{MenuName: "CareConsultations", ButtonName: "reopen"},
		{MenuName: "CareDailySummaries", ButtonName: "viewDetail"},
		{MenuName: "CareDailySummaries", ButtonName: "revise"},
		{MenuName: "CareReviewQueue", ButtonName: "viewDetail"},
		{MenuName: "CareReviewQueue", ButtonName: "guide"},
		{MenuName: "CareReviewQueue", ButtonName: "discuss"},
		{MenuName: "CareReviewQueue", ButtonName: "intervene"},
		{MenuName: "CareSatisfaction", ButtonName: "viewFollowUp"},
		{MenuName: "CareSatisfaction", ButtonName: "acknowledgeFollowUp"},
		{MenuName: "CareSatisfaction", ButtonName: "resolveFollowUp"},
	},
	phaseOneContentAdminRole: {
		{MenuName: "CareQuestionnaires", ButtonName: "preview"},
		{MenuName: "CarePlans", ButtonName: "preview"},
	},
}

var phaseOneAPIAccess = map[uint][]phaseOneAPIGrant{
	syntheticStewardRole: {
		{Method: "GET", Path: "/care/clients"},
		{Method: "GET", Path: "/care/clients/:id"},
		{Method: "GET", Path: "/care/plan-versions"},
		{Method: "GET", Path: "/care/plan-versions/:id"},
		{Method: "POST", Path: "/care/clients/:id/plan-previews"},
		{Method: "POST", Path: "/care/clients/:id/plan-instances"},
		{Method: "GET", Path: "/care/clients/:id/plan-instances"},
		{Method: "POST", Path: "/care/plan-instances/:id/pause"},
		{Method: "POST", Path: "/care/plan-instances/:id/resume"},
		{Method: "GET", Path: "/care/tasks"},
		{Method: "GET", Path: "/care/tasks/:id"},
		{Method: "POST", Path: "/care/tasks/:id/contact-records"},
		{Method: "GET", Path: "/care/workbench"},
		{Method: "GET", Path: "/care/attention-cases"},
		{Method: "GET", Path: "/care/attention-cases/:id"},
		{Method: "POST", Path: "/care/attention-cases/:id/acknowledge"},
		{Method: "POST", Path: "/care/attention-cases/:id/handling-records"},
		{Method: "POST", Path: "/care/attention-cases/:id/escalate"},
		{Method: "GET", Path: "/care/deliveries"},
		{Method: "GET", Path: "/care/notification-provider-readiness"},
		{Method: "GET", Path: "/care/ai-shadow-readiness"},
		{Method: "POST", Path: "/care/deliveries/:id/resend"},
		{Method: "GET", Path: "/care/consultations"},
		{Method: "GET", Path: "/care/consultations/:id"},
		{Method: "GET", Path: "/care/consultations/:id/assignee-options"},
		{Method: "POST", Path: "/care/consultations/:id/replies"},
		{Method: "POST", Path: "/care/consultations/:id/transfer"},
		{Method: "POST", Path: "/care/consultations/:id/escalate"},
		{Method: "POST", Path: "/care/consultations/:id/resolve"},
		{Method: "POST", Path: "/care/consultations/:id/close"},
	},
	syntheticClinicianRole: {
		{Method: "GET", Path: "/care/clients"},
		{Method: "GET", Path: "/care/clients/:id"},
		{Method: "GET", Path: "/care/questionnaire-versions"},
		{Method: "GET", Path: "/care/questionnaire-versions/:id"},
		{Method: "GET", Path: "/care/plan-versions"},
		{Method: "GET", Path: "/care/plan-versions/:id"},
		{Method: "POST", Path: "/care/clients/:id/plan-previews"},
		{Method: "POST", Path: "/care/clients/:id/plan-instances"},
		{Method: "GET", Path: "/care/clients/:id/plan-instances"},
		{Method: "POST", Path: "/care/plan-instances/:id/pause"},
		{Method: "POST", Path: "/care/plan-instances/:id/resume"},
		{Method: "GET", Path: "/care/tasks"},
		{Method: "GET", Path: "/care/tasks/:id"},
		{Method: "POST", Path: "/care/tasks/:id/contact-records"},
		{Method: "GET", Path: "/care/workbench"},
		{Method: "GET", Path: "/care/attention-cases"},
		{Method: "GET", Path: "/care/attention-cases/:id"},
		{Method: "POST", Path: "/care/attention-cases/:id/acknowledge"},
		{Method: "POST", Path: "/care/attention-cases/:id/handling-records"},
		{Method: "POST", Path: "/care/attention-cases/:id/escalate"},
		{Method: "POST", Path: "/care/attention-cases/:id/close"},
		{Method: "GET", Path: "/care/deliveries"},
		{Method: "GET", Path: "/care/notification-provider-readiness"},
		{Method: "GET", Path: "/care/ai-shadow-readiness"},
		{Method: "GET", Path: "/care/consultations"},
		{Method: "GET", Path: "/care/consultations/:id"},
		{Method: "GET", Path: "/care/consultations/:id/assignee-options"},
		{Method: "POST", Path: "/care/consultations/:id/replies"},
		{Method: "POST", Path: "/care/consultations/:id/transfer"},
		{Method: "POST", Path: "/care/consultations/:id/escalate"},
		{Method: "POST", Path: "/care/consultations/:id/resolve"},
		{Method: "POST", Path: "/care/consultations/:id/close"},
	},
	syntheticSupervisorRole: {
		{Method: "GET", Path: "/care/clients"},
		{Method: "GET", Path: "/care/clients/:id"},
		{Method: "GET", Path: "/care/client-options"},
		{Method: "GET", Path: "/care/data-governance-readiness"},
		{Method: "GET", Path: "/care/clients/:id/data-lifecycle-requests"},
		{Method: "POST", Path: "/care/clients"},
		{Method: "PUT", Path: "/care/clients/:id"},
		{Method: "POST", Path: "/care/clients/:id/assignments"},
		{Method: "POST", Path: "/care/clients/:id/consent-records"},
		{Method: "POST", Path: "/care/clients/:id/data-lifecycle-requests"},
		{Method: "GET", Path: "/care/questionnaire-versions"},
		{Method: "GET", Path: "/care/questionnaire-versions/:id"},
		{Method: "GET", Path: "/care/plan-versions"},
		{Method: "GET", Path: "/care/plan-versions/:id"},
		{Method: "GET", Path: "/care/clients/:id/plan-instances"},
		{Method: "GET", Path: "/care/tasks"},
		{Method: "GET", Path: "/care/tasks/:id"},
		{Method: "GET", Path: "/care/workbench"},
		{Method: "GET", Path: "/care/attention-cases"},
		{Method: "GET", Path: "/care/attention-cases/:id"},
		{Method: "POST", Path: "/care/attention-cases/:id/close"},
		{Method: "POST", Path: "/care/attention-cases/:id/reopen"},
		{Method: "GET", Path: "/care/deliveries"},
		{Method: "GET", Path: "/care/notification-provider-readiness"},
		{Method: "GET", Path: "/care/ai-shadow-readiness"},
		{Method: "GET", Path: "/care/operations-dashboard"},
		{Method: "GET", Path: "/care/daily-summaries"},
		{Method: "GET", Path: "/care/daily-summaries/:id"},
		{Method: "POST", Path: "/care/daily-summaries/:id/revisions"},
		{Method: "GET", Path: "/care/reviews"},
		{Method: "POST", Path: "/care/reviews/:id/guidance"},
		{Method: "POST", Path: "/care/reviews/:id/intervene"},
		{Method: "GET", Path: "/care/consultations"},
		{Method: "GET", Path: "/care/consultations/:id"},
		{Method: "GET", Path: "/care/consultations/:id/assignee-options"},
		{Method: "POST", Path: "/care/consultations/:id/assign"},
		{Method: "POST", Path: "/care/consultations/:id/replies"},
		{Method: "POST", Path: "/care/consultations/:id/transfer"},
		{Method: "POST", Path: "/care/consultations/:id/resolve"},
		{Method: "POST", Path: "/care/consultations/:id/close"},
		{Method: "POST", Path: "/care/consultations/:id/reopen"},
		{Method: "GET", Path: "/care/satisfaction-responses"},
		{Method: "GET", Path: "/care/satisfaction-follow-ups"},
		{Method: "GET", Path: "/care/satisfaction-follow-ups/:id"},
		{Method: "POST", Path: "/care/satisfaction-follow-ups/:id/acknowledge"},
		{Method: "POST", Path: "/care/satisfaction-follow-ups/:id/resolve"},
	},
	phaseOneContentAdminRole: {
		{Method: "GET", Path: "/care/questionnaire-versions"},
		{Method: "GET", Path: "/care/questionnaire-versions/:id"},
		{Method: "GET", Path: "/care/plan-versions"},
		{Method: "GET", Path: "/care/plan-versions/:id"},
	},
}

var phaseOneShellAccess = []phaseOneAPIGrant{
	{Method: "POST", Path: "/menu/getMenu"},
	{Method: "GET", Path: "/user/getUserInfo"},
	{Method: "POST", Path: "/jwt/jsonInBlacklist"},
}

// EnsurePhaseOneAccessControl reconciles the phase-one employee menu and
// permission matrix after every contributing module has registered metadata.
func EnsurePhaseOneAccessControl() error {
	if global.GVA_DB == nil {
		return nil
	}
	db := global.GVA_DB.WithContext(datascope.WithSystem(context.Background()))
	return ensurePhaseOneAccessControl(db, global.GVA_CONFIG.Care.SyntheticFixturesEnabled)
}

func ensurePhaseOneAccessControl(db *gorm.DB, grantFixtures bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		menus, err := ensurePhaseOneMenuTree(tx)
		if err != nil {
			return err
		}
		if err = ensurePhaseOneHiddenButtons(tx, menus); err != nil {
			return err
		}
		if !grantFixtures {
			return nil
		}
		if err = reconcilePhaseOneRoleSettings(tx); err != nil {
			return err
		}
		if err = reconcilePhaseOneMenus(tx, menus); err != nil {
			return err
		}
		if err = reconcilePhaseOneButtons(tx, menus); err != nil {
			return err
		}
		return reconcilePhaseOnePolicies(tx)
	})
}

func ensurePhaseOneMenuTree(tx *gorm.DB) (map[string]system.SysBaseMenu, error) {
	menus := make(map[string]system.SysBaseMenu, len(phaseOneMenuSpecs))
	for _, spec := range phaseOneMenuSpecs {
		parentID := uint(0)
		menuLevel := uint(0)
		if spec.ParentName != "" {
			parent, ok := menus[spec.ParentName]
			if !ok {
				return nil, fmt.Errorf("phase-one menu parent %s missing", spec.ParentName)
			}
			parentID = parent.ID
			menuLevel = parent.MenuLevel + 1
		}
		menu := system.SysBaseMenu{
			MenuLevel: menuLevel, ParentId: parentID, Path: spec.Path, Name: spec.Name,
			Hidden: spec.Hidden, Component: spec.Component, Sort: spec.Sort,
			Meta: system.Meta{Title: spec.Title, Icon: spec.Icon, ActiveName: spec.ActiveName},
		}
		if err := tx.Where("name = ?", spec.Name).Attrs(menu).FirstOrCreate(&menu).Error; err != nil {
			return nil, fmt.Errorf("ensure phase-one menu %s: %w", spec.Name, err)
		}
		if err := tx.Model(&menu).Updates(map[string]any{
			"menu_level":      menuLevel,
			"parent_id":       parentID,
			"path":            spec.Path,
			"hidden":          spec.Hidden,
			"component":       spec.Component,
			"sort":            spec.Sort,
			"active_name":     spec.ActiveName,
			"keep_alive":      false,
			"default_menu":    false,
			"title":           spec.Title,
			"icon":            spec.Icon,
			"close_tab":       false,
			"transition_type": "",
		}).Error; err != nil {
			return nil, fmt.Errorf("update phase-one menu %s: %w", spec.Name, err)
		}
		menu.MenuLevel = menuLevel
		menu.ParentId = parentID
		menus[spec.Name] = menu
	}
	return menus, nil
}

func ensurePhaseOneHiddenButtons(tx *gorm.DB, menus map[string]system.SysBaseMenu) error {
	for menuName, specs := range phaseOneHiddenButtonSpecs {
		menu := menus[menuName]
		for _, spec := range specs {
			button := system.SysBaseMenuBtn{Name: spec.Name, Desc: spec.Desc, SysBaseMenuID: menu.ID}
			if err := tx.Where("name = ? AND sys_base_menu_id = ?", spec.Name, menu.ID).
				Attrs(button).FirstOrCreate(&button).Error; err != nil {
				return fmt.Errorf("ensure %s button %s: %w", menuName, spec.Name, err)
			}
			if err := tx.Model(&button).Update("desc", spec.Desc).Error; err != nil {
				return fmt.Errorf("update %s button %s: %w", menuName, spec.Name, err)
			}
		}
	}
	return nil
}

func reconcilePhaseOneRoleSettings(tx *gorm.DB) error {
	for _, setting := range phaseOneRoleSettings {
		var authority system.SysAuthority
		if err := tx.Where("authority_id = ?", setting.AuthorityID).First(&authority).Error; err != nil {
			return fmt.Errorf("phase-one authority %d missing: %w", setting.AuthorityID, err)
		}
		if err := tx.Model(&system.SysAuthority{}).
			Where("authority_id = ?", setting.AuthorityID).
			Updates(map[string]any{"data_scope": setting.DataScope, "default_router": setting.DefaultRoute}).Error; err != nil {
			return err
		}
	}
	return nil
}

func reconcilePhaseOneMenus(tx *gorm.DB, menus map[string]system.SysBaseMenu) error {
	roleIDs, roleStrings := phaseOneRoleIdentifiers()
	if err := tx.Where("sys_authority_authority_id IN ?", roleStrings).Delete(&system.SysAuthorityMenu{}).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		for _, menuName := range phaseOneMenuAccess[roleID] {
			menu, ok := menus[menuName]
			if !ok {
				return fmt.Errorf("phase-one menu grant target %s missing", menuName)
			}
			link := system.SysAuthorityMenu{MenuId: strconv.FormatUint(uint64(menu.ID), 10), AuthorityId: strconv.FormatUint(uint64(roleID), 10)}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
	}
	menuIDs := make([]string, 0, len(menus))
	for _, menu := range menus {
		menuIDs = append(menuIDs, strconv.FormatUint(uint64(menu.ID), 10))
	}
	return tx.Where("sys_authority_authority_id = ? AND sys_base_menu_id IN ?", "888", menuIDs).
		Delete(&system.SysAuthorityMenu{}).Error
}

func reconcilePhaseOneButtons(tx *gorm.DB, menus map[string]system.SysBaseMenu) error {
	roleIDs, _ := phaseOneRoleIdentifiers()
	if err := tx.Where("authority_id IN ?", roleIDs).Delete(&system.SysAuthorityBtn{}).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		grants := append([]phaseOneButtonGrant(nil), phaseOneButtonAccess[roleID]...)
		grants = append(grants, phaseOneHiddenButtonGrants(roleID)...)
		for _, grant := range grants {
			menu, ok := menus[grant.MenuName]
			if !ok {
				return fmt.Errorf("phase-one button menu %s missing", grant.MenuName)
			}
			var button system.SysBaseMenuBtn
			if err := tx.Where("sys_base_menu_id = ? AND name = ?", menu.ID, grant.ButtonName).First(&button).Error; err != nil {
				return fmt.Errorf("phase-one button %s.%s missing: %w", grant.MenuName, grant.ButtonName, err)
			}
			link := system.SysAuthorityBtn{AuthorityId: roleID, SysMenuID: menu.ID, SysBaseMenuBtnID: button.ID}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
	}
	menuIDs := make([]uint, 0, len(menus))
	for _, menu := range menus {
		menuIDs = append(menuIDs, menu.ID)
	}
	return tx.Where("authority_id = ? AND sys_menu_id IN ?", uint(888), menuIDs).
		Delete(&system.SysAuthorityBtn{}).Error
}

func phaseOneHiddenButtonGrants(roleID uint) []phaseOneButtonGrant {
	visibleToHidden := map[string]string{
		"CareClients":        "CareClientDetail",
		"CareTasks":          "CareTaskDetail",
		"CareAttentionCases": "CareAttentionCaseDetail",
		"CareReviewQueue":    "CareReviewDetail",
		"CareConsultations":  "CareConsultationDetail",
	}
	grants := make([]phaseOneButtonGrant, 0)
	for _, grant := range phaseOneButtonAccess[roleID] {
		if hiddenMenu, ok := visibleToHidden[grant.MenuName]; ok {
			grants = append(grants, phaseOneButtonGrant{MenuName: hiddenMenu, ButtonName: grant.ButtonName})
		}
	}
	return grants
}

func reconcilePhaseOnePolicies(tx *gorm.DB) error {
	roleIDs, roleStrings := phaseOneRoleIdentifiers()
	if err := tx.Where("ptype = ? AND v0 IN ?", "p", roleStrings).Delete(&adapter.CasbinRule{}).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		grants := append([]phaseOneAPIGrant(nil), phaseOneShellAccess...)
		grants = append(grants, phaseOneAPIAccess[roleID]...)
		for _, grant := range grants {
			policy := adapter.CasbinRule{
				Ptype: "p", V0: strconv.FormatUint(uint64(roleID), 10), V1: grant.Path, V2: grant.Method,
			}
			if err := tx.Create(&policy).Error; err != nil {
				return err
			}
		}
	}
	return tx.Where("ptype = ? AND v0 = ? AND v1 LIKE ?", "p", "888", "/care/%").
		Delete(&adapter.CasbinRule{}).Error
}

func phaseOneRoleIdentifiers() ([]uint, []string) {
	roleIDs := make([]uint, 0, len(phaseOneRoleSettings))
	roleStrings := make([]string, 0, len(phaseOneRoleSettings))
	for _, setting := range phaseOneRoleSettings {
		roleIDs = append(roleIDs, setting.AuthorityID)
		roleStrings = append(roleStrings, strconv.FormatUint(uint64(setting.AuthorityID), 10))
	}
	return roleIDs, roleStrings
}
