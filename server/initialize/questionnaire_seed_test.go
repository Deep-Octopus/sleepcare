package initialize

import (
	"context"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

func TestEnsureQuestionnaireSyntheticFixturesIsStrictIdempotentAndRoleLimited(t *testing.T) {
	db := testutil.NewMemoryDB(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthority{}, &system.SysDepartment{},
		&system.SysUser{}, &system.SysUserAuthority{}, &system.SysUserDepartment{}, &system.SysAuthorityMenu{}, &system.SysAuthorityBtn{},
		&adapter.CasbinRule{},
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.ConsentRecord{}, &caremodel.CareOrgUnitProfile{}, &caremodel.CareAuthorityProfile{}, &caremodel.CareClientCommandReceipt{},
		&qmodel.QuestionnaireVersion{}, &qmodel.QuestionnaireQuestion{}, &qmodel.QuestionnaireOption{}, &qmodel.QuestionnaireRuleVersion{},
		&qmodel.QuestionnaireSubmission{}, &qmodel.QuestionnaireAnswerRevision{}, &qmodel.QuestionnaireRuleHit{}, &qmodel.QuestionnaireCommandReceipt{}, &qmodel.OutboxEvent{},
	)
	db = db.WithContext(datascope.WithSystem(context.Background()))
	if err := ensureCareClientMetadata(db); err != nil {
		t.Fatal(err)
	}
	if err := ensureCareClientSyntheticFixtures(db, "local-synthetic-password"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := ensureQuestionnaireMetadata(db); err != nil {
			t.Fatal(err)
		}
		if err := ensureQuestionnaireSyntheticFixtures(db); err != nil {
			t.Fatal(err)
		}
	}

	assertSeedCount(t, db, &qmodel.QuestionnaireVersion{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &qmodel.QuestionnaireQuestion{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &qmodel.QuestionnaireOption{}, "1 = 1", nil, 2)
	assertSeedCount(t, db, &qmodel.QuestionnaireRuleVersion{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &qmodel.QuestionnaireVersion{}, "usage_scope = ?", qmodel.UsageScopeFormal, 0)
	assertSeedCount(t, db, &qmodel.QuestionnaireSubmission{}, "1 = 1", nil, 0)
	assertSeedCount(t, db, &qmodel.OutboxEvent{}, "1 = 1", nil, 0)

	var leaf system.SysBaseMenu
	if err := db.Where("name = ?", "CareQuestionnaires").First(&leaf).Error; err != nil {
		t.Fatal(err)
	}
	for _, role := range []uint{syntheticClinicianRole, syntheticSupervisorRole} {
		assertSeedCount(t, db, &system.SysAuthorityMenu{}, "sys_base_menu_id = ? AND sys_authority_authority_id = ?", []any{leaf.ID, role}, 1)
		for _, api := range questionnaireAPIs {
			assertSeedCount(t, db, &adapter.CasbinRule{}, "v0 = ? AND v1 = ? AND v2 = ?", []any{role, api.Path, api.Method}, 1)
		}
	}
	for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole} {
		for _, api := range questionnaireShellAPIs {
			assertSeedCount(t, db, &adapter.CasbinRule{}, "v0 = ? AND v1 = ? AND v2 = ?", []any{role, api.Path, api.Method}, 1)
		}
	}
	for _, role := range []uint{syntheticStewardRole, 888} {
		assertSeedCount(t, db, &system.SysAuthorityMenu{}, "sys_base_menu_id = ? AND sys_authority_authority_id = ?", []any{leaf.ID, role}, 0)
		assertSeedCount(t, db, &adapter.CasbinRule{}, "v0 = ? AND v1 LIKE ?", []any{role, "/care/questionnaire-%"}, 0)
	}

	if err := db.Model(&qmodel.QuestionnaireQuestion{}).Where("id = ?", syntheticQuestionID).Update("title", "被篡改的合成题目").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureQuestionnaireSyntheticFixtures(db); err == nil {
		t.Fatal("definition collision should fail instead of overwriting content")
	}
	var question qmodel.QuestionnaireQuestion
	if err := db.Where("id = ?", syntheticQuestionID).First(&question).Error; err != nil {
		t.Fatal(err)
	}
	if question.Title != "被篡改的合成题目" {
		t.Fatal("strict seed unexpectedly overwrote existing content")
	}
}

func assertSeedCount(t *testing.T, db *gorm.DB, model any, where string, args any, want int64) {
	t.Helper()
	query := db.Model(model)
	if where != "1 = 1" {
		if values, ok := args.([]any); ok {
			query = query.Where(where, values...)
		} else {
			query = query.Where(where, args)
		}
	}
	var got int64
	if err := query.Count(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%T count=%d, want %d", model, got, want)
	}
}
