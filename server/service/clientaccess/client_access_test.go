package clientaccess

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	platformoutbox "github.com/flipped-aurora/gin-vue-admin/server/internal/platform/outbox"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	clientreq "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/request"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type clientAccessFixture struct {
	db         *gorm.DB
	service    *ClientAccessService
	now        time.Time
	client     caremodel.CareClient
	account    clientmodel.CareClientAccount
	credential clientmodel.CareClientCredential
	task       pathmodel.TaskInstance
	rawGrant   string
}

func TestRedeemIsOneTimeAndSessionKeepsTaskScope(t *testing.T) {
	fixture := newClientAccessFixture(t)

	result, rawSession, err := fixture.service.Redeem(context.Background(), fixture.rawGrant)
	if err != nil {
		t.Fatal(err)
	}
	if rawSession == "" || result.AllowedTaskCount != 1 || !result.ExpiresAt.Equal(fixture.now.Add(2*time.Hour)) {
		t.Fatalf("unexpected redeem result: %+v", result)
	}
	var grant clientmodel.ClientAccessGrant
	if err = fixture.db.Where("token_digest = ?", DigestToken(fixture.rawGrant)).First(&grant).Error; err != nil {
		t.Fatal(err)
	}
	if grant.Status != clientmodel.GrantStatusRedeemed || grant.RedeemedAt == nil || grant.TokenDigest == fixture.rawGrant {
		t.Fatalf("grant was not consumed safely: %+v", grant)
	}
	var session clientmodel.ClientSession
	if err = fixture.db.Where("grant_id = ?", grant.ID).First(&session).Error; err != nil {
		t.Fatal(err)
	}
	if session.TokenDigest == rawSession || session.TokenDigest != DigestToken(rawSession) {
		t.Fatal("session persisted a raw bearer value")
	}
	identity, err := fixture.service.Authenticate(context.Background(), rawSession)
	if err != nil {
		t.Fatal(err)
	}
	if identity.CareClientID != fixture.client.ID || identity.AuthType != clientmodel.SessionAuthGrant || !identity.Synthetic ||
		len(identity.AllowedTaskIDs) != 1 || identity.AllowedTaskIDs[0] != fixture.task.ID {
		t.Fatalf("unexpected session identity: %+v", identity)
	}

	_, _, repeatedErr := fixture.service.Redeem(context.Background(), fixture.rawGrant)
	if domainCode(repeatedErr) != clientmodel.CodeGrantInvalid {
		t.Fatalf("repeated redeem should be indistinguishable invalid grant, got %v", repeatedErr)
	}
	expiredRaw := strings.Repeat("e", 43)
	createGrant(t, fixture, expiredRaw, fixture.now.Add(-time.Minute))
	_, _, expiredErr := fixture.service.Redeem(context.Background(), expiredRaw)
	if domainCode(expiredErr) != clientmodel.CodeGrantInvalid || expiredErr.Error() != repeatedErr.Error() {
		t.Fatalf("expired and consumed grants must have the same external error: %v / %v", expiredErr, repeatedErr)
	}
}

func TestAccountLoginCreatesOwnClientSessionAndLogoutRevokesIt(t *testing.T) {
	fixture := newClientAccessFixture(t)
	secondTask := fixture.task
	secondTask.ID = 0
	secondTask.DayCode = "D2"
	secondTask.Sort = 2
	secondTask.TaskDefinitionID = 2
	secondTask.QuestionnaireVersionID = nil
	if err := fixture.db.WithContext(datascope.WithSystem(context.Background())).Create(&secondTask).Error; err != nil {
		t.Fatal(err)
	}

	result, rawSession, err := fixture.service.Login(context.Background(), clientreq.Login{
		Username: "  CLIENT_A  ",
		Password: "client-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile.DisplayName != fixture.client.DisplayName || result.Profile.DisplayCode != fixture.client.DisplayCode || rawSession == "" {
		t.Fatalf("unexpected login result: %+v", result)
	}
	var session clientmodel.ClientSession
	if err = fixture.db.Where("token_digest = ?", DigestToken(rawSession)).First(&session).Error; err != nil {
		t.Fatal(err)
	}
	if session.GrantID != nil || session.AuthType != clientmodel.SessionAuthAccount || session.TokenDigest == rawSession {
		t.Fatalf("unexpected account session: %+v", session)
	}
	identity, err := fixture.service.Authenticate(context.Background(), rawSession)
	if err != nil {
		t.Fatal(err)
	}
	if identity.AuthType != clientmodel.SessionAuthAccount || len(identity.AllowedTaskIDs) != 0 || !identity.ExpiresAt.Equal(result.ExpiresAt) {
		t.Fatalf("unexpected account identity: %+v", identity)
	}
	ctx := ContextWithSessionIdentity(context.Background(), identity)
	list, total, err := fixture.service.ListTasks(ctx, clientreq.TaskSearch{})
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("account session should list all own tasks: total=%d list=%+v err=%v", total, list, err)
	}
	if _, err = fixture.service.GetTask(ctx, secondTask.ID); err != nil {
		t.Fatalf("account session should open an own task outside grant scope: %v", err)
	}
	otherTask := secondTask
	otherTask.ID = 0
	otherTask.CareClientID = fixture.client.ID + 100
	otherTask.TaskDefinitionID = 3
	otherTask.DayCode = "OTHER"
	if err = fixture.db.WithContext(datascope.WithSystem(context.Background())).Create(&otherTask).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.GetTask(ctx, otherTask.ID); domainCode(err) != clientmodel.CodeAccessScopeDenied {
		t.Fatalf("account session must not access another client's task, got %v", err)
	}
	profile, err := fixture.service.GetProfile(ctx)
	if err != nil || profile.DisplayName != fixture.client.DisplayName {
		t.Fatalf("unexpected session profile: %+v err=%v", profile, err)
	}
	if err = fixture.db.WithContext(datascope.WithSystem(context.Background())).Model(&clientmodel.CareClientCredential{}).
		Where("id = ?", fixture.credential.ID).
		Update("status", clientmodel.CredentialStatusDisabled).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.Authenticate(context.Background(), rawSession); domainCode(err) != clientmodel.CodeSessionInvalid {
		t.Fatalf("disabled credential should invalidate its account session, got %v", err)
	}
	if err = fixture.db.WithContext(datascope.WithSystem(context.Background())).Model(&clientmodel.CareClientCredential{}).
		Where("id = ?", fixture.credential.ID).
		Update("status", clientmodel.CredentialStatusActive).Error; err != nil {
		t.Fatal(err)
	}
	logout, err := fixture.service.Logout(ctx)
	if err != nil || !logout.SignedOut {
		t.Fatalf("logout failed: %+v err=%v", logout, err)
	}
	if _, err = fixture.service.Authenticate(context.Background(), rawSession); domainCode(err) != clientmodel.CodeSessionInvalid {
		t.Fatalf("revoked session should be invalid, got %v", err)
	}
}

func TestAccountLoginCountsFailuresAndTemporarilyLocksCredential(t *testing.T) {
	fixture := newClientAccessFixture(t)
	_, _, missingErr := fixture.service.Login(context.Background(), clientreq.Login{
		Username: "missing_account",
		Password: "wrong-password",
	})
	if domainCode(missingErr) != clientmodel.CodeCredentialsInvalid {
		t.Fatalf("missing username should use the generic credential error, got %v", missingErr)
	}
	for attempt := 1; attempt <= maxCredentialFailures; attempt++ {
		_, _, err := fixture.service.Login(context.Background(), clientreq.Login{
			Username: fixture.credential.Username,
			Password: "wrong-password",
		})
		wantCode := clientmodel.CodeCredentialsInvalid
		if attempt == maxCredentialFailures {
			wantCode = clientmodel.CodeCredentialLocked
		}
		if domainCode(err) != wantCode {
			t.Fatalf("attempt %d code=%d want=%d err=%v", attempt, domainCode(err), wantCode, err)
		}
	}
	var stored clientmodel.CareClientCredential
	if err := fixture.db.First(&stored, fixture.credential.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.FailedLoginCount != maxCredentialFailures || stored.LockedUntil == nil {
		t.Fatalf("credential lock state not persisted: %+v", stored)
	}
	_, _, err := fixture.service.Login(context.Background(), clientreq.Login{
		Username: fixture.credential.Username,
		Password: "client-password",
	})
	if domainCode(err) != clientmodel.CodeCredentialLocked {
		t.Fatalf("correct password must not bypass active lock, got %v", err)
	}
	past := fixture.now.Add(-time.Minute)
	if err = fixture.db.Model(&stored).Update("locked_until", past).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err = fixture.service.Login(context.Background(), clientreq.Login{
		Username: fixture.credential.Username,
		Password: "client-password",
	}); err != nil {
		t.Fatalf("expired lock should allow a valid login: %v", err)
	}
}

func TestLimitedSessionConsultationFlowStaysWithinCurrentClient(t *testing.T) {
	fixture := newClientAccessFixture(t)
	seedCtx := datascope.WithSystem(context.Background())
	assignment := caremodel.CareAssignment{
		CareClientID: fixture.client.ID, OrganizationID: fixture.client.OrganizationID,
		TeamID: fixture.client.DeptId, AssigneeID: 701, RoleType: caremodel.AssignmentRoleCareSteward,
		ValidFrom: fixture.now.Add(-time.Hour), Reason: "当前服务责任",
		Synthetic: true, DeptId: fixture.client.DeptId,
	}
	if err := fixture.db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	ctx := redeemedContext(t, fixture)
	created, err := fixture.service.CreateConsultation(ctx, "client-consultation", caseworkreq.CreateConsultation{
		Subject: "服务时间咨询",
		Message: "我想确认后续联系时间。",
		Urgency: caseworkmodel.ConsultationUrgencyRoutine,
	})
	if err != nil {
		t.Fatal(err)
	}
	list, total, err := fixture.service.ListConsultations(ctx, caseworkreq.ClientConsultationSearch{})
	if err != nil || total != 1 || len(list) != 1 || list[0].ID != created.ConsultationID {
		t.Fatalf("unexpected limited-session consultation list: total=%d list=%+v err=%v", total, list, err)
	}
	detail, err := fixture.service.GetConsultation(ctx, created.ConsultationID)
	if err != nil || detail.InitialQuestion == "" || len(detail.Interactions) != 2 {
		t.Fatalf("unexpected limited-session consultation detail: %+v err=%v", detail, err)
	}
	message, err := fixture.service.AddConsultationMessage(
		ctx,
		created.ConsultationID,
		"client-consultation-message",
		caseworkreq.AddClientConsultationMessage{
			ExpectedVersion: detail.Version,
			Message:         "工作日下午方便联系。",
		},
	)
	if err != nil || message.Version != detail.Version+1 {
		t.Fatalf("unexpected consultation supplement result: %+v err=%v", message, err)
	}
}

func TestClientTaskFlowPersistsDraftAndSubmitsOnce(t *testing.T) {
	fixture := newClientAccessFixture(t)
	ctx := redeemedContext(t, fixture)

	opened := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "opened-key", 1, clientmodel.InteractionOpened)
	consented := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "consented-key", opened.TaskVersion, clientmodel.InteractionConsented)
	started := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "started-key", consented.TaskVersion, clientmodel.InteractionStarted)
	if started.ExecutionStatus != pathmodel.ExecutionInProgress || started.TaskVersion != 4 {
		t.Fatalf("unexpected started result: %+v", started)
	}

	draft, err := fixture.service.SaveDraft(ctx, fixture.task.ID, "draft-key", clientreq.SaveDraft{
		ExpectedVersion: 0, Answers: map[string]any{"DEMO_CHOICE": "A"},
	})
	if err != nil || draft.Version != 1 {
		t.Fatalf("save draft failed: %+v %v", draft, err)
	}
	replayedDraft, err := fixture.service.SaveDraft(ctx, fixture.task.ID, "draft-key", clientreq.SaveDraft{
		ExpectedVersion: 0, Answers: map[string]any{"DEMO_CHOICE": "A"},
	})
	if err != nil || replayedDraft.Version != 1 {
		t.Fatalf("draft replay failed: %+v %v", replayedDraft, err)
	}
	assertCount(t, fixture.db, &qmodel.QuestionnaireTaskDraft{}, "task_id = ?", fixture.task.ID, 1)

	submitRequest := clientreq.SubmitTask{
		ExpectedTaskVersion: started.TaskVersion, Source: qmodel.SubmissionSourceClientSelf,
		Answers: map[string]any{"DEMO_CHOICE": "A"},
	}
	submitted, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "submit-key", submitRequest)
	if err != nil {
		t.Fatal(err)
	}
	if submitted.ExecutionStatus != pathmodel.ExecutionSubmitted || submitted.ReviewStatus != pathmodel.ReviewPending || submitted.TaskVersion != 5 || len(submitted.AttentionCaseIDs) != 0 {
		t.Fatalf("unexpected submit result: %+v", submitted)
	}
	replayed, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "submit-key", submitRequest)
	if err != nil || replayed.SubmissionID != submitted.SubmissionID {
		t.Fatalf("same-key replay failed: %+v %v", replayed, err)
	}
	equivalent, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "equivalent-key", submitRequest)
	if err != nil || equivalent.SubmissionID != submitted.SubmissionID {
		t.Fatalf("equivalent submission should return existing result: %+v %v", equivalent, err)
	}
	changed := submitRequest
	changed.Answers = map[string]any{"DEMO_CHOICE": "B"}
	if _, err = fixture.service.SubmitTask(ctx, fixture.task.ID, "changed-key", changed); domainCode(err) != clientmodel.CodeIdempotencyConflict {
		t.Fatalf("different submitted content should conflict, got %v", err)
	}

	assertCount(t, fixture.db, &qmodel.QuestionnaireSubmission{}, "task_id = ?", fixture.task.ID, 1)
	assertCount(t, fixture.db, &qmodel.QuestionnaireAnswerRevision{}, "1 = 1", nil, 1)
	assertCount(t, fixture.db, &platformoutbox.Event{}, "event_type = ?", qmodel.EventTaskAnswerSubmitted, 1)
	assertCount(t, fixture.db, &pathmodel.CarePathEvent{}, "event_type = ?", pathmodel.EventClientTaskOpened, 1)
	assertCount(t, fixture.db, &pathmodel.CarePathEvent{}, "event_type = ?", pathmodel.EventClientTaskConsented, 1)
	assertCount(t, fixture.db, &pathmodel.CarePathEvent{}, "event_type = ?", pathmodel.EventTaskAnswerStarted, 1)
	assertCount(t, fixture.db, &pathmodel.CarePathEvent{}, "event_type = ?", pathmodel.EventTaskAnswerSubmitted, 1)
	var storedDraft qmodel.QuestionnaireTaskDraft
	if err = fixture.db.Where("task_id = ?", fixture.task.ID).First(&storedDraft).Error; err != nil || storedDraft.ConsumedAt == nil {
		t.Fatalf("submitted draft should be retained and marked consumed: %+v %v", storedDraft, err)
	}
}

func TestSubmitCreatesAttentionCaseForMatchingPublishedRule(t *testing.T) {
	fixture := newClientAccessFixture(t)
	bindMatchingAttentionRule(t, &fixture)
	ctx := redeemedContext(t, fixture)

	opened := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "opened-key", 1, clientmodel.InteractionOpened)
	consented := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "consented-key", opened.TaskVersion, clientmodel.InteractionConsented)
	started := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "started-key", consented.TaskVersion, clientmodel.InteractionStarted)

	request := clientreq.SubmitTask{
		ExpectedTaskVersion: started.TaskVersion,
		Source:              qmodel.SubmissionSourceClientSelf,
		Answers:             map[string]any{"DEMO_CHOICE": "A"},
	}
	result, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "submit-attention-key", request)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RuleHitIDs) != 1 || len(result.AttentionCaseIDs) != 1 {
		t.Fatalf("matching rule should return one hit and one attention case: %+v", result)
	}

	replayed, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "submit-attention-key", request)
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed.AttentionCaseIDs) != 1 || replayed.AttentionCaseIDs[0] != result.AttentionCaseIDs[0] {
		t.Fatalf("replayed submit should return the original attention case: first=%+v replayed=%+v", result, replayed)
	}
	assertCount(t, fixture.db, &caseworkmodel.AttentionCase{}, "submission_id = ?", result.SubmissionID, 1)
	assertCount(t, fixture.db, &caseworkmodel.TodoItem{}, "source_id = ?", result.AttentionCaseIDs[0], 1)
	assertCount(t, fixture.db, &platformoutbox.Event{}, "event_type = ?", caseworkmodel.EventAttentionCaseOpened, 1)
}

func TestSubmitMatchingRuleFailsAtomicallyWithoutActiveSteward(t *testing.T) {
	fixture := newClientAccessFixture(t)
	bindMatchingAttentionRule(t, &fixture)
	ctx := redeemedContext(t, fixture)
	if err := fixture.db.WithContext(datascope.WithSystem(context.Background())).
		Where("care_client_id = ? AND role_type = ?", fixture.client.ID, caremodel.AssignmentRoleCareSteward).
		Delete(&caremodel.CareAssignment{}).Error; err != nil {
		t.Fatal(err)
	}

	opened := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "opened-key", 1, clientmodel.InteractionOpened)
	consented := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "consented-key", opened.TaskVersion, clientmodel.InteractionConsented)
	started := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "started-key", consented.TaskVersion, clientmodel.InteractionStarted)
	_, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "submit-without-steward", clientreq.SubmitTask{
		ExpectedTaskVersion: started.TaskVersion,
		Source:              qmodel.SubmissionSourceClientSelf,
		Answers:             map[string]any{"DEMO_CHOICE": "A"},
	})
	if domainCode(err) != clientmodel.CodeCareAssignmentRequired {
		t.Fatalf("submission without an active steward should fail with %d, got %v", clientmodel.CodeCareAssignmentRequired, err)
	}
	assertCount(t, fixture.db, &qmodel.QuestionnaireSubmission{}, "1 = 1", nil, 0)
	assertCount(t, fixture.db, &qmodel.QuestionnaireRuleHit{}, "1 = 1", nil, 0)
	assertCount(t, fixture.db, &caseworkmodel.AttentionCase{}, "1 = 1", nil, 0)
	assertCount(t, fixture.db, &caseworkmodel.TodoItem{}, "1 = 1", nil, 0)
	assertCount(t, fixture.db, &platformoutbox.Event{}, "event_type = ?", caseworkmodel.EventAttentionCaseOpened, 0)
	var task pathmodel.TaskInstance
	if err = fixture.db.WithContext(datascope.WithSystem(context.Background())).First(&task, fixture.task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.ExecutionStatus != pathmodel.ExecutionInProgress || task.Version != started.TaskVersion {
		t.Fatalf("failed submission must leave task in progress: %+v", task)
	}
}

func TestClientTaskScopeAndTimingFailClosed(t *testing.T) {
	fixture := newClientAccessFixture(t)
	ctx := redeemedContext(t, fixture)

	other := fixture.task
	other.ID = 0
	other.DayCode = "D2"
	other.Sort = 2
	other.TaskDefinitionID = 2
	other.OpenAt = fixture.now.Add(24 * time.Hour)
	other.DueAt = fixture.now.Add(35 * time.Hour)
	other.ExecutionStatus = pathmodel.ExecutionScheduled
	other.QuestionnaireVersionID = nil
	if err := fixture.db.WithContext(datascope.WithSystem(context.Background())).Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.GetTask(ctx, other.ID); domainCode(err) != clientmodel.CodeAccessScopeDenied {
		t.Fatalf("out-of-scope task must be forbidden, got %v", err)
	}

	identity, _ := SessionIdentityFromContext(ctx)
	identity.AllowedTaskIDs = append(identity.AllowedTaskIDs, other.ID)
	expandedCtx := ContextWithSessionIdentity(context.Background(), identity)
	if _, err := fixture.service.GetTask(expandedCtx, other.ID); domainCode(err) != clientmodel.CodeTaskNotOpen {
		t.Fatalf("future task must stay closed, got %v", err)
	}
}

func TestListTasksOpensDueTaskOnceWithOutbox(t *testing.T) {
	fixture := newClientAccessFixture(t)
	ctx := redeemedContext(t, fixture)
	if err := fixture.db.WithContext(datascope.WithSystem(context.Background())).Model(&pathmodel.TaskInstance{}).
		Where("id = ?", fixture.task.ID).
		Updates(map[string]any{
			"execution_status": pathmodel.ExecutionScheduled,
			"opened_at":        nil,
			"version":          1,
		}).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := fixture.service.ListTasks(ctx, clientreq.TaskSearch{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ExecutionStatus != pathmodel.ExecutionOpen || list[0].Version != 2 {
		t.Fatalf("due task was not opened: total=%d list=%+v", total, list)
	}
	if _, _, err = fixture.service.ListTasks(ctx, clientreq.TaskSearch{}); err != nil {
		t.Fatal(err)
	}
	assertCount(t, fixture.db, &pathmodel.CarePathEvent{}, "event_type = ?", pathmodel.EventTaskOpened, 1)
	assertCount(t, fixture.db, &platformoutbox.Event{}, "event_type = ?", pathmodel.EventTaskOpened, 1)
}

func TestSubmitRollsBackQuestionnaireWhenTaskEventFails(t *testing.T) {
	fixture := newClientAccessFixture(t)
	ctx := redeemedContext(t, fixture)
	opened := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "opened-key", 1, clientmodel.InteractionOpened)
	consented := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "consented-key", opened.TaskVersion, clientmodel.InteractionConsented)
	started := recordInteraction(t, fixture.service, ctx, fixture.task.ID, "started-key", consented.TaskVersion, clientmodel.InteractionStarted)

	callbackName := "test:fail-submitted-task-event"
	if err := fixture.db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table != "care_path_events" {
			return
		}
		event, ok := tx.Statement.Dest.(*pathmodel.CarePathEvent)
		if ok && event.EventType == pathmodel.EventTaskAnswerSubmitted {
			tx.AddError(errors.New("forced task event failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.db.Callback().Create().Remove(callbackName) })

	_, err := fixture.service.SubmitTask(ctx, fixture.task.ID, "submit-fail-key", clientreq.SubmitTask{
		ExpectedTaskVersion: started.TaskVersion, Source: qmodel.SubmissionSourceClientSelf,
		Answers: map[string]any{"DEMO_CHOICE": "A"},
	})
	if err == nil {
		t.Fatal("submit should fail when the task event cannot be written")
	}
	assertCount(t, fixture.db, &qmodel.QuestionnaireSubmission{}, "1 = 1", nil, 0)
	assertCount(t, fixture.db, &qmodel.QuestionnaireAnswerRevision{}, "1 = 1", nil, 0)
	assertCount(t, fixture.db, &platformoutbox.Event{}, "1 = 1", nil, 0)
	var task pathmodel.TaskInstance
	if err = fixture.db.Where("id = ?", fixture.task.ID).First(&task).Error; err != nil {
		t.Fatal(err)
	}
	if task.ExecutionStatus != pathmodel.ExecutionInProgress || task.Version != started.TaskVersion || task.SubmittedAt != nil {
		t.Fatalf("task mutation was not rolled back: %+v", task)
	}
}

func newClientAccessFixture(t *testing.T) clientAccessFixture {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{},
		&clientmodel.CareClientAccount{}, &clientmodel.CareClientCredential{}, &clientmodel.ClientAccessGrant{}, &clientmodel.ClientSession{}, &clientmodel.ClientTaskCommandReceipt{},
		&pathmodel.PlanInstance{}, &pathmodel.TaskInstance{}, &pathmodel.CarePathEvent{},
		&qmodel.QuestionnaireVersion{}, &qmodel.QuestionnaireQuestion{}, &qmodel.QuestionnaireOption{}, &qmodel.QuestionnaireRuleVersion{},
		&qmodel.QuestionnaireSubmission{}, &qmodel.QuestionnaireTaskDraft{}, &qmodel.QuestionnaireAnswerRevision{},
		&qmodel.QuestionnaireRuleHit{}, &qmodel.QuestionnaireCommandReceipt{}, &platformoutbox.Event{},
		&caseworkmodel.AttentionCase{}, &caseworkmodel.CaseAction{}, &caseworkmodel.TodoItem{}, &caseworkmodel.CommandReceipt{},
		&caseworkmodel.Consultation{}, &caseworkmodel.ConsultationInteraction{},
		testutil.WithDataScopeCallbacks(),
	)
	enabled := true
	now := time.Date(2026, time.August, 18, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	service := &ClientAccessService{DB: db, Now: func() time.Time { return now }, SessionTTL: 2 * time.Hour, SyntheticFixturesEnabled: &enabled}
	seedCtx := datascope.WithSystem(context.Background())
	client := caremodel.CareClient{
		DisplayCode: "DEMO-CLIENT-A", DisplayName: "测试用户甲", ServiceReason: "流程验证",
		ServicePackageCode: "DEMO-PACKAGE", OrganizationID: 100, Status: caremodel.ClientStatusActive,
		SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: 101,
	}
	if err := db.WithContext(seedCtx).Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	account := clientmodel.CareClientAccount{CareClientID: client.ID, Status: clientmodel.AccountStatusActive, Version: 1, Synthetic: true, DeptId: client.DeptId}
	if err := db.WithContext(seedCtx).Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	credential := clientmodel.CareClientCredential{
		AccountID: account.ID, Username: "client_a", PasswordHash: utils.BcryptHash("client-password"),
		Status: clientmodel.CredentialStatusActive, PasswordUpdatedAt: now,
		Version: 1, Synthetic: true, DeptId: client.DeptId,
	}
	if err := db.WithContext(seedCtx).Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	questionnaireID := seedQuestionnaire(t, db.WithContext(seedCtx), now)
	plan := pathmodel.PlanInstance{
		EnrollmentID: 1, CareClientID: client.ID, PlanTemplateVersionID: 1, PreviewID: 1,
		AnchorAt: now.Add(-time.Hour), Status: pathmodel.EnrollmentActive,
		PauseStrategy: pathmodel.PauseStrategyKeepWindows, Version: 1, Synthetic: true, DeptId: client.DeptId,
	}
	if err := db.WithContext(seedCtx).Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	ruleJSON, _ := json.Marshal([]uint{})
	task := pathmodel.TaskInstance{
		PlanInstanceID: plan.ID, CareClientID: client.ID, TaskDefinitionID: 1,
		DayCode: "D1", Title: "今日任务", Sort: 1, ExecutionRole: pathmodel.ExecutionRoleCareClient,
		ExecutionStatus: pathmodel.ExecutionOpen, ReviewStatus: pathmodel.ReviewNotReady, ReviewRole: pathmodel.ExecutionRoleClinician,
		OpenAt: now.Add(-time.Hour), DueAt: now.Add(10 * time.Hour), QuestionnaireVersionID: &questionnaireID,
		BoundRuleVersionIDsJSON: datatypes.JSON(ruleJSON), LateSubmissionPolicy: pathmodel.LateSubmissionDeny,
		NotificationPolicy: pathmodel.NotificationPolicyDisabled, Version: 1, Synthetic: true, DeptId: client.DeptId,
	}
	if err := db.WithContext(seedCtx).Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	rawGrant := strings.Repeat("g", 43)
	fixture := clientAccessFixture{
		db: db, service: service, now: now, client: client, account: account,
		credential: credential, task: task, rawGrant: rawGrant,
	}
	createGrant(t, fixture, rawGrant, now.Add(time.Hour))
	return fixture
}

func bindMatchingAttentionRule(t *testing.T, fixture *clientAccessFixture) {
	t.Helper()
	condition := json.RawMessage(`{"questionCode":"DEMO_CHOICE","operator":"EQUALS","value":"A"}`)
	recipients := json.RawMessage(`["ASSIGNED_CLINICIAN","SUPERVISOR"]`)
	definition := qmodel.RuleDefinition{
		QuestionnaireVersionID: *fixture.task.QuestionnaireVersionID,
		Code:                   "DEMO-ATTENTION",
		Version:                "1.0.0",
		Title:                  "流程关注规则",
		UsageScope:             qmodel.UsageScopeTestOnly,
		Synthetic:              true,
		ProductionEnabled:      false,
		ConditionSchemaVersion: "v1",
		Condition:              qmodel.CanonicalJSON(condition),
		AttentionLevel:         "SYNTHETIC_ATTENTION",
		ReasonSnapshot:         "流程选项请求人工关注，不表示健康判断。",
		Recipients:             qmodel.CanonicalJSON(recipients),
		DedupKeyTemplate:       "submission:{submissionId}:rule:{ruleVersionId}",
	}
	hash, err := qmodel.HashDefinition(definition)
	if err != nil {
		t.Fatal(err)
	}
	published := fixture.now.Add(-time.Hour)
	rule := qmodel.QuestionnaireRuleVersion{
		QuestionnaireVersionID: definition.QuestionnaireVersionID,
		Code:                   definition.Code,
		Version:                definition.Version,
		Title:                  definition.Title,
		Status:                 qmodel.LifecyclePublished,
		UsageScope:             definition.UsageScope,
		Synthetic:              true,
		ProductionEnabled:      false,
		ReviewType:             qmodel.ReviewTypeEngineering,
		ReviewedBy:             700,
		ReviewedAt:             &published,
		ConditionSchemaVersion: definition.ConditionSchemaVersion,
		ConditionJSON:          datatypes.JSON(condition),
		AttentionLevel:         definition.AttentionLevel,
		ReasonSnapshot:         definition.ReasonSnapshot,
		RecipientsJSON:         datatypes.JSON(recipients),
		DedupKeyTemplate:       definition.DedupKeyTemplate,
		PublishedAt:            &published,
		DefinitionHash:         hash,
		RowVersion:             1,
	}
	seedCtx := datascope.WithSystem(context.Background())
	if err = fixture.db.WithContext(seedCtx).Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	bound, err := json.Marshal([]uint{rule.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err = fixture.db.WithContext(seedCtx).Model(&pathmodel.TaskInstance{}).
		Where("id = ?", fixture.task.ID).
		Update("bound_rule_version_ids_json", datatypes.JSON(bound)).Error; err != nil {
		t.Fatal(err)
	}
	fixture.task.BoundRuleVersionIDsJSON = datatypes.JSON(bound)
	assignment := caremodel.CareAssignment{
		CareClientID:   fixture.client.ID,
		OrganizationID: fixture.client.OrganizationID,
		TeamID:         fixture.client.DeptId,
		AssigneeID:     701,
		RoleType:       caremodel.AssignmentRoleCareSteward,
		ValidFrom:      fixture.now.Add(-time.Hour),
		Reason:         "流程验证初始责任",
		Synthetic:      true,
		DeptId:         fixture.client.DeptId,
	}
	if err = fixture.db.WithContext(seedCtx).Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
}

func seedQuestionnaire(t *testing.T, db *gorm.DB, now time.Time) uint {
	t.Helper()
	definition := qmodel.VersionDefinition{
		Code: "DEMO-FORM", Version: "1.0.0", Title: "演示任务表单", Purpose: "验证填写流程",
		UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true, ProductionEnabled: false,
		ExpectedMinutes: 1, DefinitionSchemaVersion: "v1",
		Questions: []qmodel.QuestionDefinition{{
			Code: "DEMO_CHOICE", Type: qmodel.QuestionTypeSingleChoice, Title: "请选择一个演示选项",
			Required: true, Sort: 1, ValidationSchemaVersion: "v1", Validation: json.RawMessage(`{}`),
			Options: []qmodel.OptionDefinition{{Code: "A", Label: "选项 A", Sort: 1}, {Code: "B", Label: "选项 B", Sort: 2}},
		}},
	}
	hash, err := qmodel.HashDefinition(definition)
	if err != nil {
		t.Fatal(err)
	}
	published := now.Add(-2 * time.Hour)
	version := qmodel.QuestionnaireVersion{
		Code: definition.Code, Version: definition.Version, Title: definition.Title, Purpose: definition.Purpose,
		Status: qmodel.LifecyclePublished, UsageScope: qmodel.UsageScopeTestOnly, Synthetic: true,
		ReviewType: qmodel.ReviewTypeEngineering, ReviewedBy: 1, ReviewedAt: &published, PublishedAt: &published,
		ExpectedMinutes: 1, DefinitionSchemaVersion: "v1", DefinitionHash: hash, RowVersion: 1,
	}
	if err = db.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	question := qmodel.QuestionnaireQuestion{
		QuestionnaireVersionID: version.ID, Code: "DEMO_CHOICE", Type: qmodel.QuestionTypeSingleChoice,
		Title: "请选择一个演示选项", Required: true, Sort: 1, ValidationSchemaVersion: "v1", ValidationJSON: datatypes.JSON([]byte(`{}`)),
	}
	if err = db.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	options := []qmodel.QuestionnaireOption{
		{QuestionID: question.ID, Code: "A", Label: "选项 A", Sort: 1},
		{QuestionID: question.ID, Code: "B", Label: "选项 B", Sort: 2},
	}
	if err = db.Create(&options).Error; err != nil {
		t.Fatal(err)
	}
	return version.ID
}

func createGrant(t *testing.T, fixture clientAccessFixture, raw string, expiresAt time.Time) {
	t.Helper()
	allowed, _ := json.Marshal([]uint{fixture.task.ID})
	grant := clientmodel.ClientAccessGrant{
		AccountID: fixture.account.ID, CareClientID: fixture.client.ID, TokenDigest: DigestToken(raw),
		AllowedTaskIDsJSON: datatypes.JSON(allowed), Status: clientmodel.GrantStatusIssued,
		ExpiresAt: expiresAt, Synthetic: true, DeptId: fixture.client.DeptId,
	}
	if err := fixture.db.WithContext(datascope.WithSystem(context.Background())).Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
}

func redeemedContext(t *testing.T, fixture clientAccessFixture) context.Context {
	t.Helper()
	_, rawSession, err := fixture.service.Redeem(context.Background(), fixture.rawGrant)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := fixture.service.Authenticate(context.Background(), rawSession)
	if err != nil {
		t.Fatal(err)
	}
	return ContextWithSessionIdentity(context.Background(), identity)
}

func recordInteraction(t *testing.T, service *ClientAccessService, ctx context.Context, taskID uint, key string, version uint, interaction string) clientres.InteractionResult {
	t.Helper()
	result, err := service.RecordInteraction(ctx, taskID, key, clientreq.RecordInteraction{ExpectedVersion: version, InteractionType: interaction})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func domainCode(err error) int {
	var domainErr *clientmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

func assertCount(t *testing.T, db *gorm.DB, model any, where string, arg any, want int64) {
	t.Helper()
	var count int64
	query := db.WithContext(datascope.WithSystem(context.Background())).Model(model)
	if where != "1 = 1" {
		query = query.Where(where, arg)
	}
	if err := query.Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%T count=%d, want %d", model, count, want)
	}
}
