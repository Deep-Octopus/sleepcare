package supervision

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type supervisionFixture struct {
	db            *gorm.DB
	service       *SupervisionService
	now           time.Time
	systemCtx     context.Context
	supervisorCtx context.Context
	crossCtx      context.Context
	stewardCtx    context.Context
	attentionCase caseworkmodel.AttentionCase
	requestAction caseworkmodel.CaseAction
}

func TestDailySummaryRealtimeSnapshotRevisionAndScope(t *testing.T) {
	fixture := newSupervisionFixture(t)

	list, total, err := fixture.service.ListDailySummaries(fixture.supervisorCtx, supervisionreq.DailySummarySearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].SummaryType != supervisionmodel.SummaryTypeRealtimePreview || list[0].ID != 0 {
		t.Fatalf("unexpected realtime page: total=%d list=%+v", total, list)
	}
	assertSummaryMetrics(t, list[0].ServedClients, list[0].DueTasks, list[0].SubmittedTasks,
		list[0].DeliveryIssues, list[0].OpenAttentionCases, list[0].ResolvedAttentionCases, list[0].ReviewRequired,
		1, 2, 1, 1, 1, 1, 1)

	businessDate := time.Date(2026, time.August, 18, 0, 0, 0, 0, summaryLocation)
	v1, err := fixture.service.GenerateSnapshot(fixture.systemCtx, 100, businessDate)
	if err != nil {
		t.Fatal(err)
	}
	if v1.Version == nil || *v1.Version != 1 || v1.SummaryType != supervisionmodel.SummaryTypeVersionedSnapshot || len(v1.FocusCases) != 1 {
		t.Fatalf("unexpected first snapshot: %+v", v1)
	}

	resolvedAt := fixture.now.Add(-time.Minute)
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&caseworkmodel.AttentionCase{}).
		Where("id = ?", fixture.attentionCase.ID).
		Updates(map[string]any{
			"status":          caseworkmodel.CaseStatusResolved,
			"resolved_at":     resolvedAt,
			"handling_result": "流程处理已记录",
			"version":         gorm.Expr("version + 1"),
		}).Error; err != nil {
		t.Fatal(err)
	}
	v2, err := fixture.service.GenerateSnapshot(fixture.systemCtx, 100, businessDate)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Version == nil || *v2.Version != 2 || v2.ReviewRequired != 0 || v2.ResolvedAttentionCases != 2 {
		t.Fatalf("unexpected corrected snapshot: %+v", v2)
	}

	storedV1, err := fixture.service.GetDailySummary(fixture.supervisorCtx, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedV1.Version == nil || *storedV1.Version != 1 || storedV1.ReviewRequired != 1 || storedV1.ResolvedAttentionCases != 1 {
		t.Fatalf("first snapshot must remain immutable: %+v", storedV1)
	}
	if _, err = fixture.service.GetDailySummary(fixture.crossCtx, v1.ID); domainCode(err) != supervisionmodel.CodeReviewScopeDenied {
		t.Fatalf("cross-organization snapshot must fail closed, got %v", err)
	}
}

func TestDailySummaryRealtimeIncludesFactsAtCurrentBusinessInstant(t *testing.T) {
	fixture := newSupervisionFixture(t)
	var client caremodel.CareClient
	if err := fixture.db.WithContext(fixture.systemCtx).
		Where("organization_id = ?", 100).
		Order("id ASC").
		First(&client).Error; err != nil {
		t.Fatal(err)
	}

	submittedAt := fixture.now
	task := supervisionTask(
		client.ID,
		101,
		90,
		fixture.now.Add(-time.Hour),
		fixture.now.Add(time.Hour),
		pathmodel.ExecutionSubmitted,
		&submittedAt,
	)
	if err := fixture.db.WithContext(fixture.systemCtx).Create(&task).Error; err != nil {
		t.Fatal(err)
	}

	clinicianID := uint(43)
	attentionCase := supervisionCase(
		client.ID,
		101,
		90,
		caseworkmodel.CaseStatusWaitingCollaboration,
		&clinicianID,
		fixture.now,
	)
	if err := fixture.db.WithContext(fixture.systemCtx).Create(&attentionCase).Error; err != nil {
		t.Fatal(err)
	}

	list, _, err := fixture.service.ListDailySummaries(
		fixture.supervisorCtx,
		supervisionreq.DailySummarySearch{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("unexpected realtime page: %+v", list)
	}
	assertSummaryMetrics(
		t,
		list[0].ServedClients,
		list[0].DueTasks,
		list[0].SubmittedTasks,
		list[0].DeliveryIssues,
		list[0].OpenAttentionCases,
		list[0].ResolvedAttentionCases,
		list[0].ReviewRequired,
		1,
		3,
		2,
		1,
		2,
		1,
		1,
	)
}

func TestSummaryBoundsAdvancesCurrentCutoffByPersistedPrecision(t *testing.T) {
	now := time.Date(2026, time.August, 18, 10, 0, 0, 0, summaryLocation)
	start, cutoff, err := summaryBounds(now, now)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(time.Date(2026, time.August, 18, 0, 0, 0, 0, summaryLocation)) {
		t.Fatalf("unexpected start: %s", start)
	}
	if cutoff.Sub(now) != time.Millisecond {
		t.Fatalf("cutoff=%s want=%s", cutoff, now.Add(time.Millisecond))
	}
}

func TestGuidanceIsIdempotentAppendOnlyAndKeepsTodoOpen(t *testing.T) {
	fixture := newSupervisionFixture(t)
	req := supervisionreq.Guidance{
		ExpectedVersion:       1,
		Guidance:              "请按既有流程完成复核并补充记录",
		ResponsibleAssigneeID: 43,
		DueAt:                 fixture.now.Add(2 * time.Hour),
	}

	result, err := fixture.service.AddGuidance(fixture.supervisorCtx, fixture.attentionCase.ID, "guide-key", req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != caseworkmodel.CaseStatusWaitingSupervisor || result.Version != 2 {
		t.Fatalf("unexpected guidance result: %+v", result)
	}
	replayed, err := fixture.service.AddGuidance(fixture.supervisorCtx, fixture.attentionCase.ID, "guide-key", req)
	if err != nil || replayed.ResourceID != result.ResourceID || replayed.ActionID != result.ActionID ||
		replayed.Status != result.Status || replayed.Version != result.Version || !replayed.OccurredAt.Equal(result.OccurredAt) {
		t.Fatalf("same command must replay its result: result=%+v err=%v", replayed, err)
	}

	var guidanceCount, actionCount, eventCount int64
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&supervisionmodel.SupervisorGuidance{}).Count(&guidanceCount).Error; err != nil {
		t.Fatal(err)
	}
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&caseworkmodel.CaseAction{}).
		Where("attention_case_id = ?", fixture.attentionCase.ID).Count(&actionCount).Error; err != nil {
		t.Fatal(err)
	}
	if err = fixture.db.WithContext(fixture.systemCtx).Model(&outbox.Event{}).
		Where("event_type = ?", caseworkmodel.EventSupervisorGuidanceAdded).Count(&eventCount).Error; err != nil {
		t.Fatal(err)
	}
	if guidanceCount != 1 || actionCount != 2 || eventCount != 1 {
		t.Fatalf("guidance must append exactly once: guidance=%d actions=%d events=%d", guidanceCount, actionCount, eventCount)
	}

	var stored caseworkmodel.AttentionCase
	if err = fixture.db.WithContext(fixture.systemCtx).First(&stored, fixture.attentionCase.ID).Error; err != nil {
		t.Fatal(err)
	}
	var todo caseworkmodel.TodoItem
	if err = fixture.db.WithContext(fixture.systemCtx).
		Where("source_id = ? AND active_slot = ?", fixture.attentionCase.ID, caseworkmodel.TodoActiveSlot).
		First(&todo).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != caseworkmodel.CaseStatusWaitingSupervisor || stored.AssigneeID == nil || *stored.AssigneeID != 43 ||
		todo.Status != caseworkmodel.TodoStatusOpen || todo.AssigneeID != 43 || todo.DueAt == nil {
		t.Fatalf("guidance must preserve the open responsibility chain: case=%+v todo=%+v", stored, todo)
	}

	reviews, total, err := fixture.service.ListReviews(fixture.supervisorCtx, supervisionreq.ReviewSearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(reviews) != 1 || reviews[0].Status != supervisionmodel.ReviewStatusGuided ||
		reviews[0].RequestedBy != fixture.requestAction.ActorID || !reviews[0].RequestedAt.Equal(fixture.requestAction.OccurredAt) {
		t.Fatalf("unexpected guided review projection: total=%d items=%+v", total, reviews)
	}

	changed := req
	changed.Guidance = "另一条流程安排"
	if _, err = fixture.service.AddGuidance(fixture.supervisorCtx, fixture.attentionCase.ID, "guide-key", changed); domainCode(err) != supervisionmodel.CodeIdempotencyConflict {
		t.Fatalf("changed request must conflict on the same key, got %v", err)
	}
	if _, err = fixture.service.AddGuidance(fixture.supervisorCtx, fixture.attentionCase.ID, "stale-key", req); domainCode(err) != supervisionmodel.CodeVersionConflict {
		t.Fatalf("stale case version must conflict, got %v", err)
	}

	nextRequest := supervisionRequestAction(fixture.attentionCase, 43, fixture.now.Add(time.Minute))
	nextRequest.CommandKeyDigest = "next-review"
	if err = fixture.db.WithContext(fixture.systemCtx).Create(&nextRequest).Error; err != nil {
		t.Fatal(err)
	}
	reviews, total, err = fixture.service.ListReviews(fixture.supervisorCtx, supervisionreq.ReviewSearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(reviews) != 1 || reviews[0].Status != supervisionmodel.ReviewStatusPending ||
		reviews[0].RequestedBy != nextRequest.ActorID || !reviews[0].RequestedAt.Equal(nextRequest.OccurredAt) {
		t.Fatalf("a later review request must start a new pending cycle: total=%d items=%+v", total, reviews)
	}
}

func TestInterventionAndNegativePermissionPaths(t *testing.T) {
	fixture := newSupervisionFixture(t)
	req := supervisionreq.Intervene{
		ExpectedVersion:       1,
		Result:                "已完成上级流程介入并明确后续责任",
		ResponsibleAssigneeID: 43,
		DueAt:                 fixture.now.Add(90 * time.Minute),
	}
	result, err := fixture.service.Intervene(fixture.supervisorCtx, fixture.attentionCase.ID, "intervene-key", req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != caseworkmodel.CaseStatusHandling || result.Version != 2 {
		t.Fatalf("unexpected intervention result: %+v", result)
	}
	reviews, _, err := fixture.service.ListReviews(fixture.supervisorCtx, supervisionreq.ReviewSearch{})
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 || reviews[0].Status != supervisionmodel.ReviewStatusIntervened {
		t.Fatalf("intervention should remain visible as the latest review fact: %+v", reviews)
	}

	other := newSupervisionFixture(t)
	if _, err = other.service.AddGuidance(other.stewardCtx, other.attentionCase.ID, "steward-key", supervisionreq.Guidance{
		ExpectedVersion: 1, Guidance: "无权操作", ResponsibleAssigneeID: 43, DueAt: other.now.Add(time.Hour),
	}); domainCode(err) != supervisionmodel.CodeReviewScopeDenied {
		t.Fatalf("non-supervisor must fail closed, got %v", err)
	}
	if _, err = other.service.AddGuidance(other.crossCtx, other.attentionCase.ID, "cross-key", supervisionreq.Guidance{
		ExpectedVersion: 1, Guidance: "跨范围操作", ResponsibleAssigneeID: 43, DueAt: other.now.Add(time.Hour),
	}); domainCode(err) != supervisionmodel.CodeReviewScopeDenied {
		t.Fatalf("cross-organization supervisor must fail closed, got %v", err)
	}
	if _, err = other.service.AddGuidance(other.supervisorCtx, other.attentionCase.ID, "target-key", supervisionreq.Guidance{
		ExpectedVersion: 1, Guidance: "责任人校验", ResponsibleAssigneeID: 999, DueAt: other.now.Add(time.Hour),
	}); domainCode(err) != supervisionmodel.CodeGuidanceResultRequired {
		t.Fatalf("invalid responsible assignee must fail, got %v", err)
	}
	if _, err = other.service.AddGuidance(other.supervisorCtx, other.attentionCase.ID, "empty-key", supervisionreq.Guidance{
		ExpectedVersion: 1, Guidance: " ", ResponsibleAssigneeID: 43, DueAt: other.now.Add(time.Hour),
	}); domainCode(err) != supervisionmodel.CodeGuidanceResultRequired {
		t.Fatalf("blank guidance must fail, got %v", err)
	}
}

func newSupervisionFixture(t *testing.T) supervisionFixture {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.CareOrgUnitProfile{}, &caremodel.CareAuthorityProfile{},
		&pathmodel.TaskInstance{}, &caseworkmodel.AttentionCase{}, &caseworkmodel.CaseAction{}, &caseworkmodel.TodoItem{},
		&caseworkmodel.Consultation{},
		&supervisionmodel.DailySummaryVersion{}, &supervisionmodel.SupervisorGuidance{}, &outbox.Event{},
		testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 18, 16, 0, 0, 0, summaryLocation)
	systemCtx := datascope.WithSystem(context.Background())
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: 501, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
		{AuthorityID: 502, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: 504, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
	}
	if err := db.WithContext(systemCtx).Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	units := []caremodel.CareOrgUnitProfile{
		{DepartmentID: 100, OrganizationID: 100, Code: "ORG-A", UnitType: caremodel.OrgUnitTypeOrganization, Synthetic: true, Active: true, DeptId: 100},
		{DepartmentID: 200, OrganizationID: 200, Code: "ORG-B", UnitType: caremodel.OrgUnitTypeOrganization, Synthetic: true, Active: true, DeptId: 200},
	}
	if err := db.WithContext(systemCtx).Create(&units).Error; err != nil {
		t.Fatal(err)
	}
	teamA, teamB := uint(101), uint(202)
	clients := []caremodel.CareClient{
		{DisplayCode: "SUP-001", DisplayName: "测试用户甲", OrganizationID: 100, TeamID: &teamA, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivityBasic, Synthetic: true, Version: 1, DeptId: 101},
		{DisplayCode: "SUP-002", DisplayName: "测试用户乙", OrganizationID: 200, TeamID: &teamB, Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivityBasic, Synthetic: true, Version: 1, DeptId: 202},
	}
	if err := db.WithContext(systemCtx).Create(&clients).Error; err != nil {
		t.Fatal(err)
	}
	assignments := []caremodel.CareAssignment{
		{CareClientID: clients[0].ID, OrganizationID: 100, TeamID: 101, AssigneeID: 43, RoleType: caremodel.AssignmentRoleClinician, ValidFrom: now.Add(-24 * time.Hour), Reason: "责任链验证", Synthetic: true, DeptId: 101},
		{CareClientID: clients[1].ID, OrganizationID: 200, TeamID: 202, AssigneeID: 53, RoleType: caremodel.AssignmentRoleClinician, ValidFrom: now.Add(-24 * time.Hour), Reason: "隔离验证", Synthetic: true, DeptId: 202},
	}
	if err := db.WithContext(systemCtx).Create(&assignments).Error; err != nil {
		t.Fatal(err)
	}
	submittedAt := now.Add(-2 * time.Hour)
	tasks := []pathmodel.TaskInstance{
		supervisionTask(clients[0].ID, 101, 1, now.Add(-6*time.Hour), now.Add(time.Hour), pathmodel.ExecutionSubmitted, &submittedAt),
		supervisionTask(clients[0].ID, 101, 2, now.Add(-4*time.Hour), now.Add(2*time.Hour), pathmodel.ExecutionOpen, nil),
		supervisionTask(clients[1].ID, 202, 3, now.Add(-4*time.Hour), now.Add(time.Hour), pathmodel.ExecutionOpen, nil),
	}
	if err := db.WithContext(systemCtx).Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	clinicianA, clinicianB := uint(43), uint(53)
	attentionCases := []caseworkmodel.AttentionCase{
		supervisionCase(clients[0].ID, 101, 1, caseworkmodel.CaseStatusWaitingSupervisor, &clinicianA, now.Add(-time.Hour)),
		supervisionCase(clients[0].ID, 101, 2, caseworkmodel.CaseStatusClosed, &clinicianA, now.Add(-3*time.Hour)),
		supervisionCase(clients[1].ID, 202, 3, caseworkmodel.CaseStatusWaitingSupervisor, &clinicianB, now.Add(-time.Hour)),
	}
	resolvedAt := now.Add(-40 * time.Minute)
	closedAt := now.Add(-20 * time.Minute)
	attentionCases[1].ResolvedAt = &resolvedAt
	attentionCases[1].ClosedAt = &closedAt
	attentionCases[1].HandlingResult = "流程处理已记录"
	attentionCases[1].CloseReason = "流程关闭条件已满足"
	if err := db.WithContext(systemCtx).Create(&attentionCases).Error; err != nil {
		t.Fatal(err)
	}
	requestActions := []caseworkmodel.CaseAction{
		supervisionRequestAction(attentionCases[0], 43, now.Add(-30*time.Minute)),
		supervisionRequestAction(attentionCases[2], 53, now.Add(-25*time.Minute)),
	}
	if err := db.WithContext(systemCtx).Create(&requestActions).Error; err != nil {
		t.Fatal(err)
	}
	active := caseworkmodel.TodoActiveSlot
	todos := []caseworkmodel.TodoItem{
		{Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase, SourceID: attentionCases[0].ID, ActiveSlot: &active, CareClientID: clients[0].ID, AssigneeID: 43, AssigneeRole: caremodel.AssignmentRoleClinician, Status: caseworkmodel.TodoStatusOpen, OpenedAt: now.Add(-time.Hour), Version: 1, Synthetic: true, DeptId: 101},
		{Category: caseworkmodel.TodoCategoryDeliveryIssue, SourceType: "NOTIFICATION_ATTEMPT", SourceID: 9001, ActiveSlot: &active, CareClientID: clients[0].ID, AssigneeID: 42, AssigneeRole: caremodel.AssignmentRoleCareSteward, Status: caseworkmodel.TodoStatusOpen, OpenedAt: now.Add(-2 * time.Hour), Version: 1, Synthetic: true, DeptId: 101},
		{Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase, SourceID: attentionCases[2].ID, ActiveSlot: &active, CareClientID: clients[1].ID, AssigneeID: 53, AssigneeRole: caremodel.AssignmentRoleClinician, Status: caseworkmodel.TodoStatusOpen, OpenedAt: now.Add(-time.Hour), Version: 1, Synthetic: true, DeptId: 202},
	}
	if err := db.WithContext(systemCtx).Create(&todos).Error; err != nil {
		t.Fatal(err)
	}

	return supervisionFixture{
		db:        db,
		service:   &SupervisionService{DB: db, Now: func() time.Time { return now }},
		now:       now,
		systemCtx: systemCtx,
		supervisorCtx: datascope.WithIdentity(context.Background(), &datascope.Identity{
			UserID: 50, AuthorityID: 501, DeptID: 100, DeptIDs: []uint{100}, VisibleDeptIDs: []uint{100, 101}, Scope: datascope.ScopeDeptAndChild,
		}),
		crossCtx: datascope.WithIdentity(context.Background(), &datascope.Identity{
			UserID: 60, AuthorityID: 504, DeptID: 200, DeptIDs: []uint{200}, VisibleDeptIDs: []uint{200, 202}, Scope: datascope.ScopeDeptAndChild,
		}),
		stewardCtx: datascope.WithIdentity(context.Background(), &datascope.Identity{
			UserID: 42, AuthorityID: 502, DeptID: 101, DeptIDs: []uint{101}, VisibleDeptIDs: []uint{101}, Scope: datascope.ScopeDept,
		}),
		attentionCase: attentionCases[0],
		requestAction: requestActions[0],
	}
}

func supervisionTask(careClientID, deptID, definitionID uint, openAt, dueAt time.Time, status string, submittedAt *time.Time) pathmodel.TaskInstance {
	return pathmodel.TaskInstance{
		PlanInstanceID:          definitionID,
		CareClientID:            careClientID,
		TaskDefinitionID:        definitionID,
		DayCode:                 "D1",
		Title:                   "流程任务",
		Sort:                    int(definitionID),
		ExecutionRole:           pathmodel.ExecutionRoleCareClient,
		ExecutionStatus:         status,
		ReviewStatus:            pathmodel.ReviewPending,
		ReviewRole:              caremodel.AuthorityRoleClinician,
		OpenAt:                  openAt,
		DueAt:                   dueAt,
		BoundRuleVersionIDsJSON: datatypes.JSON([]byte(`[]`)),
		LateSubmissionPolicy:    pathmodel.LateSubmissionDeny,
		NotificationPolicy:      pathmodel.NotificationPolicyDisabled,
		SubmittedAt:             submittedAt,
		Version:                 1,
		Synthetic:               true,
		DeptId:                  deptID,
	}
}

func supervisionCase(careClientID, deptID, sourceID uint, status string, assigneeID *uint, openedAt time.Time) caseworkmodel.AttentionCase {
	return caseworkmodel.AttentionCase{
		CareClientID:    careClientID,
		TaskID:          sourceID,
		SubmissionID:    sourceID,
		SourceType:      caseworkmodel.CaseSourceRuleHit,
		SourceRuleHitID: sourceID,
		DedupKey:        "supervision-case-" + time.Unix(int64(sourceID), 0).Format("150405"),
		Status:          status,
		AttentionLevel:  "TEST_ATTENTION",
		ReasonSummary:   "流程记录需要人工关注。",
		AssigneeID:      assigneeID,
		AssigneeRole:    caremodel.AssignmentRoleClinician,
		OpenedAt:        openedAt,
		Version:         1,
		Synthetic:       true,
		DeptId:          deptID,
	}
}

func supervisionRequestAction(attentionCase caseworkmodel.AttentionCase, actorID uint, occurredAt time.Time) caseworkmodel.CaseAction {
	return caseworkmodel.CaseAction{
		AttentionCaseID:  attentionCase.ID,
		ActionType:       caseworkmodel.CaseActionHandling,
		ActorID:          actorID,
		ActorRole:        caremodel.AuthorityRoleClinician,
		OrganizationID:   100,
		TeamID:           attentionCase.DeptId,
		Source:           caseworkmodel.ActionSourceStaff,
		Result:           "已记录流程处理结果",
		Reason:           "请求上级复核流程记录",
		FromStatus:       caseworkmodel.CaseStatusHandling,
		ToStatus:         caseworkmodel.CaseStatusWaitingSupervisor,
		OccurredAt:       occurredAt,
		CommandKeyDigest: "request-review",
		Synthetic:        true,
		DeptId:           attentionCase.DeptId,
	}
}

func assertSummaryMetrics(
	t *testing.T,
	served, due, submitted, delivery, openCases, resolved, review int64,
	wantServed, wantDue, wantSubmitted, wantDelivery, wantOpen, wantResolved, wantReview int64,
) {
	t.Helper()
	got := []int64{served, due, submitted, delivery, openCases, resolved, review}
	want := []int64{wantServed, wantDue, wantSubmitted, wantDelivery, wantOpen, wantResolved, wantReview}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected summary metrics: got=%v want=%v", got, want)
		}
	}
}

func domainCode(err error) int {
	var domainErr *supervisionmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}
