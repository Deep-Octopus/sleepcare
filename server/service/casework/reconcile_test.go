package casework

import (
	"context"
	"testing"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestReconcileRuleHitsIsGatedAndIdempotent(t *testing.T) {
	db := testutil.NewMemoryDB(t,
		&caremodel.CareAssignment{},
		&qmodel.QuestionnaireSubmission{}, &qmodel.QuestionnaireRuleHit{},
		&caseworkmodel.AttentionCase{}, &caseworkmodel.CaseAction{}, &caseworkmodel.TodoItem{}, &caseworkmodel.CommandReceipt{},
		&platformoutbox.Event{},
		testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	ctx := datascope.WithSystem(context.Background())
	submission := qmodel.QuestionnaireSubmission{
		TaskID: 301, CareClientID: 201, QuestionnaireVersionID: 101,
		BoundRuleVersionIDsJSON: datatypes.JSON([]byte(`[701]`)), Source: qmodel.SubmissionSourceClientSelf,
		ActorKind: qmodel.ActorKindClient, ActorID: 201, RequestHash: "request-hash",
		SubmittedAt: now.Add(-time.Hour), CurrentRevisionNo: 1, Synthetic: true, DeptId: 101, CreatedBy: 201,
	}
	if err := db.WithContext(ctx).Create(&submission).Error; err != nil {
		t.Fatal(err)
	}
	hit := qmodel.QuestionnaireRuleHit{
		SubmissionID: submission.ID, AnswerRevisionID: 601, RuleVersionID: 701,
		ConditionSnapshotJSON: datatypes.JSON([]byte(`{"questionCode":"DEMO_CHOICE","operator":"EQUALS","value":"A"}`)),
		AttentionLevel:        "SYNTHETIC_ATTENTION", ReasonSnapshot: "流程选项请求人工关注，不表示健康判断。",
		RecipientsJSON: datatypes.JSON([]byte(`["ASSIGNED_CLINICIAN","SUPERVISOR"]`)),
		DedupKey:       "submission:1:rule:701", OccurredAt: now.Add(-time.Hour), Synthetic: true, DeptId: 101, CreatedBy: 201,
	}
	if err := db.WithContext(ctx).Create(&hit).Error; err != nil {
		t.Fatal(err)
	}
	assignment := caremodel.CareAssignment{
		CareClientID: 201, OrganizationID: 100, TeamID: 101, AssigneeID: 42,
		RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: now.Add(-2 * time.Hour),
		Reason: "流程验证初始责任", Synthetic: true, DeptId: 101,
	}
	if err := db.WithContext(ctx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}

	disabled := false
	service := &CaseWorkService{DB: db, Now: func() time.Time { return now }, SyntheticFixturesEnabled: &disabled}
	count, err := service.ReconcileRuleHits(ctx)
	if err != nil || count != 0 {
		t.Fatalf("disabled reconciliation should be a no-op: count=%d err=%v", count, err)
	}
	assertCaseWorkCount(t, db, &caseworkmodel.AttentionCase{}, 0)

	enabled := true
	service.SyntheticFixturesEnabled = &enabled
	count, err = service.ReconcileRuleHits(ctx)
	if err != nil || count != 1 {
		t.Fatalf("first reconciliation should open one case: count=%d err=%v", count, err)
	}
	count, err = service.ReconcileRuleHits(ctx)
	if err != nil || count != 0 {
		t.Fatalf("repeated reconciliation should be a no-op: count=%d err=%v", count, err)
	}
	assertCaseWorkCount(t, db, &caseworkmodel.AttentionCase{}, 1)
	assertCaseWorkCount(t, db, &caseworkmodel.TodoItem{}, 1)
	assertCaseWorkCount(t, db, &platformoutbox.Event{}, 1)
}

func assertCaseWorkCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	var count int64
	if err := db.WithContext(datascope.WithSystem(context.Background())).Model(model).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%T count=%d, want %d", model, count, want)
	}
}
