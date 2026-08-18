package initialize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	syntheticPathVersionID       = 9300
	syntheticPlanTemplateID      = 9301
	syntheticTaskDefinitionD1ID  = 9311
	syntheticEnrollmentID        = 9601
	syntheticPlanPreviewID       = 9602
	syntheticPlanInstanceID      = 9603
	syntheticTaskInstanceD1ID    = 9611
	syntheticPlanStartedEventID  = 9621
	syntheticTaskOpenedEventID   = 9622
	syntheticPlanPreviewOpaqueID = "syn-p1-04-osa-a001-preview"
)

var carePathAPIs = []system.SysApi{
	{ApiGroup: "康养计划", Method: "GET", Path: "/care/plan-versions", Description: "获取计划模板版本列表"},
	{ApiGroup: "康养计划", Method: "GET", Path: "/care/plan-versions/:id", Description: "获取计划模板版本详情"},
	{ApiGroup: "康养计划", Method: "POST", Path: "/care/clients/:id/plan-previews", Description: "预览测试计划"},
	{ApiGroup: "康养计划", Method: "POST", Path: "/care/clients/:id/plan-instances", Description: "幂等启动测试计划"},
	{ApiGroup: "康养计划", Method: "GET", Path: "/care/clients/:id/plan-instances", Description: "获取康养用户计划时间线"},
	{ApiGroup: "康养计划", Method: "POST", Path: "/care/plan-instances/:id/pause", Description: "暂停测试计划"},
	{ApiGroup: "康养计划", Method: "POST", Path: "/care/plan-instances/:id/resume", Description: "恢复测试计划"},
	{ApiGroup: "康养任务", Method: "GET", Path: "/care/tasks", Description: "获取责任范围任务列表"},
	{ApiGroup: "康养任务", Method: "GET", Path: "/care/tasks/:id", Description: "获取责任范围任务详情"},
}

func EnsureCarePathData() error {
	if global.GVA_DB == nil {
		return nil
	}
	db := global.GVA_DB.WithContext(datascope.WithSystem(context.Background()))
	if err := ensureCarePathMetadata(db); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	return ensureCarePathSyntheticFixtures(db)
}

func ensureCarePathMetadata(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range carePathAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure care path API %s: %w", api.Path, err)
			}
		}
		var root, clientMenu system.SysBaseMenu
		if err := tx.Where("name = ?", "SleepCare").First(&root).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", "CareClients").First(&clientMenu).Error; err != nil {
			return err
		}
		planMenu := system.SysBaseMenu{
			ParentId: root.ID, Path: "plan-versions", Name: "CarePlans",
			Component: "view/sleep-care/plans/index.vue", Sort: 3,
			Meta: system.Meta{Title: "OSA 计划", Icon: "version-gva"},
		}
		if err := tx.Where("name = ?", planMenu.Name).Attrs(planMenu).FirstOrCreate(&planMenu).Error; err != nil {
			return err
		}
		taskMenu := system.SysBaseMenu{
			ParentId: root.ID, Path: "care-tasks", Name: "CareTasks",
			Component: "view/sleep-care/tasks/index.vue", Sort: 4,
			Meta: system.Meta{Title: "计划任务", Icon: "config-file-gva"},
		}
		if err := tx.Where("name = ?", taskMenu.Name).Attrs(taskMenu).FirstOrCreate(&taskMenu).Error; err != nil {
			return err
		}
		buttons := []system.SysBaseMenuBtn{
			{Name: "preview", Desc: "预览计划模板版本", SysBaseMenuID: planMenu.ID},
			{Name: "viewDetail", Desc: "查看计划任务详情", SysBaseMenuID: taskMenu.ID},
			{Name: "startPlan", Desc: "预览并启动测试计划", SysBaseMenuID: clientMenu.ID},
			{Name: "pausePlan", Desc: "暂停测试计划", SysBaseMenuID: clientMenu.ID},
			{Name: "resumePlan", Desc: "恢复测试计划", SysBaseMenuID: clientMenu.ID},
		}
		for i := range buttons {
			if err := tx.Where("name = ? AND sys_base_menu_id = ?", buttons[i].Name, buttons[i].SysBaseMenuID).
				Attrs(buttons[i]).FirstOrCreate(&buttons[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureCarePathSyntheticFixtures(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := grantCarePathAccess(tx); err != nil {
			return err
		}
		definitions, err := syntheticPlanDefinitions()
		if err != nil {
			return err
		}
		if err = seedSyntheticPlanDefinitions(tx, definitions); err != nil {
			return err
		}
		return seedSyntheticPlanRuntime(tx, definitions)
	})
}

func grantCarePathAccess(tx *gorm.DB) error {
	var root, clientMenu, planMenu, taskMenu system.SysBaseMenu
	for name, target := range map[string]*system.SysBaseMenu{
		"SleepCare": &root, "CareClients": &clientMenu, "CarePlans": &planMenu, "CareTasks": &taskMenu,
	} {
		if err := tx.Where("name = ?", name).First(target).Error; err != nil {
			return err
		}
	}
	roles := []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole}
	for _, role := range roles {
		for _, menuID := range []uint{root.ID, planMenu.ID, taskMenu.ID} {
			link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(role)}
			if err := tx.Where(link).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
	}
	var planPreviewButton, taskDetailButton system.SysBaseMenuBtn
	if err := tx.Where("name = ? AND sys_base_menu_id = ?", "preview", planMenu.ID).First(&planPreviewButton).Error; err != nil {
		return err
	}
	if err := tx.Where("name = ? AND sys_base_menu_id = ?", "viewDetail", taskMenu.ID).First(&taskDetailButton).Error; err != nil {
		return err
	}
	for _, role := range roles {
		for _, pair := range []struct{ menuID, buttonID uint }{{planMenu.ID, planPreviewButton.ID}, {taskMenu.ID, taskDetailButton.ID}} {
			link := system.SysAuthorityBtn{AuthorityId: role, SysMenuID: pair.menuID, SysBaseMenuBtnID: pair.buttonID}
			if err := tx.Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", role, pair.menuID, pair.buttonID).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"startPlan", "pausePlan", "resumePlan"} {
		var button system.SysBaseMenuBtn
		if err := tx.Where("name = ? AND sys_base_menu_id = ?", name, clientMenu.ID).First(&button).Error; err != nil {
			return err
		}
		if err := tx.Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", syntheticSupervisorRole, clientMenu.ID, button.ID).
			Delete(&system.SysAuthorityBtn{}).Error; err != nil {
			return err
		}
		for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole} {
			link := system.SysAuthorityBtn{AuthorityId: role, SysMenuID: clientMenu.ID, SysBaseMenuBtnID: button.ID}
			if err := tx.Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", role, clientMenu.ID, button.ID).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
	}
	readPaths := map[string]bool{
		"GET /care/plan-versions": true, "GET /care/plan-versions/:id": true,
		"GET /care/clients/:id/plan-instances": true, "GET /care/tasks": true, "GET /care/tasks/:id": true,
	}
	for _, role := range roles {
		for _, api := range carePathAPIs {
			allowed := readPaths[api.Method+" "+api.Path] || role == syntheticStewardRole || role == syntheticClinicianRole
			policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
			if allowed {
				if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Where(policy).Delete(&adapter.CasbinRule{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

type syntheticPlanBundle struct {
	Path     pathmodel.PathDefinitionVersion
	Template pathmodel.PlanTemplateVersion
	Tasks    []pathmodel.PlanTaskDefinition
}

func syntheticPlanDefinitions() (syntheticPlanBundle, error) {
	fixed := carePathFixedTime()
	emptyRules := datatypes.JSON(mustJSON([]uint{}))
	d1Rules := datatypes.JSON(mustJSON([]uint{syntheticRuleVersionID}))
	questionnaireID := uint(syntheticQuestionnaireVersionID)
	tasks := make([]pathmodel.PlanTaskDefinition, 0, 5)
	for day := 1; day <= 5; day++ {
		task := pathmodel.PlanTaskDefinition{
			GVA_MODEL:             global.GVA_MODEL{ID: uint(syntheticTaskDefinitionD1ID + day - 1)},
			PlanTemplateVersionID: syntheticPlanTemplateID, DayCode: fmt.Sprintf("D%d", day),
			Title: fmt.Sprintf("D%d 测试计划节奏验证任务", day), Sort: day,
			ExecutionRole:           pathmodel.ExecutionRoleCareClient,
			OpenOffsetSeconds:       int64(day-1) * 24 * 60 * 60,
			DueOffsetSeconds:        int64(day-1)*24*60*60 + 11*60*60,
			BoundRuleVersionIDsJSON: append(datatypes.JSON(nil), emptyRules...),
			NotificationPolicy:      pathmodel.NotificationPolicyDisabled,
		}
		if day == 1 {
			task.Title = "D1 测试流程确认任务"
			task.QuestionnaireVersionID = &questionnaireID
			task.BoundRuleVersionIDsJSON = append(datatypes.JSON(nil), d1Rules...)
			task.ReviewRequired = true
			task.ReviewRole = pathmodel.ExecutionRoleClinician
		}
		tasks = append(tasks, task)
	}
	docTasks := make([]pathmodel.TaskDefinitionDocument, 0, len(tasks))
	for _, task := range tasks {
		var ids []uint
		if err := json.Unmarshal(task.BoundRuleVersionIDsJSON, &ids); err != nil {
			return syntheticPlanBundle{}, err
		}
		docTasks = append(docTasks, pathmodel.TaskDefinitionDocument{
			DayCode: task.DayCode, Title: task.Title, Sort: task.Sort, ExecutionRole: task.ExecutionRole,
			OpenOffsetSeconds: task.OpenOffsetSeconds, DueOffsetSeconds: task.DueOffsetSeconds,
			QuestionnaireVersionID: task.QuestionnaireVersionID, BoundRuleVersionIDs: ids,
			ReviewRequired: task.ReviewRequired, ReviewRole: task.ReviewRole,
			NotificationPolicy: task.NotificationPolicy,
		})
	}
	planDocument := pathmodel.PlanDefinitionDocument{
		PathCode: "OSA", Code: "SYN-OSA-D1-D5", Version: "1.0.0-synthetic",
		Title:      "测试 OSA 流程验证计划（非医疗内容）",
		Purpose:    "仅验证计划预览、幂等启动、D1–D5 时间窗和暂停恢复的软件行为。",
		UsageScope: pathmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		AnchorDefinition:     pathmodel.AnchorFirstValidSyntheticDeviceUse,
		LateSubmissionPolicy: pathmodel.LateSubmissionDeny, PauseStrategy: pathmodel.PauseStrategyKeepWindows,
		DefinitionSchemaVersion: "v1", Tasks: docTasks,
	}
	planHash, err := pathmodel.HashDefinition(planDocument)
	if err != nil {
		return syntheticPlanBundle{}, err
	}
	pathDefinition := pathmodel.PathDefinitionDocument{
		Code: "OSA", Version: "1.0.0-synthetic", Title: "测试 OSA 软件流程路径（非医疗内容）",
		Purpose: "仅用于阶段一软件验收。", UsageScope: pathmodel.UsageScopeTestOnly,
		Synthetic: true, ProductionEnabled: false,
	}
	pathHash, err := pathmodel.HashDefinition(pathDefinition)
	if err != nil {
		return syntheticPlanBundle{}, err
	}
	path := pathmodel.PathDefinitionVersion{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticPathVersionID}, Code: pathDefinition.Code,
		Version: pathDefinition.Version, Title: pathDefinition.Title, Purpose: pathDefinition.Purpose,
		Status: pathmodel.LifecyclePublished, UsageScope: pathmodel.UsageScopeTestOnly,
		Synthetic: true, ProductionEnabled: false, ReviewType: pathmodel.ReviewTypeEngineering,
		ReviewedBy: syntheticSupervisorAID, ReviewedAt: &fixed,
		ReviewNote: "测试软件流程工程复核；不包含医疗审批。", PublishedAt: &fixed,
		DefinitionHash: pathHash, RowVersion: 1,
	}
	template := pathmodel.PlanTemplateVersion{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticPlanTemplateID}, PathDefinitionVersionID: path.ID,
		Code: planDocument.Code, Version: planDocument.Version, Title: planDocument.Title, Purpose: planDocument.Purpose,
		Status: pathmodel.LifecyclePublished, UsageScope: planDocument.UsageScope,
		Synthetic: true, ProductionEnabled: false, ReviewType: pathmodel.ReviewTypeEngineering,
		ReviewedBy: syntheticSupervisorAID, ReviewedAt: &fixed,
		ReviewNote: "测试 D1–D5 调度工程复核；不包含医疗任务审批。", PublishedAt: &fixed,
		AnchorDefinition: planDocument.AnchorDefinition, LateSubmissionPolicy: planDocument.LateSubmissionPolicy,
		PauseStrategy: planDocument.PauseStrategy, DefinitionSchemaVersion: planDocument.DefinitionSchemaVersion,
		DefinitionHash: planHash, RowVersion: 1,
	}
	return syntheticPlanBundle{Path: path, Template: template, Tasks: tasks}, nil
}

func seedSyntheticPlanDefinitions(tx *gorm.DB, bundle syntheticPlanBundle) error {
	var existing pathmodel.PathDefinitionVersion
	err := tx.Unscoped().Where("id = ?", bundle.Path.ID).First(&existing).Error
	if err == nil {
		return verifySyntheticPlanDefinitions(tx, bundle)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var occupied int64
	if err = tx.Unscoped().Model(&pathmodel.PlanTemplateVersion{}).Where("id = ?", bundle.Template.ID).Count(&occupied).Error; err != nil {
		return err
	}
	if occupied != 0 {
		return fmt.Errorf("synthetic plan template id %d is occupied", bundle.Template.ID)
	}
	if err = tx.Create(&bundle.Path).Error; err != nil {
		return err
	}
	if err = tx.Create(&bundle.Template).Error; err != nil {
		return err
	}
	return tx.Create(&bundle.Tasks).Error
}

func verifySyntheticPlanDefinitions(tx *gorm.DB, bundle syntheticPlanBundle) error {
	var path pathmodel.PathDefinitionVersion
	if err := tx.Unscoped().Where("id = ?", bundle.Path.ID).First(&path).Error; err != nil {
		return err
	}
	if path.Code != bundle.Path.Code || path.Version != bundle.Path.Version || path.Title != bundle.Path.Title ||
		path.Purpose != bundle.Path.Purpose || path.Status != bundle.Path.Status || path.UsageScope != bundle.Path.UsageScope ||
		path.Synthetic != bundle.Path.Synthetic || path.ProductionEnabled != bundle.Path.ProductionEnabled ||
		path.ReviewType != bundle.Path.ReviewType || path.ReviewedBy != bundle.Path.ReviewedBy ||
		!sameTimePointer(path.ReviewedAt, bundle.Path.ReviewedAt) || path.ReviewNote != bundle.Path.ReviewNote ||
		!sameTimePointer(path.PublishedAt, bundle.Path.PublishedAt) || path.DefinitionHash != bundle.Path.DefinitionHash ||
		path.RowVersion != bundle.Path.RowVersion || path.DeletedAt.Valid {
		return fmt.Errorf("synthetic path version id %d is occupied or definition differs", bundle.Path.ID)
	}
	var template pathmodel.PlanTemplateVersion
	if err := tx.Unscoped().Where("id = ?", bundle.Template.ID).First(&template).Error; err != nil {
		return fmt.Errorf("synthetic plan template differs: %w", err)
	}
	if template.PathDefinitionVersionID != bundle.Template.PathDefinitionVersionID || template.Code != bundle.Template.Code ||
		template.Version != bundle.Template.Version || template.Title != bundle.Template.Title || template.Purpose != bundle.Template.Purpose ||
		template.Status != bundle.Template.Status || template.UsageScope != bundle.Template.UsageScope ||
		template.Synthetic != bundle.Template.Synthetic || template.ProductionEnabled != bundle.Template.ProductionEnabled ||
		template.ReviewType != bundle.Template.ReviewType || template.ReviewedBy != bundle.Template.ReviewedBy ||
		!sameTimePointer(template.ReviewedAt, bundle.Template.ReviewedAt) || template.ReviewNote != bundle.Template.ReviewNote ||
		!sameTimePointer(template.PublishedAt, bundle.Template.PublishedAt) ||
		template.AnchorDefinition != bundle.Template.AnchorDefinition || template.LateSubmissionPolicy != bundle.Template.LateSubmissionPolicy ||
		template.PauseStrategy != bundle.Template.PauseStrategy || template.DefinitionSchemaVersion != bundle.Template.DefinitionSchemaVersion ||
		template.DefinitionHash != bundle.Template.DefinitionHash || template.RowVersion != bundle.Template.RowVersion || template.DeletedAt.Valid {
		return fmt.Errorf("synthetic plan template definition differs")
	}
	var dependencyCount int64
	if err := tx.Unscoped().Model(&pathmodel.PlanTaskDependency{}).
		Where("plan_template_version_id = ?", bundle.Template.ID).Count(&dependencyCount).Error; err != nil {
		return err
	}
	if dependencyCount != 0 {
		return fmt.Errorf("synthetic plan template must not contain dependencies")
	}
	var tasks []pathmodel.PlanTaskDefinition
	if err := tx.Unscoped().Where("plan_template_version_id = ?", bundle.Template.ID).Order("sort ASC").Find(&tasks).Error; err != nil {
		return err
	}
	if len(tasks) != 5 {
		return fmt.Errorf("synthetic plan template must contain exactly five tasks")
	}
	for i := range tasks {
		expected := bundle.Tasks[i]
		if tasks[i].ID != expected.ID || tasks[i].PlanTemplateVersionID != expected.PlanTemplateVersionID ||
			tasks[i].DayCode != expected.DayCode || tasks[i].Title != expected.Title || tasks[i].Sort != expected.Sort ||
			tasks[i].ExecutionRole != pathmodel.ExecutionRoleCareClient ||
			tasks[i].OpenOffsetSeconds != expected.OpenOffsetSeconds || tasks[i].DueOffsetSeconds != expected.DueOffsetSeconds ||
			tasks[i].ExpiresOffsetSeconds != nil || !sameUintPointer(tasks[i].QuestionnaireVersionID, expected.QuestionnaireVersionID) ||
			!jsonDocumentEqual(tasks[i].BoundRuleVersionIDsJSON, expected.BoundRuleVersionIDsJSON) ||
			tasks[i].ReviewRequired != expected.ReviewRequired || tasks[i].ReviewRole != expected.ReviewRole ||
			tasks[i].NotificationPolicy != pathmodel.NotificationPolicyDisabled || tasks[i].DeletedAt.Valid {
			return fmt.Errorf("synthetic task definition %s differs", expected.DayCode)
		}
	}
	return nil
}

func seedSyntheticPlanRuntime(tx *gorm.DB, bundle syntheticPlanBundle) error {
	var existing pathmodel.PlanInstance
	err := tx.Unscoped().Where("id = ?", syntheticPlanInstanceID).First(&existing).Error
	if err == nil {
		return verifySyntheticPlanRuntime(tx, bundle)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var client caremodel.CareClient
	if err = tx.Where("id = ? AND synthetic = ?", 20001, true).First(&client).Error; err != nil {
		return fmt.Errorf("synthetic care client fixture missing: %w", err)
	}
	var questionnaire qmodel.QuestionnaireVersion
	if err = tx.Where("id = ? AND synthetic = ?", syntheticQuestionnaireVersionID, true).First(&questionnaire).Error; err != nil {
		return fmt.Errorf("synthetic questionnaire fixture missing: %w", err)
	}
	fixed := carePathFixedTime()
	activeSlot := bundle.Path.Code
	enrollment := pathmodel.Enrollment{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticEnrollmentID}, CareClientID: client.ID,
		PathDefinitionVersionID: bundle.Path.ID, PathCode: bundle.Path.Code, ActiveSlot: &activeSlot,
		Status: pathmodel.EnrollmentActive, StartedAt: &fixed, Version: 1, Synthetic: true,
		DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID,
	}
	consumedAt := fixed
	planID := uint(syntheticPlanInstanceID)
	preview := pathmodel.PlanPreview{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticPlanPreviewID}, PreviewID: syntheticPlanPreviewOpaqueID,
		CareClientID: client.ID, PlanTemplateVersionID: bundle.Template.ID, AnchorAt: fixed,
		ExpiresAt: fixed.Add(30 * time.Minute), TemplateDefinitionHash: bundle.Template.DefinitionHash,
		ConsumedAt: &consumedAt, PlanInstanceID: &planID, Synthetic: true,
		DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID,
	}
	plan := pathmodel.PlanInstance{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticPlanInstanceID}, EnrollmentID: enrollment.ID,
		CareClientID: client.ID, PlanTemplateVersionID: bundle.Template.ID, PreviewID: preview.ID,
		AnchorAt: fixed, Status: pathmodel.EnrollmentActive, PauseStrategy: pathmodel.PauseStrategyKeepWindows,
		Version: 1, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID,
	}
	if err = tx.Create(&enrollment).Error; err != nil {
		return err
	}
	if err = tx.Create(&preview).Error; err != nil {
		return err
	}
	if err = tx.Create(&plan).Error; err != nil {
		return err
	}
	tasks := make([]pathmodel.TaskInstance, 0, 5)
	for i, definition := range bundle.Tasks {
		status := pathmodel.ExecutionScheduled
		var openedAt *time.Time
		if i == 0 {
			status = pathmodel.ExecutionOpen
			openedAt = &fixed
		}
		reviewStatus := pathmodel.ReviewNotRequired
		if definition.ReviewRequired {
			reviewStatus = pathmodel.ReviewNotReady
		}
		tasks = append(tasks, pathmodel.TaskInstance{
			GVA_MODEL:      global.GVA_MODEL{ID: uint(syntheticTaskInstanceD1ID + i)},
			PlanInstanceID: plan.ID, CareClientID: client.ID, TaskDefinitionID: definition.ID,
			DayCode: definition.DayCode, Title: definition.Title, Sort: definition.Sort,
			ExecutionRole: definition.ExecutionRole, ExecutionStatus: status,
			ReviewStatus: reviewStatus, ReviewRole: definition.ReviewRole,
			OpenAt:                  fixed.Add(time.Duration(definition.OpenOffsetSeconds) * time.Second),
			DueAt:                   fixed.Add(time.Duration(definition.DueOffsetSeconds) * time.Second),
			QuestionnaireVersionID:  definition.QuestionnaireVersionID,
			BoundRuleVersionIDsJSON: append(datatypes.JSON(nil), definition.BoundRuleVersionIDsJSON...),
			LateSubmissionPolicy:    pathmodel.LateSubmissionDeny,
			NotificationPolicy:      pathmodel.NotificationPolicyDisabled, OpenedAt: openedAt,
			Version: 1, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID,
		})
	}
	if err = tx.Create(&tasks).Error; err != nil {
		return err
	}
	return ensureSyntheticPlanRuntimeFacts(tx, bundle, enrollment, preview, plan, tasks)
}

func ensureSyntheticPlanRuntimeFacts(tx *gorm.DB, bundle syntheticPlanBundle, enrollment pathmodel.Enrollment, preview pathmodel.PlanPreview, plan pathmodel.PlanInstance, tasks []pathmodel.TaskInstance) error {
	if len(tasks) != 5 {
		return fmt.Errorf("synthetic runtime facts require exactly five tasks")
	}
	fixed := carePathFixedTime()
	d1ID := tasks[0].ID
	events := []pathmodel.CarePathEvent{
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticPlanStartedEventID}, EventID: "00000000-0000-4000-8000-000000009621", EventType: pathmodel.EventPlanStarted, CareClientID: plan.CareClientID, EnrollmentID: enrollment.ID, PlanInstanceID: plan.ID, ActorID: syntheticStewardAID, Source: pathmodel.EventSourceCareSteward, FromStatus: pathmodel.EnrollmentPendingStart, ToStatus: pathmodel.EnrollmentActive, OccurredAt: fixed, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticTaskOpenedEventID}, EventID: "00000000-0000-4000-8000-000000009622", EventType: pathmodel.EventTaskOpened, CareClientID: plan.CareClientID, EnrollmentID: enrollment.ID, PlanInstanceID: plan.ID, TaskInstanceID: &d1ID, ActorID: syntheticStewardAID, Source: pathmodel.EventSourceSystem, FromStatus: pathmodel.ExecutionScheduled, ToStatus: pathmodel.ExecutionOpen, OccurredAt: fixed, Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID},
	}
	for i := range events {
		if err := ensureSyntheticCarePathEvent(tx, events[i]); err != nil {
			return err
		}
	}
	planPayload, err := json.Marshal(map[string]any{
		"careClientId": plan.CareClientID, "enrollmentId": enrollment.ID, "planInstanceId": plan.ID,
		"planTemplateVersionId": bundle.Template.ID, "anchorAt": fixed,
		"taskIds": []uint{tasks[0].ID, tasks[1].ID, tasks[2].ID, tasks[3].ID, tasks[4].ID}, "synthetic": true,
	})
	if err != nil {
		return err
	}
	taskPayload, err := json.Marshal(map[string]any{
		"careClientId": plan.CareClientID, "enrollmentId": enrollment.ID, "planInstanceId": plan.ID,
		"taskInstanceId": tasks[0].ID, "dayCode": tasks[0].DayCode, "openAt": tasks[0].OpenAt,
		"openedAt": fixed, "synthetic": true,
	})
	if err != nil {
		return err
	}
	outboxEvents := []platformoutbox.Event{
		{
			EventID: "00000000-0000-4000-8000-000000009604", EventType: pathmodel.EventPlanStarted,
			PayloadVersion: "v1", AggregateType: "CarePlan", AggregateID: fmt.Sprint(plan.ID),
			PayloadJSON: datatypes.JSON(planPayload), OccurredAt: fixed, CausationID: preview.PreviewID,
			Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID,
		},
		{
			EventID: "00000000-0000-4000-8000-000000009605", EventType: pathmodel.EventTaskOpened,
			PayloadVersion: "v1", AggregateType: "CareTask", AggregateID: fmt.Sprint(tasks[0].ID),
			PayloadJSON: datatypes.JSON(taskPayload), OccurredAt: fixed, CausationID: preview.PreviewID,
			Synthetic: true, DeptId: syntheticTeamAID, CreatedBy: syntheticStewardAID,
		},
	}
	for i := range outboxEvents {
		if err = ensureSyntheticOutboxEvent(tx, outboxEvents[i]); err != nil {
			return err
		}
	}
	return nil
}

func ensureSyntheticCarePathEvent(tx *gorm.DB, expected pathmodel.CarePathEvent) error {
	var actual pathmodel.CarePathEvent
	err := tx.Unscoped().Where("event_id = ?", expected.EventID).First(&actual).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&expected).Error
	}
	if err != nil {
		return err
	}
	actualTaskID, expectedTaskID := uint(0), uint(0)
	if actual.TaskInstanceID != nil {
		actualTaskID = *actual.TaskInstanceID
	}
	if expected.TaskInstanceID != nil {
		expectedTaskID = *expected.TaskInstanceID
	}
	if actual.ID != expected.ID || actual.EventType != expected.EventType || actual.CareClientID != expected.CareClientID ||
		actual.EnrollmentID != expected.EnrollmentID || actual.PlanInstanceID != expected.PlanInstanceID || actualTaskID != expectedTaskID ||
		actual.ActorID != expected.ActorID || actual.Source != expected.Source || actual.FromStatus != expected.FromStatus ||
		actual.ToStatus != expected.ToStatus || !actual.OccurredAt.Equal(expected.OccurredAt) || !actual.Synthetic ||
		actual.DeptId != expected.DeptId || actual.CreatedBy != expected.CreatedBy || actual.DeletedAt.Valid {
		return fmt.Errorf("synthetic care path event %s differs", expected.EventID)
	}
	return nil
}

func ensureSyntheticOutboxEvent(tx *gorm.DB, expected platformoutbox.Event) error {
	var actual platformoutbox.Event
	err := tx.Unscoped().Where("event_id = ?", expected.EventID).First(&actual).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&expected).Error
	}
	if err != nil {
		return err
	}
	if actual.EventType != expected.EventType || actual.PayloadVersion != expected.PayloadVersion ||
		actual.AggregateType != expected.AggregateType || actual.AggregateID != expected.AggregateID ||
		!jsonDocumentEqual(actual.PayloadJSON, expected.PayloadJSON) || !actual.OccurredAt.Equal(expected.OccurredAt) ||
		actual.CorrelationID != expected.CorrelationID || actual.CausationID != expected.CausationID ||
		!actual.Synthetic || actual.DeptId != expected.DeptId || actual.CreatedBy != expected.CreatedBy || actual.DeletedAt.Valid {
		return fmt.Errorf("synthetic outbox event %s differs", expected.EventID)
	}
	return nil
}

func verifySyntheticPlanRuntime(tx *gorm.DB, bundle syntheticPlanBundle) error {
	var plan pathmodel.PlanInstance
	if err := tx.Unscoped().Where("id = ?", syntheticPlanInstanceID).First(&plan).Error; err != nil {
		return err
	}
	if plan.EnrollmentID != syntheticEnrollmentID || plan.CareClientID != 20001 ||
		plan.PlanTemplateVersionID != bundle.Template.ID || plan.PreviewID != syntheticPlanPreviewID ||
		!plan.AnchorAt.Equal(carePathFixedTime()) || plan.PauseStrategy != pathmodel.PauseStrategyKeepWindows ||
		!plan.Synthetic || plan.DeptId != syntheticTeamAID || plan.CreatedBy != syntheticStewardAID || plan.DeletedAt.Valid {
		return fmt.Errorf("synthetic plan runtime differs")
	}
	var enrollment pathmodel.Enrollment
	if err := tx.Unscoped().Where("id = ?", plan.EnrollmentID).First(&enrollment).Error; err != nil {
		return err
	}
	if enrollment.CareClientID != plan.CareClientID || enrollment.PathDefinitionVersionID != bundle.Path.ID ||
		enrollment.PathCode != bundle.Path.Code || enrollment.StartedAt == nil || !enrollment.StartedAt.Equal(carePathFixedTime()) ||
		!enrollment.Synthetic || enrollment.DeptId != syntheticTeamAID || enrollment.CreatedBy != syntheticStewardAID || enrollment.DeletedAt.Valid {
		return fmt.Errorf("synthetic enrollment runtime differs")
	}
	var preview pathmodel.PlanPreview
	if err := tx.Unscoped().Where("id = ?", plan.PreviewID).First(&preview).Error; err != nil {
		return err
	}
	if preview.PreviewID != syntheticPlanPreviewOpaqueID || preview.CareClientID != plan.CareClientID ||
		preview.PlanTemplateVersionID != bundle.Template.ID || !preview.AnchorAt.Equal(carePathFixedTime()) ||
		!preview.ExpiresAt.Equal(carePathFixedTime().Add(30*time.Minute)) ||
		preview.TemplateDefinitionHash != bundle.Template.DefinitionHash || preview.ConsumedAt == nil ||
		preview.PlanInstanceID == nil || *preview.PlanInstanceID != plan.ID || !preview.Synthetic ||
		preview.DeptId != syntheticTeamAID || preview.CreatedBy != syntheticStewardAID || preview.DeletedAt.Valid {
		return fmt.Errorf("synthetic plan preview runtime differs")
	}
	var tasks []pathmodel.TaskInstance
	if err := tx.Unscoped().Where("plan_instance_id = ?", plan.ID).Order("sort ASC").Find(&tasks).Error; err != nil {
		return err
	}
	if len(tasks) != 5 {
		return fmt.Errorf("synthetic runtime plan must contain exactly five tasks")
	}
	for i, task := range tasks {
		expected := bundle.Tasks[i]
		expectedOpen := carePathFixedTime().Add(time.Duration(expected.OpenOffsetSeconds) * time.Second)
		expectedDue := carePathFixedTime().Add(time.Duration(expected.DueOffsetSeconds) * time.Second)
		if task.ID != uint(syntheticTaskInstanceD1ID+i) || task.DayCode != fmt.Sprintf("D%d", i+1) ||
			task.CareClientID != plan.CareClientID || task.TaskDefinitionID != expected.ID || task.Title != expected.Title ||
			task.Sort != expected.Sort || task.ExecutionRole != pathmodel.ExecutionRoleCareClient || task.ReviewRole != expected.ReviewRole ||
			!task.OpenAt.Equal(expectedOpen) || !task.DueAt.Equal(expectedDue) || task.ExpiresAt != nil ||
			!sameUintPointer(task.QuestionnaireVersionID, expected.QuestionnaireVersionID) ||
			!jsonDocumentEqual(task.BoundRuleVersionIDsJSON, expected.BoundRuleVersionIDsJSON) ||
			task.LateSubmissionPolicy != pathmodel.LateSubmissionDeny || task.NotificationPolicy != pathmodel.NotificationPolicyDisabled ||
			!task.Synthetic || task.DeptId != syntheticTeamAID || task.CreatedBy != syntheticStewardAID || task.DeletedAt.Valid {
			return fmt.Errorf("synthetic runtime task D%d differs", i+1)
		}
	}
	return ensureSyntheticPlanRuntimeFacts(tx, bundle, enrollment, preview, plan, tasks)
}

func sameUintPointer(left, right *uint) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameTimePointer(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func jsonDocumentEqual(left, right []byte) bool {
	var leftValue, rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}
	leftCanonical, leftErr := json.Marshal(leftValue)
	rightCanonical, rightErr := json.Marshal(rightValue)
	return leftErr == nil && rightErr == nil && string(leftCanonical) == string(rightCanonical)
}

func carePathFixedTime() time.Time {
	return time.Date(2026, time.August, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
}
