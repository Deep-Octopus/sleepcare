package casework

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestGetWorkbenchProjectsScopedCounts(t *testing.T) {
	db, service, now := newWorkbenchFixture(t)
	seedCtx := datascope.WithSystem(context.Background())
	stewardCtx := datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: 42, AuthorityID: 502, DeptID: 101,
		DeptIDs: []uint{101}, VisibleDeptIDs: []uint{101}, Scope: datascope.ScopeDept,
	})

	data, err := service.GetWorkbench(stewardCtx)
	if err != nil {
		t.Fatal(err)
	}
	if data.DueToday != 1 || data.WaitingClient != 2 || data.DeliveryIssues != 1 ||
		data.AttentionCases != 1 || data.AssignedToMe != 1 || data.ReviewRequired != 1 {
		t.Fatalf("unexpected workbench projection: %+v", data)
	}

	var closedCount int64
	if err = db.WithContext(seedCtx).Model(&caseworkmodel.AttentionCase{}).
		Where("status = ?", caseworkmodel.CaseStatusClosed).Count(&closedCount).Error; err != nil {
		t.Fatal(err)
	}
	if closedCount != 1 {
		t.Fatalf("fixture should retain one closed case, got %d", closedCount)
	}
	start, end := workbenchDayBounds(now.UTC())
	if start.Location().String() != "Asia/Shanghai" || end.Sub(start) != 24*time.Hour {
		t.Fatalf("unexpected workbench day bounds: %s - %s", start, end)
	}
}

func TestGetWorkbenchFailsClosedAndHonorsResponsibility(t *testing.T) {
	_, service, _ := newWorkbenchFixture(t)
	if _, err := service.GetWorkbench(context.Background()); caseDomainCode(err) != caremodel.CodeAccessScopeDenied {
		t.Fatalf("missing identity should fail closed, got %v", err)
	}

	unassignedCtx := datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: 77, AuthorityID: 502, DeptID: 101,
		DeptIDs: []uint{101}, VisibleDeptIDs: []uint{101}, Scope: datascope.ScopeDept,
	})
	data, err := service.GetWorkbench(unassignedCtx)
	if err != nil {
		t.Fatal(err)
	}
	if data != (caseworkres.WorkbenchData{}) {
		t.Fatalf("unassigned staff should see an empty projection: %+v", data)
	}

	supervisorCtx := datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: 88, AuthorityID: 503, DeptID: 100,
		DeptIDs: []uint{100}, VisibleDeptIDs: []uint{100, 101}, Scope: datascope.ScopeDeptAndChild,
	})
	data, err = service.GetWorkbench(supervisorCtx)
	if err != nil {
		t.Fatal(err)
	}
	if data.AttentionCases != 2 || data.ReviewRequired != 2 {
		t.Fatalf("supervisor projection should include visible department records only: %+v", data)
	}
}

func newWorkbenchFixture(t *testing.T) (*gorm.DB, *CaseWorkService, time.Time) {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.CareAuthorityProfile{},
		&pathmodel.TaskInstance{}, &caseworkmodel.AttentionCase{}, &caseworkmodel.TodoItem{},
		testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 18, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	seedCtx := datascope.WithSystem(context.Background())
	for _, profile := range []caremodel.CareAuthorityProfile{
		{AuthorityID: 502, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: 503, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
	} {
		if err := db.WithContext(seedCtx).Create(&profile).Error; err != nil {
			t.Fatal(err)
		}
	}
	clients := []caremodel.CareClient{
		{DisplayCode: "WB-001", DisplayName: "测试用户甲", OrganizationID: 100, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivityBasic, Synthetic: true, Version: 1, DeptId: 101},
		{DisplayCode: "WB-002", DisplayName: "测试用户乙", OrganizationID: 100, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivityBasic, Synthetic: true, Version: 1, DeptId: 101},
		{DisplayCode: "WB-003", DisplayName: "测试用户丙", OrganizationID: 200, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivityBasic, Synthetic: true, Version: 1, DeptId: 202},
	}
	if err := db.WithContext(seedCtx).Create(&clients).Error; err != nil {
		t.Fatal(err)
	}
	assignments := []caremodel.CareAssignment{
		{CareClientID: clients[0].ID, OrganizationID: 100, TeamID: 101, AssigneeID: 42, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: now.Add(-time.Hour), Reason: "工作台范围验证", Synthetic: true, DeptId: 101},
		{CareClientID: clients[1].ID, OrganizationID: 100, TeamID: 101, AssigneeID: 99, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: now.Add(-time.Hour), Reason: "责任关系隔离验证", Synthetic: true, DeptId: 101},
		{CareClientID: clients[2].ID, OrganizationID: 200, TeamID: 202, AssigneeID: 42, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: now.Add(-time.Hour), Reason: "机构范围隔离验证", Synthetic: true, DeptId: 202},
	}
	if err := db.WithContext(seedCtx).Create(&assignments).Error; err != nil {
		t.Fatal(err)
	}
	tasks := []pathmodel.TaskInstance{
		workbenchTask(1, clients[0].ID, 101, now.Add(-time.Hour), now.Add(6*time.Hour), pathmodel.ExecutionOpen, pathmodel.ReviewNotReady),
		workbenchTask(2, clients[0].ID, 101, now.Add(-time.Hour), now.Add(30*time.Hour), pathmodel.ExecutionInProgress, pathmodel.ReviewPending),
		workbenchTask(3, clients[0].ID, 101, now.Add(-time.Hour), now.Add(7*time.Hour), pathmodel.ExecutionSubmitted, pathmodel.ReviewNotRequired),
		workbenchTask(4, clients[1].ID, 101, now.Add(-time.Hour), now.Add(5*time.Hour), pathmodel.ExecutionOpen, pathmodel.ReviewPending),
		workbenchTask(5, clients[2].ID, 202, now.Add(-time.Hour), now.Add(4*time.Hour), pathmodel.ExecutionOpen, pathmodel.ReviewPending),
	}
	if err := db.WithContext(seedCtx).Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	assignee42 := uint(42)
	assignee99 := uint(99)
	cases := []caseworkmodel.AttentionCase{
		workbenchCase(clients[0].ID, 101, 1, caseworkmodel.CaseStatusHandling, &assignee42, now),
		workbenchCase(clients[0].ID, 101, 2, caseworkmodel.CaseStatusClosed, &assignee42, now),
		workbenchCase(clients[1].ID, 101, 3, caseworkmodel.CaseStatusHandling, &assignee99, now),
		workbenchCase(clients[2].ID, 202, 4, caseworkmodel.CaseStatusHandling, &assignee42, now),
	}
	if err := db.WithContext(seedCtx).Create(&cases).Error; err != nil {
		t.Fatal(err)
	}
	active := caseworkmodel.TodoActiveSlot
	todos := []caseworkmodel.TodoItem{
		{Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase, SourceID: cases[0].ID, ActiveSlot: &active, CareClientID: clients[0].ID, AssigneeID: 42, AssigneeRole: caremodel.AssignmentRoleCareSteward, Status: caseworkmodel.TodoStatusOpen, OpenedAt: now, Version: 1, Synthetic: true, DeptId: 101},
		{Category: caseworkmodel.TodoCategoryDeliveryIssue, SourceType: "NOTIFICATION_ATTEMPT", SourceID: 1001, ActiveSlot: &active, CareClientID: clients[0].ID, AssigneeID: 99, AssigneeRole: caremodel.AssignmentRoleCareSteward, Status: caseworkmodel.TodoStatusOpen, OpenedAt: now, Version: 1, Synthetic: true, DeptId: 101},
		{Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase, SourceID: cases[2].ID, ActiveSlot: &active, CareClientID: clients[1].ID, AssigneeID: 42, AssigneeRole: caremodel.AssignmentRoleCareSteward, Status: caseworkmodel.TodoStatusOpen, OpenedAt: now, Version: 1, Synthetic: true, DeptId: 101},
	}
	if err := db.WithContext(seedCtx).Create(&todos).Error; err != nil {
		t.Fatal(err)
	}
	return db, &CaseWorkService{DB: db, Now: func() time.Time { return now }}, now
}

func workbenchTask(id, careClientID, deptID uint, openAt, dueAt time.Time, executionStatus, reviewStatus string) pathmodel.TaskInstance {
	return pathmodel.TaskInstance{
		PlanInstanceID: id, CareClientID: careClientID, TaskDefinitionID: id,
		DayCode: "D1", Title: "流程任务", Sort: int(id), ExecutionRole: pathmodel.ExecutionRoleCareClient,
		ExecutionStatus: executionStatus, ReviewStatus: reviewStatus, ReviewRole: caremodel.AuthorityRoleClinician,
		OpenAt: openAt, DueAt: dueAt, BoundRuleVersionIDsJSON: datatypes.JSON([]byte(`[]`)),
		LateSubmissionPolicy: pathmodel.LateSubmissionDeny, NotificationPolicy: pathmodel.NotificationPolicyDisabled,
		Version: 1, Synthetic: true, DeptId: deptID,
	}
}

func workbenchCase(careClientID, deptID, sourceID uint, status string, assigneeID *uint, now time.Time) caseworkmodel.AttentionCase {
	return caseworkmodel.AttentionCase{
		CareClientID: careClientID, TaskID: sourceID, SubmissionID: sourceID,
		SourceType: caseworkmodel.CaseSourceRuleHit, SourceRuleHitID: sourceID, DedupKey: "workbench-case-" + time.Unix(int64(sourceID), 0).Format("150405"),
		Status: status, AttentionLevel: "TEST_ATTENTION", ReasonSummary: "流程记录需要人工关注。",
		AssigneeID: assigneeID, AssigneeRole: caremodel.AssignmentRoleCareSteward,
		OpenedAt: now, Version: 1, Synthetic: true, DeptId: deptID,
	}
}
