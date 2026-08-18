package notification

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestDemoAdapterEmitsAcceptedBeforeTerminal(t *testing.T) {
	now := time.Date(2026, time.August, 18, 8, 54, 0, 0, time.UTC)
	adapter := DemoNotificationAdapter{Outcome: notificationmodel.AttemptStatusFailed}
	receipts, err := adapter.Submit(context.Background(), SendCommand{
		NotificationAttemptID: 81, RequestedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		notificationmodel.AttemptStatusSubmittedToProvider,
		notificationmodel.AttemptStatusAccepted,
		notificationmodel.AttemptStatusFailed,
	}
	if len(receipts) != len(want) {
		t.Fatalf("receipt count = %d, want %d", len(receipts), len(want))
	}
	for i := range want {
		if receipts[i].Status != want[i] {
			t.Fatalf("receipt %d status = %s, want %s", i, receipts[i].Status, want[i])
		}
	}
	if receipts[1].Status == notificationmodel.AttemptStatusDelivered || receipts[2].FailureCode == "" {
		t.Fatalf("accepted and failed facts were collapsed: %+v", receipts)
	}
}

func TestDeliveryReceiptRejectsOutOfOrderTransition(t *testing.T) {
	fixture := newNotificationFixture(t)
	fixture.service.Adapter = noReceiptAdapter{}
	systemCtx := datascope.WithSystem(context.Background())
	pending, err := fixture.service.CreateInitial(systemCtx, fixture.task.ID, "invalid-order")
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.service.ApplyDeliveryReceipt(systemCtx, pending.ResourceID, DeliveryReceipt{
		EventKey:   "accepted-before-submitted",
		Status:     notificationmodel.AttemptStatusAccepted,
		OccurredAt: fixture.now.Add(time.Minute),
	})
	if notificationCode(err) != notificationmodel.CodeDeliveryEventInvalid {
		t.Fatalf("out-of-order receipt should be rejected, got %v", err)
	}
	var stored notificationmodel.NotificationAttempt
	if err = fixture.db.WithContext(systemCtx).First(&stored, pending.ResourceID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != notificationmodel.AttemptStatusPending || stored.Version != 1 {
		t.Fatalf("invalid receipt changed attempt: %+v", stored)
	}
	assertNotificationCount(t, fixture.db, &notificationmodel.DeliveryEvent{}, "notification_attempt_id = ?", stored.ID, 1)
}

func TestFailureResendAndUnknownPreserveTasksAndOneTodo(t *testing.T) {
	fixture := newNotificationFixture(t)
	systemCtx := datascope.WithSystem(context.Background())
	failed, err := fixture.service.CreateInitial(systemCtx, fixture.task.ID, "initial-a")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != notificationmodel.AttemptStatusFailed || failed.Version != 4 {
		t.Fatalf("unexpected failed action: %+v", failed)
	}

	var first notificationmodel.NotificationAttempt
	if err = fixture.db.WithContext(systemCtx).First(&first, failed.ResourceID).Error; err != nil {
		t.Fatal(err)
	}
	if first.AcceptedAt == nil || first.DeliveredAt != nil || first.FinalizedAt == nil {
		t.Fatalf("accepted must remain distinct from delivery: %+v", first)
	}
	assertNotificationCount(t, fixture.db, &notificationmodel.DeliveryEvent{}, "notification_attempt_id = ?", first.ID, 4)
	assertNotificationCount(t, fixture.db, &caseworkmodel.TodoItem{}, "source_type = ? AND source_id = ?", notificationmodel.TodoSourceNotificationRequest, first.NotificationRequestID, 1)

	_, err = fixture.service.Resend(fixture.stewardCtx, first.ID, "resend-stale", notificationreq.Resend{
		ExpectedVersion: first.Version - 1, Reason: "验证旧版本请求被拒绝",
	})
	if notificationCode(err) != notificationmodel.CodeVersionConflict {
		t.Fatalf("stale attempt version should conflict, got %v", err)
	}

	fixture.service.Adapter = DemoNotificationAdapter{Outcome: notificationmodel.AttemptStatusUnknown}
	resent, err := fixture.service.Resend(fixture.stewardCtx, first.ID, "resend-a", notificationreq.Resend{
		ExpectedVersion: first.Version, Reason: "首轮失败后改用人工确认流程",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resent.ResourceID == first.ID || resent.Status != notificationmodel.AttemptStatusUnknown || resent.Version != 4 {
		t.Fatalf("unexpected resend result: %+v", resent)
	}
	var second notificationmodel.NotificationAttempt
	if err = fixture.db.WithContext(systemCtx).First(&second, resent.ResourceID).Error; err != nil {
		t.Fatal(err)
	}
	if second.AttemptNo != 2 || second.RetryOfAttemptID == nil || *second.RetryOfAttemptID != first.ID {
		t.Fatalf("retry lineage missing: %+v", second)
	}
	var storedFirst notificationmodel.NotificationAttempt
	if err = fixture.db.WithContext(systemCtx).First(&storedFirst, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedFirst.Status != notificationmodel.AttemptStatusFailed || storedFirst.Version != first.Version {
		t.Fatalf("old attempt was overwritten: %+v", storedFirst)
	}
	assertNotificationCount(t, fixture.db, &caseworkmodel.TodoItem{}, "source_type = ? AND source_id = ? AND active_slot = ?", notificationmodel.TodoSourceNotificationRequest, first.NotificationRequestID, caseworkmodel.TodoActiveSlot, 1)

	replayed, err := fixture.service.Resend(fixture.stewardCtx, first.ID, "resend-a", notificationreq.Resend{
		ExpectedVersion: first.Version, Reason: "首轮失败后改用人工确认流程",
	})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ResourceID != resent.ResourceID || replayed.ActionID != resent.ActionID {
		t.Fatalf("idempotent replay differs: first=%+v replay=%+v", resent, replayed)
	}
	assertNotificationCount(t, fixture.db, &notificationmodel.NotificationAttempt{}, "notification_request_id = ?", first.NotificationRequestID, 2)

	_, err = fixture.service.Resend(fixture.stewardCtx, first.ID, "resend-a", notificationreq.Resend{
		ExpectedVersion: first.Version, Reason: "不同参数",
	})
	if notificationCode(err) != notificationmodel.CodeIdempotencyConflict {
		t.Fatalf("same key with another request should conflict, got %v", err)
	}

	duplicate := DeliveryReceipt{
		EventKey:    fmt.Sprintf("demo:%d:final", second.ID),
		Status:      notificationmodel.AttemptStatusUnknown,
		OccurredAt:  *second.FinalizedAt,
		FailureCode: notificationmodel.DemoUnknownCode,
	}
	if _, err = fixture.service.ApplyDeliveryReceipt(systemCtx, second.ID, duplicate); err != nil {
		t.Fatalf("duplicate receipt should be idempotent: %v", err)
	}
	assertNotificationCount(t, fixture.db, &caseworkmodel.TodoItem{}, "source_type = ? AND source_id = ?", notificationmodel.TodoSourceNotificationRequest, first.NotificationRequestID, 1)

	_, err = fixture.service.ApplyDeliveryReceipt(systemCtx, second.ID, DeliveryReceipt{
		EventKey: "new-terminal-event", Status: notificationmodel.AttemptStatusDelivered,
		OccurredAt: fixture.now.Add(time.Hour),
	})
	if notificationCode(err) != notificationmodel.CodeNotificationFinalized {
		t.Fatalf("final attempt accepted another event: %v", err)
	}

	var task pathmodel.TaskInstance
	if err = fixture.db.WithContext(systemCtx).First(&task, fixture.task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.ID != fixture.task.ID || task.ExecutionStatus != pathmodel.ExecutionOpen || task.Version != fixture.task.Version {
		t.Fatalf("notification lifecycle changed task facts: before=%+v after=%+v", fixture.task, task)
	}
}

func TestDeliveryScopeAndResendRole(t *testing.T) {
	fixture := newNotificationFixture(t)
	systemCtx := datascope.WithSystem(context.Background())
	if _, err := fixture.service.CreateInitial(systemCtx, fixture.task.ID, "scope-a"); err != nil {
		t.Fatal(err)
	}
	otherTask := fixture.seedTask(t, 202, 302, 99, 101)
	if _, err := fixture.service.CreateInitial(systemCtx, otherTask.ID, "scope-b"); err != nil {
		t.Fatal(err)
	}

	list, total, err := fixture.service.ListDeliveries(fixture.stewardCtx, notificationreq.DeliverySearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].CareClientID != fixture.task.CareClientID || len(list[0].Events) != 4 {
		t.Fatalf("steward responsibility scope leaked: total=%d list=%+v", total, list)
	}
	clinicianCtx := fixture.seedActor(t, 43, 503, caremodel.AuthorityRoleClinician, fixture.task.CareClientID, 101)
	if _, _, err = fixture.service.ListDeliveries(clinicianCtx, notificationreq.DeliverySearch{}); err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.Resend(clinicianCtx, list[0].ID, "clinician", notificationreq.Resend{
		ExpectedVersion: list[0].Version, Reason: "角色边界验证",
	})
	if notificationCode(err) != notificationmodel.CodeAccessScopeDenied {
		t.Fatalf("clinician resend should be denied, got %v", err)
	}
	supervisorCtx := fixture.seedActor(t, 44, 504, caremodel.AuthorityRoleSupervisor, 0, 101)
	supervisorList, supervisorTotal, err := fixture.service.ListDeliveries(supervisorCtx, notificationreq.DeliverySearch{})
	if err != nil {
		t.Fatal(err)
	}
	if supervisorTotal != 2 || len(supervisorList) != 2 {
		t.Fatalf("supervisor department projection = %d/%d, want 2", len(supervisorList), supervisorTotal)
	}
	otherStewardCtx := fixture.contextFor(99, 502, 101)
	_, err = fixture.service.Resend(otherStewardCtx, list[0].ID, "wrong-owner", notificationreq.Resend{
		ExpectedVersion: list[0].Version, Reason: "责任隔离验证",
	})
	if notificationCode(err) != notificationmodel.CodeAccessScopeDenied {
		t.Fatalf("unassigned steward resend should be denied, got %v", err)
	}
}

type notificationFixture struct {
	db         *gorm.DB
	service    *NotificationService
	now        time.Time
	task       pathmodel.TaskInstance
	stewardCtx context.Context
}

func newNotificationFixture(t *testing.T) notificationFixture {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.CareAuthorityProfile{},
		&pathmodel.TaskInstance{}, &caseworkmodel.TodoItem{},
		&notificationmodel.NotificationRequest{}, &notificationmodel.NotificationAttempt{}, &notificationmodel.DeliveryEvent{},
		&outbox.Event{}, testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 18, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	seedCtx := datascope.WithSystem(context.Background())
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: 501, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: 502, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
	}
	if err := db.WithContext(seedCtx).Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	client := caremodel.CareClient{
		GVA_MODEL: global.GVA_MODEL{ID: 201}, DisplayCode: "FIXED-CLIENT-201", DisplayName: "固定测试用户甲",
		OrganizationID: 100, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive,
		Synthetic: true, Version: 1, DeptId: 101,
	}
	if err := db.WithContext(seedCtx).Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	assignment := caremodel.CareAssignment{
		CareClientID: client.ID, OrganizationID: 100, TeamID: 101, AssigneeID: 42,
		RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: now.Add(-time.Hour),
		Reason: "固定测试责任关系", Synthetic: true, DeptId: 101,
	}
	if err := db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	task := testTask(301, client.ID, 101, now)
	if err := db.WithContext(seedCtx).Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	enabled := true
	service := &NotificationService{
		DB: db, Clock: FixedClock{Time: now},
		Adapter:                  DemoNotificationAdapter{Outcome: notificationmodel.AttemptStatusFailed},
		SyntheticFixturesEnabled: &enabled,
	}
	fixture := notificationFixture{db: db, service: service, now: now, task: task}
	fixture.stewardCtx = fixture.contextFor(42, 501, 101)
	return fixture
}

func (f notificationFixture) seedTask(t *testing.T, clientID, taskID, stewardID, deptID uint) pathmodel.TaskInstance {
	t.Helper()
	seedCtx := datascope.WithSystem(context.Background())
	client := caremodel.CareClient{
		GVA_MODEL: global.GVA_MODEL{ID: clientID}, DisplayCode: fmt.Sprintf("FIXED-CLIENT-%d", clientID),
		DisplayName: "固定测试用户乙", OrganizationID: 100, Status: caremodel.ClientStatusActive,
		SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: deptID,
	}
	if err := f.db.WithContext(seedCtx).Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	assignment := caremodel.CareAssignment{
		CareClientID: client.ID, OrganizationID: 100, TeamID: deptID, AssigneeID: stewardID,
		RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: f.now.Add(-time.Hour),
		Reason: "责任隔离验证", Synthetic: true, DeptId: deptID,
	}
	if err := f.db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	task := testTask(taskID, client.ID, deptID, f.now)
	if err := f.db.WithContext(seedCtx).Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	return task
}

func (f notificationFixture) seedActor(t *testing.T, userID, authorityID uint, role string, clientID, deptID uint) context.Context {
	t.Helper()
	seedCtx := datascope.WithSystem(context.Background())
	profile := caremodel.CareAuthorityProfile{AuthorityID: authorityID, RoleType: role, Synthetic: true, Active: true}
	if err := f.db.WithContext(seedCtx).Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	if role != caremodel.AuthorityRoleSupervisor {
		assignmentRole := caremodel.AssignmentRoleCareSteward
		if role == caremodel.AuthorityRoleClinician {
			assignmentRole = caremodel.AssignmentRoleClinician
		}
		assignment := caremodel.CareAssignment{
			CareClientID: clientID, OrganizationID: 100, TeamID: deptID, AssigneeID: userID,
			RoleType: assignmentRole, ValidFrom: f.now.Add(-time.Hour),
			Reason: "角色边界验证", Synthetic: true, DeptId: deptID,
		}
		if err := f.db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
			t.Fatal(err)
		}
	}
	return f.contextFor(userID, authorityID, deptID)
}

func (f notificationFixture) contextFor(userID, authorityID, deptID uint) context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: userID, AuthorityID: authorityID, DeptID: deptID,
		DeptIDs: []uint{deptID}, VisibleDeptIDs: []uint{deptID}, Scope: datascope.ScopeDept,
	})
}

func testTask(id, clientID, deptID uint, now time.Time) pathmodel.TaskInstance {
	return pathmodel.TaskInstance{
		GVA_MODEL: global.GVA_MODEL{ID: id}, PlanInstanceID: id + 1000,
		CareClientID: clientID, TaskDefinitionID: id + 2000, DayCode: "D1", Title: "固定流程任务", Sort: 1,
		ExecutionRole: pathmodel.ExecutionRoleCareClient, ExecutionStatus: pathmodel.ExecutionOpen,
		ReviewStatus: pathmodel.ReviewNotRequired, OpenAt: now, DueAt: now.Add(11 * time.Hour),
		BoundRuleVersionIDsJSON: datatypes.JSON([]byte("[]")),
		LateSubmissionPolicy:    pathmodel.LateSubmissionDeny,
		NotificationPolicy:      pathmodel.NotificationPolicyDisabled,
		Version:                 1, Synthetic: true, DeptId: deptID,
	}
}

func assertNotificationCount(t *testing.T, db *gorm.DB, model any, query string, args ...any) {
	t.Helper()
	want := args[len(args)-1].(int)
	args = args[:len(args)-1]
	var count int64
	if err := db.WithContext(datascope.WithSystem(context.Background())).Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(want) {
		t.Fatalf("count = %d, want %d", count, want)
	}
}

func notificationCode(err error) int {
	var domainErr *notificationmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

type noReceiptAdapter struct{}

func (noReceiptAdapter) Submit(context.Context, SendCommand) ([]DeliveryReceipt, error) {
	return nil, nil
}
