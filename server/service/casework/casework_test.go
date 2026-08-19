package casework

import (
	"context"
	"errors"
	"testing"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

type caseWorkFixture struct {
	db      *gorm.DB
	service *CaseWorkService
	ctx     context.Context
	now     time.Time
	caseRow caseworkmodel.AttentionCase
}

func TestCloseRequiresHandlingResult(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleClinician)

	_, err := fixture.service.Close(fixture.ctx, fixture.caseRow.ID, "close-without-result", caseworkreq.CloseCase{
		ExpectedVersion: fixture.caseRow.Version,
		CloseReason:     "流程验证已结束",
	})
	if caseDomainCode(err) != caseworkmodel.CodeHandlingResultRequired {
		t.Fatalf("close without handling result should return %d, got %v", caseworkmodel.CodeHandlingResultRequired, err)
	}
}

func TestAcknowledgeIsIdempotent(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	request := caseworkreq.AcknowledgeCase{ExpectedVersion: fixture.caseRow.Version, Result: "已确认进入人工跟进"}

	first, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-key", request)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-key", request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != caseworkmodel.CaseStatusAcknowledged || first.Version != 2 || first.ActionID == 0 {
		t.Fatalf("unexpected acknowledge result: %+v", first)
	}
	if replayed.ResourceID != first.ResourceID || replayed.ActionID != first.ActionID || replayed.Status != first.Status ||
		replayed.Version != first.Version || !replayed.OccurredAt.Equal(first.OccurredAt) {
		t.Fatalf("acknowledge replay should return the original result: first=%+v replayed=%+v", first, replayed)
	}
}

func TestAcknowledgeRejectsChangedReplayAndStaleVersion(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	request := caseworkreq.AcknowledgeCase{ExpectedVersion: fixture.caseRow.Version, Result: "已确认进入人工跟进"}
	if _, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-key", request); err != nil {
		t.Fatal(err)
	}
	changed := request
	changed.Result = "改用另一确认结果"
	if _, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-key", changed); caseDomainCode(err) != caseworkmodel.CodeIdempotencyConflict {
		t.Fatalf("changed replay should be rejected, got %v", err)
	}
	if _, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "second-ack-key", request); caseDomainCode(err) != caseworkmodel.CodeVersionConflict {
		t.Fatalf("stale expected version should be rejected, got %v", err)
	}
}

func TestStewardCanContactButCannotWriteProfessionalHandling(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	acknowledged, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-key", caseworkreq.AcknowledgeCase{
		ExpectedVersion: 1,
		Result:          "已确认进入人工跟进",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.service.RecordHandling(fixture.ctx, fixture.caseRow.ID, "professional-key", caseworkreq.HandlingRecord{
		ExpectedVersion: acknowledged.Version,
		ActionType:      caseworkmodel.CaseActionHandling,
		Result:          "尝试写入专业处置",
		NextStatus:      caseworkmodel.CaseStatusHandling,
	})
	if caseDomainCode(err) != caseworkmodel.CodeCaseResponsibilityRequired {
		t.Fatalf("steward professional handling should be denied, got %v", err)
	}

	contact, err := fixture.service.RecordHandling(fixture.ctx, fixture.caseRow.ID, "contact-key", caseworkreq.HandlingRecord{
		ExpectedVersion: acknowledged.Version,
		ActionType:      caseworkmodel.CaseActionContact,
		Result:          "已完成流程联系",
		NextStatus:      caseworkmodel.CaseStatusHandling,
	})
	if err != nil {
		t.Fatal(err)
	}
	if contact.Status != caseworkmodel.CaseStatusHandling || contact.Version != 3 {
		t.Fatalf("unexpected contact result: %+v", contact)
	}
}

func TestStewardWaitingCollaborationAutomaticallyTransfersToClinician(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	seedCaseActor(t, fixture.db, fixture.now, 43, 503, fixture.caseRow.CareClientID, caremodel.AuthorityRoleClinician)

	acknowledged, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-auto-transfer", caseworkreq.AcknowledgeCase{
		ExpectedVersion: fixture.caseRow.Version,
		Result:          "已确认并准备联系",
	})
	if err != nil {
		t.Fatal(err)
	}

	transferred, err := fixture.service.RecordHandling(fixture.ctx, fixture.caseRow.ID, "contact-auto-transfer", caseworkreq.HandlingRecord{
		ExpectedVersion: acknowledged.Version,
		ActionType:      caseworkmodel.CaseActionContact,
		Result:          "已完成联系",
		NextAction:      "请责任医护继续处理",
		NextStatus:      caseworkmodel.CaseStatusWaitingCollaboration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if transferred.Status != caseworkmodel.CaseStatusWaitingCollaboration || transferred.Version != 4 {
		t.Fatalf("contact should finish with an automatic clinician transfer: %+v", transferred)
	}

	seedCtx := datascope.WithSystem(context.Background())
	var stored caseworkmodel.AttentionCase
	if err = fixture.db.WithContext(seedCtx).First(&stored, fixture.caseRow.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeID == nil || *stored.AssigneeID != 43 || stored.AssigneeRole != caremodel.AssignmentRoleClinician {
		t.Fatalf("automatic transfer did not assign the current clinician: %+v", stored)
	}

	var todos []caseworkmodel.TodoItem
	if err = fixture.db.WithContext(seedCtx).Where("source_id = ?", fixture.caseRow.ID).Order("id ASC").Find(&todos).Error; err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 || todos[0].Status != caseworkmodel.TodoStatusSuperseded ||
		todos[1].Status != caseworkmodel.TodoStatusOpen || todos[1].AssigneeID != 43 || todos[1].ActiveSlot == nil {
		t.Fatalf("automatic transfer must replace the steward todo with a clinician todo: %+v", todos)
	}

	var actions []caseworkmodel.CaseAction
	if err = fixture.db.WithContext(seedCtx).Where("attention_case_id = ?", fixture.caseRow.ID).Order("id ASC").Find(&actions).Error; err != nil {
		t.Fatal(err)
	}
	if len(actions) != 3 || actions[2].ActionType != caseworkmodel.CaseActionEscalate ||
		actions[2].Source != caseworkmodel.ActionSourceSystem || actions[2].TargetAssigneeID == nil || *actions[2].TargetAssigneeID != 43 {
		t.Fatalf("automatic transfer must retain a separate system action: %+v", actions)
	}
}

func TestStewardAutomaticTransferRollsBackWithoutClinician(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	acknowledged, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-auto-transfer-without-clinician", caseworkreq.AcknowledgeCase{
		ExpectedVersion: fixture.caseRow.Version,
		Result:          "已确认并准备联系",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.service.RecordHandling(fixture.ctx, fixture.caseRow.ID, "contact-auto-transfer-without-clinician", caseworkreq.HandlingRecord{
		ExpectedVersion: acknowledged.Version,
		ActionType:      caseworkmodel.CaseActionContact,
		Result:          "已完成联系",
		NextStatus:      caseworkmodel.CaseStatusWaitingCollaboration,
	})
	if caseDomainCode(err) != caseworkmodel.CodeCaseResponsibilityRequired {
		t.Fatalf("automatic transfer without a clinician should be rejected, got %v", err)
	}

	seedCtx := datascope.WithSystem(context.Background())
	var stored caseworkmodel.AttentionCase
	if err = fixture.db.WithContext(seedCtx).First(&stored, fixture.caseRow.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != caseworkmodel.CaseStatusAcknowledged || stored.Version != acknowledged.Version ||
		stored.AssigneeID == nil || *stored.AssigneeID != 42 {
		t.Fatalf("failed automatic transfer must leave the acknowledged case unchanged: %+v", stored)
	}

	var actionCount int64
	if err = fixture.db.WithContext(seedCtx).Model(&caseworkmodel.CaseAction{}).
		Where("attention_case_id = ?", fixture.caseRow.ID).Count(&actionCount).Error; err != nil {
		t.Fatal(err)
	}
	if actionCount != 1 {
		t.Fatalf("failed automatic transfer must roll back the contact action, got %d actions", actionCount)
	}

	var todos []caseworkmodel.TodoItem
	if err = fixture.db.WithContext(seedCtx).Where("source_id = ?", fixture.caseRow.ID).Find(&todos).Error; err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].Status != caseworkmodel.TodoStatusOpen || todos[0].AssigneeID != 42 || todos[0].ActiveSlot == nil {
		t.Fatalf("failed automatic transfer must retain the steward todo: %+v", todos)
	}
}

func TestEscalatedClinicianCanResolveCloseAndReopen(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	clinicianCtx := seedCaseActor(t, fixture.db, fixture.now, 43, 503, fixture.caseRow.CareClientID, caremodel.AuthorityRoleClinician)
	supervisorCtx := seedCaseActor(t, fixture.db, fixture.now, 44, 504, fixture.caseRow.CareClientID, caremodel.AuthorityRoleSupervisor)

	acknowledged, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-key", caseworkreq.AcknowledgeCase{
		ExpectedVersion: 1,
		Result:          "已确认进入人工跟进",
	})
	if err != nil {
		t.Fatal(err)
	}
	contact, err := fixture.service.RecordHandling(fixture.ctx, fixture.caseRow.ID, "contact-key", caseworkreq.HandlingRecord{
		ExpectedVersion: acknowledged.Version,
		ActionType:      caseworkmodel.CaseActionContact,
		Result:          "已完成流程联系",
		NextStatus:      caseworkmodel.CaseStatusHandling,
	})
	if err != nil {
		t.Fatal(err)
	}
	escalated, err := fixture.service.Escalate(fixture.ctx, fixture.caseRow.ID, "escalate-key", caseworkreq.EscalateCase{
		ExpectedVersion:  contact.Version,
		TargetAssigneeID: 43,
		Reason:           "需要医护继续完成流程处理",
	})
	if err != nil {
		t.Fatal(err)
	}
	if escalated.Status != caseworkmodel.CaseStatusWaitingCollaboration || escalated.Version != 4 {
		t.Fatalf("unexpected escalation result: %+v", escalated)
	}

	resolved, err := fixture.service.RecordHandling(clinicianCtx, fixture.caseRow.ID, "resolve-key", caseworkreq.HandlingRecord{
		ExpectedVersion: escalated.Version,
		ActionType:      caseworkmodel.CaseActionHandling,
		Result:          "已完成流程处理",
		NextStatus:      caseworkmodel.CaseStatusResolved,
	})
	if err != nil {
		t.Fatal(err)
	}
	closed, err := fixture.service.Close(clinicianCtx, fixture.caseRow.ID, "close-key", caseworkreq.CloseCase{
		ExpectedVersion: resolved.Version,
		CloseReason:     "流程处理已完成并留痕",
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != caseworkmodel.CaseStatusClosed || closed.Version != 6 {
		t.Fatalf("unexpected close result: %+v", closed)
	}
	reopened, err := fixture.service.Reopen(supervisorCtx, fixture.caseRow.ID, "reopen-key", caseworkreq.ReopenCase{
		ExpectedVersion: closed.Version,
		Reason:          "需要重新完成一轮流程处理",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Status != caseworkmodel.CaseStatusHandling || reopened.Version != 7 {
		t.Fatalf("unexpected reopen result: %+v", reopened)
	}
	seedCtx := datascope.WithSystem(context.Background())
	var stored caseworkmodel.AttentionCase
	if err = fixture.db.WithContext(seedCtx).First(&stored, fixture.caseRow.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.HandlingResult != "" || stored.CloseReason != "" || stored.ResolvedAt != nil || stored.ClosedAt != nil {
		t.Fatalf("reopen must start a new handling round without deleting history: %+v", stored)
	}
	var todos []caseworkmodel.TodoItem
	if err = fixture.db.WithContext(seedCtx).Where("source_id = ?", fixture.caseRow.ID).Order("id ASC").Find(&todos).Error; err != nil {
		t.Fatal(err)
	}
	if len(todos) != 3 || todos[0].Status != caseworkmodel.TodoStatusSuperseded ||
		todos[1].Status != caseworkmodel.TodoStatusCompleted || todos[2].Status != caseworkmodel.TodoStatusOpen || todos[2].ActiveSlot == nil {
		t.Fatalf("escalate, close and reopen must retain todo history with one active row: %+v", todos)
	}
	var actionCount int64
	if err = fixture.db.WithContext(seedCtx).Model(&caseworkmodel.CaseAction{}).Where("attention_case_id = ?", fixture.caseRow.ID).Count(&actionCount).Error; err != nil {
		t.Fatal(err)
	}
	if actionCount != 7 {
		t.Fatalf("all state changes must remain append-only, actions=%d", actionCount)
	}
	_, err = fixture.service.Close(clinicianCtx, fixture.caseRow.ID, "close-after-reopen", caseworkreq.CloseCase{
		ExpectedVersion: reopened.Version,
		CloseReason:     "尝试沿用上轮结果关闭",
	})
	if caseDomainCode(err) != caseworkmodel.CodeHandlingResultRequired {
		t.Fatalf("reopened case should require a new handling result, got %v", err)
	}
}

func TestEscalatedClinicianCanRequestSupervisorReviewAndTodoRemainsOpen(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	clinicianCtx := seedCaseActor(t, fixture.db, fixture.now, 43, 503, fixture.caseRow.CareClientID, caremodel.AuthorityRoleClinician)

	acknowledged, err := fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "ack-review-key", caseworkreq.AcknowledgeCase{
		ExpectedVersion: 1,
		Result:          "已确认进入人工跟进",
	})
	if err != nil {
		t.Fatal(err)
	}
	contact, err := fixture.service.RecordHandling(fixture.ctx, fixture.caseRow.ID, "contact-review-key", caseworkreq.HandlingRecord{
		ExpectedVersion: acknowledged.Version,
		ActionType:      caseworkmodel.CaseActionContact,
		Result:          "已完成流程联系",
		NextStatus:      caseworkmodel.CaseStatusHandling,
	})
	if err != nil {
		t.Fatal(err)
	}
	escalated, err := fixture.service.Escalate(fixture.ctx, fixture.caseRow.ID, "escalate-review-key", caseworkreq.EscalateCase{
		ExpectedVersion:  contact.Version,
		TargetAssigneeID: 43,
		Reason:           "请责任医护继续处理",
	})
	if err != nil {
		t.Fatal(err)
	}
	reviewRequested, err := fixture.service.RecordHandling(clinicianCtx, fixture.caseRow.ID, "request-review-key", caseworkreq.HandlingRecord{
		ExpectedVersion: escalated.Version,
		ActionType:      caseworkmodel.CaseActionHandling,
		Result:          "已记录流程处理结果",
		NextAction:      "请求上级复核流程记录",
		NextStatus:      caseworkmodel.CaseStatusWaitingSupervisor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reviewRequested.Status != caseworkmodel.CaseStatusWaitingSupervisor || reviewRequested.Version != 5 {
		t.Fatalf("unexpected review request result: %+v", reviewRequested)
	}
	_, err = fixture.service.Escalate(clinicianCtx, fixture.caseRow.ID, "self-escalate-review-key", caseworkreq.EscalateCase{
		ExpectedVersion:  reviewRequested.Version,
		TargetAssigneeID: 43,
		Reason:           "重复转交给当前责任医护",
	})
	if caseDomainCode(err) != caseworkmodel.CodeCaseTransitionDenied {
		t.Fatalf("self escalation should preserve supervisor review with %d, got %v", caseworkmodel.CodeCaseTransitionDenied, err)
	}
	var storedCase caseworkmodel.AttentionCase
	if err = fixture.db.WithContext(datascope.WithSystem(context.Background())).First(&storedCase, fixture.caseRow.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedCase.Status != caseworkmodel.CaseStatusWaitingSupervisor || storedCase.Version != reviewRequested.Version {
		t.Fatalf("rejected self escalation changed the case: %+v", storedCase)
	}

	seedCtx := datascope.WithSystem(context.Background())
	var todo caseworkmodel.TodoItem
	if err = fixture.db.WithContext(seedCtx).
		Where("source_type = ? AND source_id = ? AND active_slot = ?", caseworkmodel.TodoSourceAttentionCase, fixture.caseRow.ID, caseworkmodel.TodoActiveSlot).
		First(&todo).Error; err != nil {
		t.Fatal(err)
	}
	if todo.Status != caseworkmodel.TodoStatusOpen || todo.AssigneeID != 43 {
		t.Fatalf("unresolved review request must retain the clinician todo: %+v", todo)
	}
}

func newCaseWorkFixture(t *testing.T, role string) caseWorkFixture {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareAssignment{}, &caremodel.CareAuthorityProfile{},
		&caseworkmodel.AttentionCase{}, &caseworkmodel.CaseAction{}, &caseworkmodel.TodoItem{}, &caseworkmodel.CommandReceipt{},
		&qmodel.QuestionnaireRuleHit{},
		&platformoutbox.Event{},
		testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 18, 11, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	const (
		actorID      = 42
		authorityID  = 502
		departmentID = 101
		careClientID = 201
	)
	seedCtx := datascope.WithSystem(context.Background())
	profile := caremodel.CareAuthorityProfile{AuthorityID: authorityID, RoleType: role, Synthetic: true, Active: true}
	if err := db.WithContext(seedCtx).Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	assignmentRole := caremodel.AssignmentRoleClinician
	if role == caremodel.AuthorityRoleCareSteward {
		assignmentRole = caremodel.AssignmentRoleCareSteward
	}
	assignment := caremodel.CareAssignment{
		CareClientID: careClientID, OrganizationID: 100, TeamID: departmentID,
		AssigneeID: actorID, RoleType: assignmentRole, ValidFrom: now.Add(-time.Hour),
		Reason: "流程验证责任关系", Synthetic: true, DeptId: departmentID,
	}
	if err := db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	assigneeID := uint(actorID)
	caseRow := caseworkmodel.AttentionCase{
		CareClientID: careClientID, TaskID: 301, SubmissionID: 401,
		SourceType: caseworkmodel.CaseSourceRuleHit, SourceRuleHitID: 501, DedupKey: "submission:401:rule:501",
		Status: caseworkmodel.CaseStatusPendingAck, AttentionLevel: "SYNTHETIC_ATTENTION",
		ReasonSummary: "流程选项请求人工关注，不表示健康判断。",
		AssigneeID:    &assigneeID, AssigneeRole: assignmentRole, OpenedAt: now, Version: 1,
		Synthetic: true, DeptId: departmentID,
	}
	if err := db.WithContext(seedCtx).Create(&caseRow).Error; err != nil {
		t.Fatal(err)
	}
	active := caseworkmodel.TodoActiveSlot
	todo := caseworkmodel.TodoItem{
		Category: caseworkmodel.TodoCategoryContentAttention, SourceType: caseworkmodel.TodoSourceAttentionCase,
		SourceID: caseRow.ID, ActiveSlot: &active, CareClientID: careClientID,
		AssigneeID: actorID, AssigneeRole: assignmentRole, Status: caseworkmodel.TodoStatusOpen,
		OpenedAt: now, Version: 1, Synthetic: true, DeptId: departmentID,
	}
	if err := db.WithContext(seedCtx).Create(&todo).Error; err != nil {
		t.Fatal(err)
	}
	ctx := datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: actorID, AuthorityID: authorityID, DeptID: departmentID,
		DeptIDs: []uint{departmentID}, VisibleDeptIDs: []uint{departmentID}, Scope: datascope.ScopeDept,
	})
	return caseWorkFixture{db: db, service: &CaseWorkService{DB: db, Now: func() time.Time { return now }}, ctx: ctx, now: now, caseRow: caseRow}
}

func caseDomainCode(err error) int {
	var domainErr *caseworkmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

func seedCaseActor(t *testing.T, db *gorm.DB, now time.Time, actorID, authorityID, careClientID uint, role string) context.Context {
	t.Helper()
	seedCtx := datascope.WithSystem(context.Background())
	profile := caremodel.CareAuthorityProfile{AuthorityID: authorityID, RoleType: role, Synthetic: true, Active: true}
	if err := db.WithContext(seedCtx).Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	assignmentRole := caremodel.AssignmentRoleCareSteward
	if role == caremodel.AuthorityRoleClinician {
		assignmentRole = caremodel.AssignmentRoleClinician
	}
	if role != caremodel.AuthorityRoleSupervisor {
		assignment := caremodel.CareAssignment{
			CareClientID: careClientID, OrganizationID: 100, TeamID: 101,
			AssigneeID: actorID, RoleType: assignmentRole, ValidFrom: now.Add(-time.Hour),
			Reason: "流程验证责任关系", Synthetic: true, DeptId: 101,
		}
		if err := db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
			t.Fatal(err)
		}
	}
	scope := datascope.ScopeDept
	visible := []uint{101}
	if role == caremodel.AuthorityRoleSupervisor {
		scope = datascope.ScopeDeptAndChild
	}
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: actorID, AuthorityID: authorityID, DeptID: 101,
		DeptIDs: []uint{101}, VisibleDeptIDs: visible, Scope: scope,
	})
}
