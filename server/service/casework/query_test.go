package casework

import (
	"context"
	"testing"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
)

func TestListAndGetAttentionCasesWithinResponsibilityScope(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	seedCaseRuleHit(t, fixture)

	list, total, err := fixture.service.List(fixture.ctx, caseworkreq.AttentionCaseSearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != fixture.caseRow.ID {
		t.Fatalf("unexpected attention case list: total=%d list=%+v", total, list)
	}
	filtered, filteredTotal, err := fixture.service.List(fixture.ctx, caseworkreq.AttentionCaseSearch{AssigneeID: 999})
	if err != nil || filteredTotal != 0 || len(filtered) != 0 {
		t.Fatalf("assignee filter must only shrink the authorized result: total=%d list=%+v err=%v", filteredTotal, filtered, err)
	}

	if _, err = fixture.service.Acknowledge(fixture.ctx, fixture.caseRow.ID, "query-ack-key", caseworkreq.AcknowledgeCase{
		ExpectedVersion: fixture.caseRow.Version,
		Result:          "已确认进入人工跟进",
	}); err != nil {
		t.Fatal(err)
	}
	detail, err := fixture.service.Get(fixture.ctx, fixture.caseRow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Status != caseworkmodel.CaseStatusAcknowledged || len(detail.RuleHits) != 1 || len(detail.Actions) != 1 {
		t.Fatalf("unexpected attention case detail: %+v", detail)
	}
	if detail.RuleHits[0].ID != fixture.caseRow.SourceRuleHitID || detail.Actions[0].ActionType != caseworkmodel.CaseActionAcknowledge {
		t.Fatalf("detail history is incomplete: %+v", detail)
	}
}

func TestAttentionCaseQueryFailsClosedOutsideScope(t *testing.T) {
	fixture := newCaseWorkFixture(t, caremodel.AuthorityRoleCareSteward)
	seedCaseRuleHit(t, fixture)
	otherCtx := seedCaseActor(t, fixture.db, fixture.now, 45, 505, 999, caremodel.AuthorityRoleCareSteward)

	list, total, err := fixture.service.List(otherCtx, caseworkreq.AttentionCaseSearch{})
	if err != nil || total != 0 || len(list) != 0 {
		t.Fatalf("unrelated steward must see an empty list: total=%d list=%+v err=%v", total, list, err)
	}
	if _, err = fixture.service.Get(otherCtx, fixture.caseRow.ID); caseDomainCode(err) != caseworkmodel.CodeAccessScopeDenied {
		t.Fatalf("unrelated steward detail must fail closed, got %v", err)
	}
	if _, _, err = fixture.service.List(context.Background(), caseworkreq.AttentionCaseSearch{}); caseDomainCode(err) != caseworkmodel.CodeAccessScopeDenied {
		t.Fatalf("missing employee identity must fail closed, got %v", err)
	}

	assignedCtx := seedCaseActor(t, fixture.db, fixture.now, 46, 506, fixture.caseRow.CareClientID, caremodel.AuthorityRoleCareSteward)
	identity, _ := datascope.FromContext(assignedCtx)
	crossDepartmentCtx := datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: identity.UserID, AuthorityID: identity.AuthorityID, DeptID: 202,
		DeptIDs: []uint{202}, VisibleDeptIDs: []uint{202}, Scope: datascope.ScopeDept,
	})
	if _, err = fixture.service.Get(crossDepartmentCtx, fixture.caseRow.ID); caseDomainCode(err) != caseworkmodel.CodeAccessScopeDenied {
		t.Fatalf("cross-department detail must fail closed, got %v", err)
	}
}

func seedCaseRuleHit(t *testing.T, fixture caseWorkFixture) {
	t.Helper()
	hit := qmodel.QuestionnaireRuleHit{
		SubmissionID: fixture.caseRow.SubmissionID, AnswerRevisionID: 601, RuleVersionID: 701,
		ConditionSnapshotJSON: datatypes.JSON([]byte(`{"questionCode":"DEMO_CHOICE","operator":"EQUALS","value":"A"}`)),
		AttentionLevel:        fixture.caseRow.AttentionLevel, ReasonSnapshot: fixture.caseRow.ReasonSummary,
		RecipientsJSON: datatypes.JSON([]byte(`["ASSIGNED_CLINICIAN","SUPERVISOR"]`)),
		DedupKey:       fixture.caseRow.DedupKey, OccurredAt: fixture.now, Synthetic: true, DeptId: fixture.caseRow.DeptId,
	}
	hit.ID = fixture.caseRow.SourceRuleHitID
	if err := fixture.db.WithContext(datascope.WithSystem(context.Background())).Create(&hit).Error; err != nil {
		t.Fatal(err)
	}
}
