package supervision

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
	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	caseworkservice "github.com/flipped-aurora/gin-vue-admin/server/service/casework"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

type satisfactionFixture struct {
	db              *gorm.DB
	service         *SupervisionService
	caseWork        *caseworkservice.CaseWorkService
	now             time.Time
	systemCtx       context.Context
	client          caremodel.CareClient
	clientCtx       context.Context
	clientIdentity  ClientSatisfactionIdentity
	supervisor      system.SysUser
	supervisorCtx   context.Context
	crossSupervisor system.SysUser
	crossCtx        context.Context
	steward         system.SysUser
	stewardCtx      context.Context
	consultation    caseworkmodel.Consultation
	projector       *capturingSatisfactionProjector
}

type capturingSatisfactionProjector struct {
	service      *SupervisionService
	calls        int
	consultation caseworkmodel.Consultation
	interaction  caseworkmodel.ConsultationInteraction
}

func (p *capturingSatisfactionProjector) ProjectConsultationClosed(
	ctx context.Context,
	tx *gorm.DB,
	consultation caseworkmodel.Consultation,
	interaction caseworkmodel.ConsultationInteraction,
) error {
	p.calls++
	p.consultation = consultation
	p.interaction = interaction
	return p.service.ProjectConsultationClosed(ctx, tx, consultation, interaction)
}

func TestSatisfactionLowScoreLifecycleIsAnonymousIdempotentAndBounded(t *testing.T) {
	fixture := newSatisfactionFixture(t)
	closed, err := fixture.caseWork.CloseConsultation(
		fixture.supervisorCtx,
		fixture.consultation.ID,
		"close-for-evaluation",
		caseworkreq.CloseConsultation{
			ExpectedVersion: fixture.consultation.Version,
			CloseReason:     "本轮服务沟通已经完成。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if fixture.projector.calls != 1 || !fixture.projector.consultation.Synthetic ||
		fixture.projector.interaction.ID != closed.InteractionID ||
		fixture.projector.interaction.ActionType != caseworkmodel.ConsultationActionClose {
		t.Fatalf("unexpected consultation-close projection call: %+v", fixture.projector)
	}
	if _, found, policyErr := fixture.service.satisfactionPolicyAt(
		fixture.db.WithContext(fixture.systemCtx).Set("data_scope:skip", true),
		fixture.projector.interaction.OccurredAt,
	); policyErr != nil || !found {
		t.Fatalf("evaluation policy must cover close event: found=%t err=%v event=%+v", found, policyErr, fixture.projector.interaction)
	}
	if !closed.OccurredAt.Equal(fixture.now) {
		t.Fatalf("close event must use the service clock: got=%v want=%v", closed.OccurredAt, fixture.now)
	}
	var closedInteraction caseworkmodel.ConsultationInteraction
	if err = fixture.db.WithContext(fixture.systemCtx).First(&closedInteraction, closed.InteractionID).Error; err != nil {
		t.Fatal(err)
	}
	if closedInteraction.ActionType != caseworkmodel.ConsultationActionClose || !closedInteraction.Synthetic {
		t.Fatalf("unexpected close event projection input: %+v", closedInteraction)
	}
	replayedClose, err := fixture.caseWork.CloseConsultation(
		fixture.supervisorCtx,
		fixture.consultation.ID,
		"close-for-evaluation",
		caseworkreq.CloseConsultation{
			ExpectedVersion: fixture.consultation.Version,
			CloseReason:     "本轮服务沟通已经完成。",
		},
	)
	if err != nil || replayedClose.InteractionID != closed.InteractionID || replayedClose.Version != closed.Version {
		t.Fatalf("close replay mismatch: result=%+v err=%v", replayedClose, err)
	}

	var request supervisionmodel.SatisfactionRequest
	if err = fixture.db.WithContext(fixture.systemCtx).Where(
		"source_type = ? AND source_event_id = ?",
		supervisionmodel.SatisfactionSourceConsultation,
		closed.InteractionID,
	).First(&request).Error; err != nil {
		t.Fatal(err)
	}
	if request.Status != supervisionmodel.SatisfactionRequestPending || request.Version != 1 ||
		request.DeptId != fixture.client.DeptId || request.AnonymityMode != supervisionmodel.SatisfactionAnonymousStaff ||
		request.LowScoreThreshold != 2 || request.ExpiresAt.Sub(request.InvitedAt) != 7*24*time.Hour {
		t.Fatalf("unexpected evaluation request snapshot: %+v", request)
	}
	assertSatisfactionCount(t, fixture.db, &supervisionmodel.SatisfactionRequest{}, "source_event_id = ?", closed.InteractionID, 1)
	assertSatisfactionCount(t, fixture.db, &platformoutbox.Event{}, "event_type = ?", supervisionmodel.EventSatisfactionRequested, 1)

	clientList, total, err := fixture.service.ListClientSatisfactionRequests(
		fixture.clientCtx,
		fixture.clientIdentity,
		supervisionreq.ClientSatisfactionSearch{},
	)
	if err != nil || total != 1 || len(clientList) != 1 || clientList[0].PublicCode == "" {
		t.Fatalf("unexpected client evaluation list: total=%d list=%+v err=%v", total, clientList, err)
	}
	submitRequest := supervisionreq.SubmitSatisfactionResponse{
		ExpectedVersion: request.Version,
		Rating:          2,
		Comment:         "希望后续沟通安排更清晰。",
	}
	submitted, err := fixture.service.SubmitClientSatisfactionResponse(
		fixture.clientCtx,
		fixture.clientIdentity,
		request.ID,
		"submit-evaluation",
		submitRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if submitted.Status != supervisionmodel.SatisfactionRequestSubmitted || submitted.Version != 2 || !submitted.FollowUpCreated {
		t.Fatalf("unexpected submit result: %+v", submitted)
	}
	replayed, err := fixture.service.SubmitClientSatisfactionResponse(
		fixture.clientCtx,
		fixture.clientIdentity,
		request.ID,
		"submit-evaluation",
		submitRequest,
	)
	if err != nil || replayed.ResponseID != submitted.ResponseID || replayed.Version != submitted.Version ||
		!replayed.SubmittedAt.Equal(submitted.SubmittedAt) {
		t.Fatalf("evaluation replay mismatch: result=%+v err=%v", replayed, err)
	}
	changed := submitRequest
	changed.Rating = 1
	if _, err = fixture.service.SubmitClientSatisfactionResponse(
		fixture.clientCtx,
		fixture.clientIdentity,
		request.ID,
		"submit-evaluation",
		changed,
	); satisfactionDomainCode(err) != supervisionmodel.CodeIdempotencyConflict {
		t.Fatalf("same key with changed rating should conflict, got %v", err)
	}

	responses, responseTotal, err := fixture.service.ListSatisfactionResponses(
		fixture.supervisorCtx,
		supervisionreq.SatisfactionResponseSearch{},
	)
	if err != nil || responseTotal != 1 || len(responses) != 1 || responses[0].PublicCode != clientList[0].PublicCode ||
		responses[0].Rating != 2 || responses[0].FollowUpID == nil {
		t.Fatalf("unexpected anonymous response projection: total=%d list=%+v err=%v", responseTotal, responses, err)
	}
	if _, _, err = fixture.service.ListSatisfactionResponses(
		fixture.stewardCtx,
		supervisionreq.SatisfactionResponseSearch{},
	); satisfactionDomainCode(err) != supervisionmodel.CodeReviewScopeDenied {
		t.Fatalf("non-supervisor should fail closed, got %v", err)
	}
	assertSatisfactionAnonymousJSON(t, responses[0])

	followUps, followUpTotal, err := fixture.service.ListSatisfactionFollowUps(
		fixture.supervisorCtx,
		supervisionreq.SatisfactionFollowUpSearch{},
	)
	if err != nil || followUpTotal != 1 || len(followUps) != 1 ||
		followUps[0].Status != supervisionmodel.SatisfactionFollowUpOpen || followUps[0].AssigneeID == nil ||
		*followUps[0].AssigneeID != fixture.supervisor.ID {
		t.Fatalf("unexpected quality follow-up: total=%d list=%+v err=%v", followUpTotal, followUps, err)
	}
	followUpID := followUps[0].ID
	if _, err = fixture.service.GetSatisfactionFollowUp(fixture.crossCtx, followUpID); satisfactionDomainCode(err) != supervisionmodel.CodeSatisfactionScopeDenied {
		t.Fatalf("cross-organization detail should fail closed, got %v", err)
	}

	acknowledged, err := fixture.service.AcknowledgeSatisfactionFollowUp(
		fixture.supervisorCtx,
		followUpID,
		"acknowledge-evaluation",
		supervisionreq.AcknowledgeSatisfactionFollowUp{
			ExpectedVersion: followUps[0].Version,
			Note:            "已接收并安排质量核查。",
		},
	)
	if err != nil || acknowledged.Status != supervisionmodel.SatisfactionFollowUpInReview || acknowledged.Version != 2 {
		t.Fatalf("unexpected acknowledge result: %+v err=%v", acknowledged, err)
	}
	if _, err = fixture.service.ResolveSatisfactionFollowUp(
		fixture.supervisorCtx,
		followUpID,
		"resolve-without-boundary",
		supervisionreq.ResolveSatisfactionFollowUp{
			ExpectedVersion: acknowledged.Version,
			Resolution:      "已完成流程核查。",
		},
	); satisfactionDomainCode(err) != supervisionmodel.CodeSatisfactionUsageBoundaryRequired {
		t.Fatalf("resolution without usage-boundary confirmation should fail, got %v", err)
	}
	resolvedRequest := supervisionreq.ResolveSatisfactionFollowUp{
		ExpectedVersion:        acknowledged.Version,
		Resolution:             "已完成流程核查并记录事实。",
		ImprovementAction:      "后续沟通前再次确认时间安排。",
		UsageBoundaryConfirmed: true,
	}
	resolved, err := fixture.service.ResolveSatisfactionFollowUp(
		fixture.supervisorCtx,
		followUpID,
		"resolve-evaluation",
		resolvedRequest,
	)
	if err != nil || resolved.Status != supervisionmodel.SatisfactionFollowUpResolved || resolved.Version != 3 {
		t.Fatalf("unexpected resolve result: %+v err=%v", resolved, err)
	}
	replayedResolve, err := fixture.service.ResolveSatisfactionFollowUp(
		fixture.supervisorCtx,
		followUpID,
		"resolve-evaluation",
		resolvedRequest,
	)
	if err != nil || replayedResolve.ActionID != resolved.ActionID || replayedResolve.Version != resolved.Version {
		t.Fatalf("resolve replay mismatch: result=%+v err=%v", replayedResolve, err)
	}
	detail, err := fixture.service.GetSatisfactionFollowUp(fixture.supervisorCtx, followUpID)
	if err != nil || detail.Status != supervisionmodel.SatisfactionFollowUpResolved || len(detail.Actions) != 2 ||
		detail.Comment != submitRequest.Comment || !detail.Actions[1].UsageBoundaryConfirmed {
		t.Fatalf("unexpected resolved follow-up detail: %+v err=%v", detail, err)
	}
	assertSatisfactionAnonymousJSON(t, detail)
	var todo caseworkmodel.TodoItem
	if err = fixture.db.WithContext(fixture.systemCtx).Where(
		"source_type = ? AND source_id = ?",
		caseworkmodel.TodoSourceSatisfactionFollowUp,
		followUpID,
	).First(&todo).Error; err != nil {
		t.Fatal(err)
	}
	if todo.Status != caseworkmodel.TodoStatusCompleted || todo.ActiveSlot != nil || todo.CompletedAt == nil {
		t.Fatalf("resolved quality follow-up must complete its todo: %+v", todo)
	}
}

func TestSatisfactionHighScoreExpiryAndCompensatingProjection(t *testing.T) {
	fixture := newSatisfactionFixture(t)
	highConsultation := seedSatisfactionConsultation(t, fixture, fixture.now.Add(-time.Hour), caseworkmodel.ConsultationStatusResolved)
	highClosed, err := fixture.caseWork.CloseConsultation(
		fixture.supervisorCtx,
		highConsultation.ID,
		"close-high-score",
		caseworkreq.CloseConsultation{ExpectedVersion: 1, CloseReason: "本轮服务沟通已经完成。"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var highRequest supervisionmodel.SatisfactionRequest
	if err = fixture.db.WithContext(fixture.systemCtx).
		Where("source_event_id = ?", highClosed.InteractionID).First(&highRequest).Error; err != nil {
		t.Fatal(err)
	}
	highResult, err := fixture.service.SubmitClientSatisfactionResponse(
		fixture.clientCtx,
		fixture.clientIdentity,
		highRequest.ID,
		"high-score",
		supervisionreq.SubmitSatisfactionResponse{ExpectedVersion: 1, Rating: 5, Comment: "本次沟通安排清楚。"},
	)
	if err != nil || highResult.FollowUpCreated {
		t.Fatalf("high score should not create a follow-up: result=%+v err=%v", highResult, err)
	}
	assertSatisfactionCount(t, fixture.db, &supervisionmodel.SatisfactionFollowUp{}, "request_id = ?", highRequest.ID, 0)

	oldConsultation := seedSatisfactionConsultation(t, fixture, fixture.now.Add(-9*24*time.Hour), caseworkmodel.ConsultationStatusClosed)
	oldInteraction := caseworkmodel.ConsultationInteraction{
		ConsultationID:   oldConsultation.ID,
		ActionType:       caseworkmodel.ConsultationActionClose,
		ActorType:        caseworkmodel.ConsultationActorStaff,
		ActorID:          fixture.supervisor.ID,
		ActorRole:        caremodel.AuthorityRoleSupervisor,
		Content:          "本轮服务已经关闭",
		FromStatus:       caseworkmodel.ConsultationStatusResolved,
		ToStatus:         caseworkmodel.ConsultationStatusClosed,
		ClientVisible:    true,
		OccurredAt:       fixture.now.Add(-8 * 24 * time.Hour),
		CommandKeyDigest: "historical-close",
		Synthetic:        true,
		DeptId:           fixture.client.DeptId,
	}
	if err = fixture.db.WithContext(fixture.systemCtx).Create(&oldInteraction).Error; err != nil {
		t.Fatal(err)
	}
	if err = fixture.service.ReconcileClientSatisfactionRequests(fixture.clientCtx, fixture.clientIdentity); err != nil {
		t.Fatal(err)
	}
	if err = fixture.service.ReconcileClientSatisfactionRequests(fixture.clientCtx, fixture.clientIdentity); err != nil {
		t.Fatal(err)
	}
	var expired supervisionmodel.SatisfactionRequest
	if err = fixture.db.WithContext(fixture.systemCtx).
		Where("source_event_id = ?", oldInteraction.ID).First(&expired).Error; err != nil {
		t.Fatal(err)
	}
	if expired.Status != supervisionmodel.SatisfactionRequestExpired {
		t.Fatalf("historical invitation should be expired: %+v", expired)
	}
	assertSatisfactionCount(t, fixture.db, &supervisionmodel.SatisfactionRequest{}, "source_event_id = ?", oldInteraction.ID, 1)
	if _, err = fixture.service.SubmitClientSatisfactionResponse(
		fixture.clientCtx,
		fixture.clientIdentity,
		expired.ID,
		"expired-score",
		supervisionreq.SubmitSatisfactionResponse{ExpectedVersion: expired.Version, Rating: 4},
	); satisfactionDomainCode(err) != supervisionmodel.CodeSatisfactionTransitionDenied {
		t.Fatalf("expired invitation should reject submission, got %v", err)
	}
}

func TestSatisfactionProjectionFailureRollsBackConsultationClose(t *testing.T) {
	fixture := newSatisfactionFixture(t)
	if err := fixture.db.WithContext(fixture.systemCtx).Migrator().DropTable(&supervisionmodel.SatisfactionRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.caseWork.CloseConsultation(
		fixture.supervisorCtx,
		fixture.consultation.ID,
		"close-with-missing-projection",
		caseworkreq.CloseConsultation{
			ExpectedVersion: fixture.consultation.Version,
			CloseReason:     "本轮服务沟通已经完成。",
		},
	); err == nil {
		t.Fatal("missing evaluation projection table should fail the close transaction")
	}
	var stored caseworkmodel.Consultation
	if err := fixture.db.WithContext(fixture.systemCtx).First(&stored, fixture.consultation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != caseworkmodel.ConsultationStatusResolved || stored.Version != fixture.consultation.Version || stored.ClosedAt != nil {
		t.Fatalf("failed projection must roll back consultation close: %+v", stored)
	}
	assertSatisfactionCount(t, fixture.db, &caseworkmodel.ConsultationInteraction{}, "consultation_id = ? AND action_type = ?", []any{fixture.consultation.ID, caseworkmodel.ConsultationActionClose}, 0)
}

func newSatisfactionFixture(t *testing.T) satisfactionFixture {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.CareOrgUnitProfile{}, &caremodel.CareAuthorityProfile{},
		&caseworkmodel.Consultation{}, &caseworkmodel.ConsultationInteraction{}, &caseworkmodel.TodoItem{}, &caseworkmodel.CommandReceipt{},
		&supervisionmodel.SatisfactionPolicy{}, &supervisionmodel.SatisfactionRequest{}, &supervisionmodel.SatisfactionResponse{},
		&supervisionmodel.SatisfactionFollowUp{}, &supervisionmodel.SatisfactionFollowUpAction{},
		&platformoutbox.Event{}, &system.SysUser{},
		testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 19, 10, 0, 0, 0, summaryLocation)
	systemCtx := datascope.WithSystem(context.Background())
	policy := supervisionmodel.SatisfactionPolicy{
		Code:              "CONSULTATION-CLOSE-TEST",
		Version:           1,
		TriggerType:       supervisionmodel.SatisfactionTriggerConsultation,
		AnonymityMode:     supervisionmodel.SatisfactionAnonymousStaff,
		ValidForHours:     7 * 24,
		LowScoreThreshold: 2,
		EffectiveFrom:     now.Add(-30 * 24 * time.Hour),
		Status:            supervisionmodel.SatisfactionPolicyStatusPublished,
		Synthetic:         true,
	}
	if err := db.WithContext(systemCtx).Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: 501, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
		{AuthorityID: 502, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: 503, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
	}
	if err := db.WithContext(systemCtx).Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	units := []caremodel.CareOrgUnitProfile{
		{DepartmentID: 100, OrganizationID: 100, Code: "SAT-ORG-A", UnitType: caremodel.OrgUnitTypeOrganization, Synthetic: true, Active: true, DeptId: 100},
		{DepartmentID: 200, OrganizationID: 200, Code: "SAT-ORG-B", UnitType: caremodel.OrgUnitTypeOrganization, Synthetic: true, Active: true, DeptId: 200},
	}
	if err := db.WithContext(systemCtx).Create(&units).Error; err != nil {
		t.Fatal(err)
	}
	users := []system.SysUser{
		{Username: "quality-supervisor-a", NickName: "质量上级甲", AuthorityId: 501, DeptId: 100, Enable: 1},
		{Username: "service-steward-a", NickName: "服务管家甲", AuthorityId: 502, DeptId: 101, Enable: 1},
		{Username: "quality-supervisor-b", NickName: "质量上级乙", AuthorityId: 503, DeptId: 200, Enable: 1},
	}
	if err := db.WithContext(systemCtx).Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	teamID := uint(101)
	client := caremodel.CareClient{
		DisplayCode: "CARE-E001", DisplayName: "林安然", ServiceReason: "日常服务跟进",
		ServicePackageCode: "CARE-TEST", OrganizationID: 100, TeamID: &teamID,
		Status: caremodel.ClientStatusActive, SensitivityLevel: caremodel.SensitivitySensitive,
		Synthetic: true, Version: 1, DeptId: 101,
	}
	if err := db.WithContext(systemCtx).Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	enabled := true
	service := &SupervisionService{DB: db, Now: func() time.Time { return now }, SyntheticFixturesEnabled: &enabled}
	if _, found, err := service.satisfactionPolicyAt(
		db.WithContext(systemCtx).Set("data_scope:skip", true),
		now,
	); err != nil || !found {
		t.Fatalf("evaluation policy must be effective at fixture time: found=%t err=%v", found, err)
	}
	projector := &capturingSatisfactionProjector{service: service}
	caseWork := &caseworkservice.CaseWorkService{
		DB:                          db,
		Now:                         func() time.Time { return now },
		SyntheticFixturesEnabled:    &enabled,
		ConsultationClosedProjector: projector,
	}
	fixture := satisfactionFixture{
		db:              db,
		service:         service,
		caseWork:        caseWork,
		now:             now,
		systemCtx:       systemCtx,
		client:          client,
		clientCtx:       satisfactionClientContext(client),
		clientIdentity:  ClientSatisfactionIdentity{CareClientID: client.ID, DeptID: client.DeptId, Synthetic: true},
		supervisor:      users[0],
		supervisorCtx:   satisfactionStaffContext(users[0], datascope.ScopeDeptAndChild, []uint{100, 101}),
		crossSupervisor: users[2],
		crossCtx:        satisfactionStaffContext(users[2], datascope.ScopeDeptAndChild, []uint{200, 202}),
		steward:         users[1],
		stewardCtx:      satisfactionStaffContext(users[1], datascope.ScopeDept, []uint{101}),
		projector:       projector,
	}
	fixture.consultation = seedSatisfactionConsultation(t, fixture, now.Add(-time.Hour), caseworkmodel.ConsultationStatusResolved)
	return fixture
}

func seedSatisfactionConsultation(
	t *testing.T,
	fixture satisfactionFixture,
	openedAt time.Time,
	status string,
) caseworkmodel.Consultation {
	t.Helper()
	assigneeID := fixture.steward.ID
	consultation := caseworkmodel.Consultation{
		CareClientID: fixture.client.ID, Source: caseworkmodel.ConsultationSourceOnline,
		Subject: "服务安排确认", InitialQuestion: "希望确认本轮服务安排。",
		Urgency: caseworkmodel.ConsultationUrgencyRoutine, Status: status,
		AssigneeID: &assigneeID, AssigneeRole: caremodel.AssignmentRoleCareSteward,
		OpenedAt: openedAt, Resolution: "本轮服务事项已经确认。", Version: 1,
		Synthetic: true, DeptId: fixture.client.DeptId,
	}
	if status == caseworkmodel.ConsultationStatusClosed {
		resolvedAt := openedAt.Add(time.Hour)
		closedAt := openedAt.Add(2 * time.Hour)
		consultation.ResolvedAt = &resolvedAt
		consultation.ClosedAt = &closedAt
		consultation.CloseReason = "本轮服务沟通已经完成。"
	}
	if err := fixture.db.WithContext(fixture.systemCtx).Create(&consultation).Error; err != nil {
		t.Fatal(err)
	}
	if status == caseworkmodel.ConsultationStatusResolved {
		active := caseworkmodel.TodoActiveSlot
		todo := caseworkmodel.TodoItem{
			Category: caseworkmodel.TodoCategoryConsultation, SourceType: caseworkmodel.TodoSourceConsultation,
			SourceID: consultation.ID, ActiveSlot: &active, CareClientID: fixture.client.ID,
			AssigneeID: fixture.steward.ID, AssigneeRole: caremodel.AssignmentRoleCareSteward,
			Status: caseworkmodel.TodoStatusOpen, OpenedAt: openedAt, Version: 1,
			Synthetic: true, DeptId: fixture.client.DeptId,
		}
		if err := fixture.db.WithContext(fixture.systemCtx).Create(&todo).Error; err != nil {
			t.Fatal(err)
		}
	}
	return consultation
}

func satisfactionClientContext(client caremodel.CareClient) context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: client.ID, DeptID: client.DeptId, DeptIDs: []uint{client.DeptId},
		VisibleDeptIDs: []uint{client.DeptId}, Scope: datascope.ScopeDept,
	})
}

func satisfactionStaffContext(user system.SysUser, scope int, visible []uint) context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: user.ID, AuthorityID: user.AuthorityId, DeptID: user.DeptId,
		DeptIDs: []uint{user.DeptId}, VisibleDeptIDs: visible, Scope: scope,
	})
}

func satisfactionDomainCode(err error) int {
	var domainErr *supervisionmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

func assertSatisfactionAnonymousJSON(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, forbidden := range []string{
		"careClientId",
		"sourceId",
		"sourceEventId",
		"serviceAssigneeId",
		"serviceAssigneeRole",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("anonymous employee response exposed %s: %s", forbidden, text)
		}
	}
}

func assertSatisfactionCount(
	t *testing.T,
	db *gorm.DB,
	model any,
	where string,
	args any,
	want int64,
) {
	t.Helper()
	query := db.WithContext(datascope.WithSystem(context.Background())).Model(model)
	if values, ok := args.([]any); ok {
		query = query.Where(where, values...)
	} else {
		query = query.Where(where, args)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%T count=%d want=%d", model, count, want)
	}
}
