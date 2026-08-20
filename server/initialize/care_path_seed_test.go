package initialize

import (
	"context"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestEnsureCarePathSyntheticFixturesIsStrictIdempotentAndRoleLimited(t *testing.T) {
	db := testutil.NewMemoryDB(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{}, &system.SysAuthority{}, &system.SysDepartment{},
		&system.SysUser{}, &system.SysUserAuthority{}, &system.SysUserDepartment{}, &system.SysAuthorityMenu{}, &system.SysAuthorityBtn{},
		&adapter.CasbinRule{},
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.ConsentRecord{}, &caremodel.CareOrgUnitProfile{}, &caremodel.CareAuthorityProfile{}, &caremodel.CareClientCommandReceipt{},
		&clientmodel.CareClientAccount{}, &clientmodel.CareClientCredential{}, &clientmodel.ClientAccessGrant{}, &clientmodel.ClientSession{}, &clientmodel.ClientTaskCommandReceipt{},
		&qmodel.QuestionnaireVersion{}, &qmodel.QuestionnaireQuestion{}, &qmodel.QuestionnaireOption{}, &qmodel.QuestionnaireRuleVersion{},
		&qmodel.QuestionnaireSubmission{}, &qmodel.QuestionnaireAnswerRevision{}, &qmodel.QuestionnaireRuleHit{}, &qmodel.QuestionnaireCommandReceipt{},
		&pathmodel.PathDefinitionVersion{}, &pathmodel.PlanTemplateVersion{}, &pathmodel.PlanTaskDefinition{}, &pathmodel.PlanTaskDependency{},
		&pathmodel.Enrollment{}, &pathmodel.PlanPreview{}, &pathmodel.PlanInstance{}, &pathmodel.TaskInstance{}, &pathmodel.CarePathEvent{}, &pathmodel.CommandReceipt{},
		&platformoutbox.Event{},
	)
	db = db.WithContext(datascope.WithSystem(context.Background()))
	if err := ensureCareClientMetadata(db); err != nil {
		t.Fatal(err)
	}
	if err := ensureCareClientSyntheticFixtures(db, "local-synthetic-password"); err != nil {
		t.Fatal(err)
	}
	if err := ensureQuestionnaireMetadata(db); err != nil {
		t.Fatal(err)
	}
	if err := ensureQuestionnaireSyntheticFixtures(db); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := ensureCarePathMetadata(db); err != nil {
			t.Fatal(err)
		}
		if err := ensureCarePathSyntheticFixtures(db); err != nil {
			t.Fatal(err)
		}
		if err := ensureClientAccessFixture(db, "local-synthetic-password"); err != nil {
			t.Fatal(err)
		}
	}

	assertSeedCount(t, db, &pathmodel.PathDefinitionVersion{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &pathmodel.PlanTemplateVersion{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &pathmodel.PlanTaskDefinition{}, "1 = 1", nil, 5)
	assertSeedCount(t, db, &pathmodel.PlanTaskDependency{}, "1 = 1", nil, 0)
	assertSeedCount(t, db, &pathmodel.Enrollment{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &pathmodel.PlanPreview{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &pathmodel.PlanInstance{}, "1 = 1", nil, 1)
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "1 = 1", nil, 5)
	assertSeedCount(t, db, &pathmodel.CarePathEvent{}, "1 = 1", nil, 2)
	assertSeedCount(t, db, &pathmodel.CarePathEvent{}, "event_type = ?", pathmodel.EventTaskOpened, 1)
	assertSeedCount(t, db, &platformoutbox.Event{}, "1 = 1", nil, 2)
	assertSeedCount(t, db, &platformoutbox.Event{}, "event_type = ?", pathmodel.EventTaskOpened, 1)
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "execution_role = ?", pathmodel.ExecutionRoleCareClient, 5)
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "questionnaire_version_id IS NULL", []any{}, 4)
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "notification_policy = ?", pathmodel.NotificationPolicyDisabled, 5)
	assertSeedCount(t, db, &clientmodel.CareClientAccount{}, "care_client_id = ?", 20001, 1)
	assertSeedCount(t, db, &clientmodel.CareClientCredential{}, "account_id = ?", clientAccessFixtureAccountID, 1)
	assertSeedCount(t, db, &clientmodel.ClientAccessGrant{}, "1 = 1", nil, 0)
	assertSeedCount(t, db, &clientmodel.ClientSession{}, "1 = 1", nil, 0)
	var credential clientmodel.CareClientCredential
	if err := db.Where("account_id = ?", clientAccessFixtureAccountID).First(&credential).Error; err != nil {
		t.Fatal(err)
	}
	if credential.Username != clientAccessFixtureUsername || !utils.BcryptCheck("local-synthetic-password", credential.PasswordHash) {
		t.Fatal("client credential seed does not match its fixed account")
	}
	if err := ensureClientAccessFixture(db, "rotated-client-password"); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("account_id = ?", clientAccessFixtureAccountID).First(&credential).Error; err != nil {
		t.Fatal(err)
	}
	if !utils.BcryptCheck("rotated-client-password", credential.PasswordHash) || utils.BcryptCheck("local-synthetic-password", credential.PasswordHash) {
		t.Fatal("client credential password did not rotate safely")
	}

	for _, menuName := range []string{"CarePlans", "CareTasks"} {
		var menu system.SysBaseMenu
		if err := db.Where("name = ?", menuName).First(&menu).Error; err != nil {
			t.Fatal(err)
		}
		for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole} {
			assertSeedCount(t, db, &system.SysAuthorityMenu{}, "sys_base_menu_id = ? AND sys_authority_authority_id = ?", []any{menu.ID, role}, 1)
		}
		assertSeedCount(t, db, &system.SysAuthorityMenu{}, "sys_base_menu_id = ? AND sys_authority_authority_id = ?", []any{menu.ID, 888}, 0)
	}
	writePaths := map[string]bool{
		"POST /care/clients/:id/plan-previews":  true,
		"POST /care/clients/:id/plan-instances": true,
		"POST /care/plan-instances/:id/pause":   true,
		"POST /care/plan-instances/:id/resume":  true,
	}
	for _, api := range carePathAPIs {
		for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole} {
			assertSeedCount(t, db, &adapter.CasbinRule{}, "v0 = ? AND v1 = ? AND v2 = ?", []any{role, api.Path, api.Method}, 1)
		}
		wantSupervisor := int64(1)
		if writePaths[api.Method+" "+api.Path] {
			wantSupervisor = 0
		}
		assertSeedCount(t, db, &adapter.CasbinRule{}, "v0 = ? AND v1 = ? AND v2 = ?", []any{syntheticSupervisorRole, api.Path, api.Method}, wantSupervisor)
		assertSeedCount(t, db, &adapter.CasbinRule{}, "v0 = ? AND v1 = ? AND v2 = ?", []any{888, api.Path, api.Method}, 0)
	}

	if err := db.Model(&pathmodel.PlanInstance{}).Where("id = ?", syntheticPlanInstanceID).
		Updates(map[string]any{"status": pathmodel.EnrollmentPaused, "version": 2}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&pathmodel.Enrollment{}).Where("id = ?", syntheticEnrollmentID).
		Updates(map[string]any{"status": pathmodel.EnrollmentPaused, "version": 2}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&pathmodel.TaskInstance{}).Where("id = ?", syntheticTaskInstanceD1ID+1).
		Updates(map[string]any{"execution_status": pathmodel.ExecutionOpen, "version": 2}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureCarePathSyntheticFixtures(db); err != nil {
		t.Fatalf("legitimate runtime transitions must survive seed reruns: %v", err)
	}
	var transitionedPlan pathmodel.PlanInstance
	if err := db.Where("id = ?", syntheticPlanInstanceID).First(&transitionedPlan).Error; err != nil {
		t.Fatal(err)
	}
	if transitionedPlan.Status != pathmodel.EnrollmentPaused || transitionedPlan.Version != 2 {
		t.Fatalf("seed overwrote runtime plan transition: %+v", transitionedPlan)
	}

	if err := db.Model(&pathmodel.PlanTaskDefinition{}).Where("id = ?", syntheticTaskDefinitionD1ID).Update("title", "被篡改的测试任务").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureCarePathSyntheticFixtures(db); err == nil {
		t.Fatal("definition collision should fail instead of overwriting content")
	}
	var task pathmodel.PlanTaskDefinition
	if err := db.Where("id = ?", syntheticTaskDefinitionD1ID).First(&task).Error; err != nil {
		t.Fatal(err)
	}
	if task.Title != "被篡改的测试任务" {
		t.Fatal("strict seed unexpectedly overwrote existing content")
	}
}
