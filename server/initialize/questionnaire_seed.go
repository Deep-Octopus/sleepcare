package initialize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	syntheticQuestionnaireVersionID = 9401
	syntheticQuestionID             = 940101
	syntheticContinueOptionID       = 940111
	syntheticAttentionOptionID      = 940112
	syntheticRuleVersionID          = 9501
)

var questionnaireAPIs = []system.SysApi{
	{ApiGroup: "问卷版本", Method: "GET", Path: "/care/questionnaire-versions", Description: "获取问卷版本预览列表"},
	{ApiGroup: "问卷版本", Method: "GET", Path: "/care/questionnaire-versions/:id", Description: "获取问卷与关注规则版本详情"},
}

var questionnaireShellAPIs = []struct {
	Path   string
	Method string
}{
	{Path: "/menu/getMenu", Method: "POST"},
	{Path: "/user/getUserInfo", Method: "GET"},
	{Path: "/jwt/jsonInBlacklist", Method: "POST"},
}

func EnsureQuestionnaireData() error {
	if global.GVA_DB == nil {
		return nil
	}
	ctx := datascope.WithSystem(context.Background())
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureQuestionnaireMetadata(db); err != nil {
		return err
	}
	if !global.GVA_CONFIG.Care.SyntheticFixturesEnabled {
		return nil
	}
	return ensureQuestionnaireSyntheticFixtures(db)
}

func ensureQuestionnaireMetadata(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, api := range questionnaireAPIs {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).Attrs(api).FirstOrCreate(&system.SysApi{}).Error; err != nil {
				return fmt.Errorf("ensure questionnaire API %s: %w", api.Path, err)
			}
		}
		root := system.SysBaseMenu{Path: "sleep-care", Name: "SleepCare", Component: "view/routerHolder.vue", Sort: 20, Meta: system.Meta{Title: "睡眠康养随访", Icon: "user"}}
		if err := tx.Where("name = ?", root.Name).Attrs(root).FirstOrCreate(&root).Error; err != nil {
			return err
		}
		leaf := system.SysBaseMenu{
			ParentId: root.ID, Path: "questionnaire-versions", Name: "CareQuestionnaires",
			Component: "view/sleep-care/questionnaires/index.vue", Sort: 2,
			Meta: system.Meta{Title: "问卷版本", Icon: "version-gva"},
		}
		if err := tx.Where("name = ?", leaf.Name).Attrs(leaf).FirstOrCreate(&leaf).Error; err != nil {
			return err
		}
		button := system.SysBaseMenuBtn{Name: "preview", Desc: "预览问卷版本", SysBaseMenuID: leaf.ID}
		return tx.Where("name = ? AND sys_base_menu_id = ?", button.Name, leaf.ID).Attrs(button).FirstOrCreate(&button).Error
	})
}

func ensureQuestionnaireSyntheticFixtures(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedSyntheticQuestionnaire(tx); err != nil {
			return err
		}
		return grantQuestionnaireAccess(tx)
	})
}

func seedSyntheticQuestionnaire(tx *gorm.DB) error {
	fixed := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	validation := mustJSON(map[string]any{})
	condition := mustJSON(map[string]any{
		"questionCode": "synthetic_process_confirmation",
		"operator":     qmodel.RuleOperatorEquals,
		"value":        "CONTINUE_WITH_ATTENTION",
	})
	recipients := mustJSON([]string{"ASSIGNED_CLINICIAN", "SUPERVISOR"})
	versionDefinition := qmodel.VersionDefinition{
		Code: "SYN-WORKFLOW-CHECK", Version: "1.0.0-synthetic", Title: "合成流程验证问卷（非医疗内容）",
		Purpose:    "仅验证打开、草稿、提交、规则命中和人工闭环的软件行为。",
		UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		ExpectedMinutes: 1, DefinitionSchemaVersion: "v1",
		Questions: []qmodel.QuestionDefinition{{
			Code: "synthetic_process_confirmation", Type: qmodel.QuestionTypeSingleChoice,
			Title: "是否继续完成本次合成流程验证？（非医疗问题）", Required: true, Sort: 1,
			ValidationSchemaVersion: "v1", Validation: validation,
			Options: []qmodel.OptionDefinition{
				{Code: "CONTINUE_WITHOUT_ATTENTION", Label: "继续验证，不创建人工关注流程", Sort: 1},
				{Code: "CONTINUE_WITH_ATTENTION", Label: "继续验证，并创建人工关注流程", Sort: 2},
			},
		}},
	}
	versionHash, err := qmodel.HashDefinition(versionDefinition)
	if err != nil {
		return err
	}
	ruleDefinition := qmodel.RuleDefinition{
		QuestionnaireVersionID: syntheticQuestionnaireVersionID, Code: "SYN-MANUAL-ATTENTION-FLOW", Version: "1.0.0-synthetic",
		Title: "合成人工关注流程规则（非医疗规则）", UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		ConditionSchemaVersion: "v1", Condition: condition, AttentionLevel: "SYNTHETIC_ATTENTION",
		ReasonSnapshot: "合成流程选项请求创建人工关注链；不表示健康风险或诊断。",
		Recipients:     recipients, DedupKeyTemplate: "submission:{submissionId}:rule:9501",
	}
	ruleHash, err := qmodel.HashDefinition(ruleDefinition)
	if err != nil {
		return err
	}
	version := qmodel.QuestionnaireVersion{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticQuestionnaireVersionID},
		Code:      versionDefinition.Code, Version: versionDefinition.Version, Title: versionDefinition.Title, Purpose: versionDefinition.Purpose,
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		ReviewType: qmodel.ReviewTypeEngineering, ReviewedBy: syntheticSupervisorAID, ReviewedAt: &fixed,
		ReviewNote: "合成软件流程工程复核；不包含医疗审批。", ExpectedMinutes: 1, PublishedAt: &fixed,
		DefinitionSchemaVersion: "v1", DefinitionHash: versionHash, RowVersion: 1,
	}
	var existing qmodel.QuestionnaireVersion
	err = tx.Unscoped().Where("id = ?", version.ID).First(&existing).Error
	if err == nil {
		return verifySyntheticQuestionnaire(tx, versionHash, ruleHash)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err = tx.Create(&version).Error; err != nil {
		return err
	}
	question := qmodel.QuestionnaireQuestion{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticQuestionID}, QuestionnaireVersionID: version.ID,
		Code: versionDefinition.Questions[0].Code, Type: versionDefinition.Questions[0].Type,
		Title: versionDefinition.Questions[0].Title, Required: true, Sort: 1,
		ValidationSchemaVersion: "v1", ValidationJSON: datatypes.JSON(validation),
	}
	if err = tx.Create(&question).Error; err != nil {
		return err
	}
	options := []qmodel.QuestionnaireOption{
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticContinueOptionID}, QuestionID: question.ID, Code: "CONTINUE_WITHOUT_ATTENTION", Label: "继续验证，不创建人工关注流程", Sort: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: syntheticAttentionOptionID}, QuestionID: question.ID, Code: "CONTINUE_WITH_ATTENTION", Label: "继续验证，并创建人工关注流程", Sort: 2},
	}
	if err = tx.Create(&options).Error; err != nil {
		return err
	}
	rule := qmodel.QuestionnaireRuleVersion{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticRuleVersionID}, QuestionnaireVersionID: version.ID,
		Code: ruleDefinition.Code, Version: ruleDefinition.Version, Title: ruleDefinition.Title,
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		ReviewType: qmodel.ReviewTypeEngineering, ReviewedBy: syntheticSupervisorAID, ReviewedAt: &fixed,
		ReviewNote: "合成规则工程复核；不包含正式医疗审批。", ConditionSchemaVersion: "v1",
		ConditionJSON: datatypes.JSON(condition), AttentionLevel: ruleDefinition.AttentionLevel,
		ReasonSnapshot: ruleDefinition.ReasonSnapshot, RecipientsJSON: datatypes.JSON(recipients),
		DedupKeyTemplate: ruleDefinition.DedupKeyTemplate, PublishedAt: &fixed, DefinitionHash: ruleHash, RowVersion: 1,
	}
	return tx.Create(&rule).Error
}

func verifySyntheticQuestionnaire(tx *gorm.DB, expectedVersionHash, expectedRuleHash string) error {
	var version qmodel.QuestionnaireVersion
	if err := tx.Unscoped().Where("id = ?", syntheticQuestionnaireVersionID).First(&version).Error; err != nil {
		return err
	}
	if version.Code != "SYN-WORKFLOW-CHECK" || version.Version != "1.0.0-synthetic" || !version.Synthetic || version.DefinitionHash != expectedVersionHash || version.DeletedAt.Valid {
		return fmt.Errorf("synthetic questionnaire version id %d is occupied or definition differs", syntheticQuestionnaireVersionID)
	}
	var question qmodel.QuestionnaireQuestion
	if err := tx.Unscoped().Where("id = ? AND questionnaire_version_id = ?", syntheticQuestionID, version.ID).First(&question).Error; err != nil {
		return fmt.Errorf("synthetic questionnaire question differs: %w", err)
	}
	if question.Code != "synthetic_process_confirmation" || question.Type != qmodel.QuestionTypeSingleChoice || question.DeletedAt.Valid {
		return fmt.Errorf("synthetic questionnaire question definition differs")
	}
	var options []qmodel.QuestionnaireOption
	if err := tx.Unscoped().Where("question_id = ?", question.ID).Order("sort ASC").Find(&options).Error; err != nil {
		return err
	}
	if len(options) != 2 || options[0].Code != "CONTINUE_WITHOUT_ATTENTION" || options[1].Code != "CONTINUE_WITH_ATTENTION" {
		return fmt.Errorf("synthetic questionnaire option definition differs")
	}
	actualVersionDefinition := qmodel.VersionDefinition{
		Code: version.Code, Version: version.Version, Title: version.Title, Purpose: version.Purpose,
		UsageScope: version.UsageScope, Synthetic: version.Synthetic, ProductionEnabled: version.ProductionEnabled,
		ExpectedMinutes: version.ExpectedMinutes, DefinitionSchemaVersion: version.DefinitionSchemaVersion,
		Questions: []qmodel.QuestionDefinition{{
			Code: question.Code, Type: question.Type, Title: question.Title, Required: question.Required, Sort: question.Sort,
			ValidationSchemaVersion: question.ValidationSchemaVersion, Validation: qmodel.CanonicalJSON(json.RawMessage(question.ValidationJSON)),
			Options: []qmodel.OptionDefinition{
				{Code: options[0].Code, Label: options[0].Label, Sort: options[0].Sort},
				{Code: options[1].Code, Label: options[1].Label, Sort: options[1].Sort},
			},
		}},
	}
	actualVersionHash, err := qmodel.HashDefinition(actualVersionDefinition)
	if err != nil || actualVersionHash != expectedVersionHash {
		return fmt.Errorf("synthetic questionnaire version definition differs")
	}
	var rule qmodel.QuestionnaireRuleVersion
	if err := tx.Unscoped().Where("id = ?", syntheticRuleVersionID).First(&rule).Error; err != nil {
		return fmt.Errorf("synthetic questionnaire rule differs: %w", err)
	}
	if rule.QuestionnaireVersionID != version.ID || rule.Code != "SYN-MANUAL-ATTENTION-FLOW" || rule.DefinitionHash != expectedRuleHash || rule.DeletedAt.Valid {
		return fmt.Errorf("synthetic questionnaire rule definition differs")
	}
	actualRuleDefinition := qmodel.RuleDefinition{
		QuestionnaireVersionID: rule.QuestionnaireVersionID, Code: rule.Code, Version: rule.Version, Title: rule.Title,
		UsageScope: rule.UsageScope, Synthetic: rule.Synthetic, ProductionEnabled: rule.ProductionEnabled,
		ConditionSchemaVersion: rule.ConditionSchemaVersion, Condition: qmodel.CanonicalJSON(json.RawMessage(rule.ConditionJSON)),
		AttentionLevel: rule.AttentionLevel, ReasonSnapshot: rule.ReasonSnapshot,
		Recipients: qmodel.CanonicalJSON(json.RawMessage(rule.RecipientsJSON)), SLAFirstResponseMinutes: rule.SLAFirstResponseMinutes,
		SLAResolutionMinutes: rule.SLAResolutionMinutes, DedupKeyTemplate: rule.DedupKeyTemplate,
	}
	actualRuleHash, err := qmodel.HashDefinition(actualRuleDefinition)
	if err != nil || actualRuleHash != expectedRuleHash {
		return fmt.Errorf("synthetic questionnaire rule definition differs")
	}
	return nil
}

func grantQuestionnaireAccess(tx *gorm.DB) error {
	var root, leaf system.SysBaseMenu
	if err := tx.Where("name = ?", "SleepCare").First(&root).Error; err != nil {
		return err
	}
	if err := tx.Where("name = ?", "CareQuestionnaires").First(&leaf).Error; err != nil {
		return err
	}
	var button system.SysBaseMenuBtn
	if err := tx.Where("name = ? AND sys_base_menu_id = ?", "preview", leaf.ID).First(&button).Error; err != nil {
		return err
	}
	for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole} {
		for _, api := range questionnaireShellAPIs {
			policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
			if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
				return err
			}
		}
	}
	for _, role := range []uint{syntheticClinicianRole, syntheticSupervisorRole} {
		for _, menuID := range []uint{root.ID, leaf.ID} {
			link := system.SysAuthorityMenu{MenuId: fmt.Sprint(menuID), AuthorityId: fmt.Sprint(role)}
			if err := tx.Where(link).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
		buttonLink := system.SysAuthorityBtn{AuthorityId: role, SysMenuID: leaf.ID, SysBaseMenuBtnID: button.ID}
		if err := tx.Where("authority_id = ? AND sys_menu_id = ? AND sys_base_menu_btn_id = ?", role, leaf.ID, button.ID).FirstOrCreate(&buttonLink).Error; err != nil {
			return err
		}
		for _, api := range questionnaireAPIs {
			policy := adapter.CasbinRule{Ptype: "p", V0: fmt.Sprint(role), V1: api.Path, V2: api.Method}
			if err := tx.Where(policy).FirstOrCreate(&policy).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func mustJSON(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}
