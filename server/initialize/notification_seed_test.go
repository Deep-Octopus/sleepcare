package initialize

import (
	"context"
	"fmt"
	"testing"
	"time"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestEnsureNotificationMetadataIsIdempotentAndRoleLimited(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&system.SysApi{}, &system.SysBaseMenu{}, &system.SysBaseMenuBtn{},
		&system.SysAuthorityMenu{}, &system.SysAuthorityBtn{}, &adapter.CasbinRule{},
	).WithContext(datascope.WithSystem(context.Background()))
	root := system.SysBaseMenu{Path: "sleep-care", Name: "SleepCare", Component: "view/routerHolder.vue", Meta: system.Meta{Title: "睡眠照护"}}
	if err := db.Create(&root).Error; err != nil {
		t.Fatal(err)
	}
	execution := system.SysBaseMenu{ParentId: root.ID, Path: "care-execution", Name: "CareExecution", Component: "view/routerHolder.vue"}
	taskMenu := system.SysBaseMenu{ParentId: execution.ID, Path: "care-tasks", Name: "CareTasks", Component: "view/sleep-care/tasks/index.vue"}
	if err := db.Create(&execution).Error; err != nil {
		t.Fatal(err)
	}
	taskMenu.ParentId = execution.ID
	if err := db.Create(&taskMenu).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := ensureNotificationMetadata(db, true); err != nil {
			t.Fatal(err)
		}
	}
	var deliveryMenu system.SysBaseMenu
	if err := db.Where("name = ?", "CareDeliveries").First(&deliveryMenu).Error; err != nil {
		t.Fatal(err)
	}
	for _, role := range []uint{syntheticStewardRole, syntheticClinicianRole, syntheticSupervisorRole} {
		assertCaseWorkPolicy(t, db, role, "/care/deliveries", "GET", true)
		assertCaseWorkPolicy(t, db, role, "/care/notification-provider-readiness", "GET", true)
	}
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/deliveries/:id/resend", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/deliveries/:id/resend", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/deliveries/:id/resend", "POST", false)
	assertCaseWorkPolicy(t, db, syntheticStewardRole, "/care/tasks/:id/contact-records", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticClinicianRole, "/care/tasks/:id/contact-records", "POST", true)
	assertCaseWorkPolicy(t, db, syntheticSupervisorRole, "/care/tasks/:id/contact-records", "POST", false)
	assertCaseWorkButton(t, db, deliveryMenu.ID, "resendNotice", syntheticStewardRole, true)
	assertCaseWorkButton(t, db, deliveryMenu.ID, "resendNotice", syntheticClinicianRole, false)
	assertCaseWorkButton(t, db, taskMenu.ID, "recordContact", syntheticStewardRole, true)
	assertCaseWorkButton(t, db, taskMenu.ID, "recordContact", syntheticClinicianRole, true)
}

func TestScenarioBNotificationFixturesAreIdempotentAndPreserveTasks(t *testing.T) {
	db := testutil.NewMemoryDBWithoutGlobal(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{},
		&pathmodel.PathDefinitionVersion{}, &pathmodel.PlanTemplateVersion{}, &pathmodel.PlanTaskDefinition{},
		&pathmodel.Enrollment{}, &pathmodel.PlanPreview{}, &pathmodel.PlanInstance{}, &pathmodel.TaskInstance{}, &pathmodel.CarePathEvent{},
		&caseworkmodel.TodoItem{},
		&notificationmodel.NotificationRequest{}, &notificationmodel.NotificationAttempt{}, &notificationmodel.DeliveryEvent{},
		&platformoutbox.Event{},
	).WithContext(datascope.WithSystem(context.Background()))
	seedNotificationPrerequisites(t, db)
	ctx := datascope.WithSystem(context.Background())
	for i := 0; i < 2; i++ {
		if err := ensureScenarioBPlan(db); err != nil {
			t.Fatal(err)
		}
		if err := ensureNotificationFixtures(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "plan_instance_id = ?", scenarioBPlanID, 5)
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "plan_instance_id = ? AND execution_status = ?", []any{scenarioBPlanID, pathmodel.ExecutionOpen}, 1)
	assertSeedCount(t, db, &pathmodel.TaskInstance{}, "plan_instance_id = ? AND execution_status = ?", []any{scenarioBPlanID, pathmodel.ExecutionScheduled}, 4)
	assertSeedCount(t, db, &notificationmodel.NotificationAttempt{}, "notification_request_id = ?", scenarioBRequestID, 2)
	assertSeedCount(t, db, &caseworkmodel.TodoItem{}, "source_type = ? AND source_id = ?", []any{notificationmodel.TodoSourceNotificationRequest, scenarioBRequestID}, 1)

	var first, second notificationmodel.NotificationAttempt
	if err := db.First(&first, scenarioBAttempt1ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&second, scenarioBAttempt2ID).Error; err != nil {
		t.Fatal(err)
	}
	if first.Status != notificationmodel.AttemptStatusFailed || second.Status != notificationmodel.AttemptStatusUnknown ||
		second.RetryOfAttemptID == nil || *second.RetryOfAttemptID != first.ID {
		t.Fatalf("scenario B attempts differ: first=%+v second=%+v", first, second)
	}
	taskIDs := make([]uint, 0, 5)
	var tasks []pathmodel.TaskInstance
	if err := db.Where("plan_instance_id = ?", scenarioBPlanID).Order("sort ASC").Find(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	if fmt.Sprint(taskIDs) != "[24101 24102 24103 24104 24105]" {
		t.Fatalf("scenario B task IDs changed: %v", taskIDs)
	}
}

func seedNotificationPrerequisites(t *testing.T, db *gorm.DB) {
	t.Helper()
	teamID := uint(syntheticTeamAID)
	clients := []caremodel.CareClient{
		{
			GVA_MODEL: global.GVA_MODEL{ID: 20001}, DisplayCode: "FIXED-A", DisplayName: "固定测试用户甲",
			OrganizationID: syntheticOrgAID, TeamID: &teamID, Status: caremodel.ClientStatusActive,
			SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: syntheticTeamAID,
		},
		{
			GVA_MODEL: global.GVA_MODEL{ID: 20002}, DisplayCode: "FIXED-B", DisplayName: "固定测试用户乙",
			OrganizationID: syntheticOrgAID, TeamID: &teamID, Status: caremodel.ClientStatusActive,
			SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: syntheticTeamAID,
		},
	}
	if err := db.Create(&clients).Error; err != nil {
		t.Fatal(err)
	}
	fixed := carePathFixedTime()
	assignments := []caremodel.CareAssignment{
		{
			CareClientID: 20001, OrganizationID: syntheticOrgAID, TeamID: syntheticTeamAID,
			AssigneeID: syntheticStewardAID, RoleType: caremodel.AssignmentRoleCareSteward,
			ValidFrom: fixed.Add(-time.Hour), Reason: "固定责任关系", Synthetic: true, DeptId: syntheticTeamAID,
		},
		{
			CareClientID: 20002, OrganizationID: syntheticOrgAID, TeamID: syntheticTeamAID,
			AssigneeID: syntheticStewardA2ID, RoleType: caremodel.AssignmentRoleCareSteward,
			ValidFrom: fixed.Add(-time.Hour), Reason: "固定责任关系", Synthetic: true, DeptId: syntheticTeamAID,
		},
	}
	if err := db.Create(&assignments).Error; err != nil {
		t.Fatal(err)
	}
	path := pathmodel.PathDefinitionVersion{
		GVA_MODEL: global.GVA_MODEL{ID: syntheticPathVersionID},
		Code:      "OSA", Version: "1.0.0-test", Title: "固定流程路径", Purpose: "验证软件流程",
		Status: pathmodel.LifecyclePublished, UsageScope: pathmodel.UsageScopeTestOnly,
		Synthetic: true, ReviewType: pathmodel.ReviewTypeEngineering,
		DefinitionHash: "fixed-path-hash", RowVersion: 1,
	}
	template := pathmodel.PlanTemplateVersion{
		GVA_MODEL:               global.GVA_MODEL{ID: syntheticPlanTemplateID},
		PathDefinitionVersionID: path.ID, Code: "FIXED-D1-D5", Version: "1.0.0-test",
		Title: "固定 D1-D5 流程", Purpose: "验证软件流程",
		Status: pathmodel.LifecyclePublished, UsageScope: pathmodel.UsageScopeTestOnly,
		Synthetic: true, ReviewType: pathmodel.ReviewTypeEngineering,
		AnchorDefinition:        pathmodel.AnchorFirstValidSyntheticDeviceUse,
		LateSubmissionPolicy:    pathmodel.LateSubmissionDeny,
		PauseStrategy:           pathmodel.PauseStrategyKeepWindows,
		DefinitionSchemaVersion: "v1", DefinitionHash: "fixed-plan-hash", RowVersion: 1,
	}
	if err := db.Create(&path).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	definitions := make([]pathmodel.PlanTaskDefinition, 0, 5)
	for i := 0; i < 5; i++ {
		definitions = append(definitions, pathmodel.PlanTaskDefinition{
			GVA_MODEL:             global.GVA_MODEL{ID: uint(syntheticTaskDefinitionD1ID + i)},
			PlanTemplateVersionID: template.ID, DayCode: fmt.Sprintf("D%d", i+1),
			Title: fmt.Sprintf("D%d 固定流程任务", i+1), Sort: i + 1,
			ExecutionRole:           pathmodel.ExecutionRoleCareClient,
			OpenOffsetSeconds:       int64(i) * int64(24*time.Hour/time.Second),
			DueOffsetSeconds:        int64(i)*int64(24*time.Hour/time.Second) + int64(11*time.Hour/time.Second),
			BoundRuleVersionIDsJSON: datatypes.JSON([]byte("[]")),
			NotificationPolicy:      pathmodel.NotificationPolicyDisabled,
		})
	}
	if err := db.Create(&definitions).Error; err != nil {
		t.Fatal(err)
	}
	taskA := pathmodel.TaskInstance{
		GVA_MODEL:      global.GVA_MODEL{ID: syntheticTaskInstanceD1ID},
		PlanInstanceID: 9603, CareClientID: 20001, TaskDefinitionID: syntheticTaskDefinitionD1ID,
		DayCode: "D1", Title: "D1 固定流程任务", Sort: 1,
		ExecutionRole: pathmodel.ExecutionRoleCareClient, ExecutionStatus: pathmodel.ExecutionOpen,
		ReviewStatus: pathmodel.ReviewNotRequired, OpenAt: fixed, DueAt: fixed.Add(11 * time.Hour),
		BoundRuleVersionIDsJSON: datatypes.JSON([]byte("[]")),
		LateSubmissionPolicy:    pathmodel.LateSubmissionDeny,
		NotificationPolicy:      pathmodel.NotificationPolicyDisabled,
		Version:                 1, Synthetic: true, DeptId: syntheticTeamAID,
	}
	if err := db.Create(&taskA).Error; err != nil {
		t.Fatal(err)
	}
}
