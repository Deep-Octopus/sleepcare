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
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

type consultationFixture struct {
	db             *gorm.DB
	service        *CaseWorkService
	now            time.Time
	client         caremodel.CareClient
	clientCtx      context.Context
	clientIdentity ClientConsultationIdentity
	steward        system.SysUser
	stewardCtx     context.Context
	clinician      system.SysUser
	clinicianCtx   context.Context
	supervisor     system.SysUser
	supervisorCtx  context.Context
	crossCtx       context.Context
}

func TestConsultationLifecycleIsIdempotentAndAppendOnly(t *testing.T) {
	fixture := newConsultationFixture(t)
	createRequest := caseworkreq.CreateConsultation{
		Subject: "服务安排咨询",
		Message: "我想确认下一次服务安排。",
		Urgency: caseworkmodel.ConsultationUrgencyRoutine,
	}
	created, err := fixture.service.CreateClientConsultation(
		fixture.clientCtx,
		fixture.clientIdentity,
		"consultation-create",
		createRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != caseworkmodel.ConsultationStatusAssigned || created.Version != 2 || created.InteractionID == 0 {
		t.Fatalf("unexpected create result: %+v", created)
	}
	replayed, err := fixture.service.CreateClientConsultation(
		fixture.clientCtx,
		fixture.clientIdentity,
		"consultation-create",
		createRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ConsultationID != created.ConsultationID || replayed.InteractionID != created.InteractionID ||
		replayed.Status != created.Status || replayed.Version != created.Version ||
		!replayed.OccurredAt.Equal(created.OccurredAt) {
		t.Fatalf("same-key create should replay the original result: first=%+v replayed=%+v", created, replayed)
	}
	changed := createRequest
	changed.Subject = "另一项服务安排"
	if _, err = fixture.service.CreateClientConsultation(
		fixture.clientCtx,
		fixture.clientIdentity,
		"consultation-create",
		changed,
	); consultationDomainCode(err) != caseworkmodel.CodeIdempotencyConflict {
		t.Fatalf("changed replay should be rejected, got %v", err)
	}

	detail, err := fixture.service.GetClientConsultation(
		fixture.clientCtx,
		fixture.clientIdentity,
		created.ConsultationID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Interactions) != 2 || detail.Interactions[0].SenderType != "CLIENT" ||
		detail.Interactions[1].SenderType != "SYSTEM" {
		t.Fatalf("client timeline should include the request and intake acknowledgement: %+v", detail.Interactions)
	}

	replied, err := fixture.service.ReplyConsultation(
		fixture.stewardCtx,
		created.ConsultationID,
		"consultation-reply",
		caseworkreq.ReplyConsultation{
			ExpectedVersion: created.Version,
			Message:         "服务团队已收到，请补充方便联系的时间段。",
			NextStatus:      caseworkmodel.ConsultationStatusWaitingClient,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	message, err := fixture.service.AddClientConsultationMessage(
		fixture.clientCtx,
		fixture.clientIdentity,
		created.ConsultationID,
		"consultation-message",
		caseworkreq.AddClientConsultationMessage{
			ExpectedVersion: replied.Version,
			Message:         "工作日下午均可联系。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if message.Status != caseworkmodel.ConsultationStatusHandling {
		t.Fatalf("client supplement should return the request to handling: %+v", message)
	}
	transferred, err := fixture.service.TransferConsultation(
		fixture.stewardCtx,
		created.ConsultationID,
		"consultation-transfer",
		caseworkreq.TransferConsultation{
			ExpectedVersion:  message.Version,
			TargetAssigneeID: fixture.clinician.ID,
			TargetRole:       caremodel.AssignmentRoleClinician,
			Reason:           "请协同确认服务安排。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	escalated, err := fixture.service.EscalateConsultation(
		fixture.clinicianCtx,
		created.ConsultationID,
		"consultation-escalate",
		caseworkreq.EscalateConsultation{
			ExpectedVersion:  transferred.Version,
			TargetAssigneeID: fixture.supervisor.ID,
			Reason:           "请上级协同确认本次服务安排。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := fixture.service.ResolveConsultation(
		fixture.supervisorCtx,
		created.ConsultationID,
		"consultation-resolve",
		caseworkreq.ResolveConsultation{
			ExpectedVersion: escalated.Version,
			Resolution:      "已确认下一次服务安排。",
			FollowUpPlan:    "按约定时间继续联系。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := fixture.service.CloseConsultation(
		fixture.supervisorCtx,
		created.ConsultationID,
		"consultation-close",
		caseworkreq.CloseConsultation{
			ExpectedVersion: resolved.Version,
			CloseReason:     "本次服务安排已确认。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != caseworkmodel.ConsultationStatusClosed {
		t.Fatalf("unexpected close result: %+v", closed)
	}
	if _, err = fixture.service.AddClientConsultationMessage(
		fixture.clientCtx,
		fixture.clientIdentity,
		created.ConsultationID,
		"consultation-message-after-close",
		caseworkreq.AddClientConsultationMessage{
			ExpectedVersion: closed.Version,
			Message:         "尝试继续补充。",
		},
	); consultationDomainCode(err) != caseworkmodel.CodeConsultationTransitionDenied {
		t.Fatalf("closed consultation should reject client supplements, got %v", err)
	}
	reopened, err := fixture.service.ReopenConsultation(
		fixture.supervisorCtx,
		created.ConsultationID,
		"consultation-reopen",
		caseworkreq.ReopenConsultation{
			ExpectedVersion: closed.Version,
			Reason:          "需要重新确认服务时间。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Status != caseworkmodel.ConsultationStatusAssigned {
		t.Fatalf("unexpected reopen result: %+v", reopened)
	}

	seedCtx := datascope.WithSystem(context.Background())
	var stored caseworkmodel.Consultation
	if err = fixture.db.WithContext(seedCtx).First(&stored, created.ConsultationID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Resolution != "" || stored.FollowUpPlan != "" || stored.CloseReason != "" ||
		stored.ResolvedAt != nil || stored.ClosedAt != nil {
		t.Fatalf("reopen must start a new resolution round without deleting interactions: %+v", stored)
	}
	assertConsultationCount(t, fixture.db, &caseworkmodel.Consultation{}, "id = ?", created.ConsultationID, 1)
	assertConsultationCount(t, fixture.db, &caseworkmodel.ConsultationInteraction{}, "consultation_id = ?", created.ConsultationID, 9)
	assertConsultationCount(t, fixture.db, &caseworkmodel.CommandReceipt{}, "operation = ?", "client-consultation:create", 1)
	assertConsultationCount(t, fixture.db, &platformoutbox.Event{}, "aggregate_id = ?", created.ConsultationID, 9)

	var todos []caseworkmodel.TodoItem
	if err = fixture.db.WithContext(seedCtx).
		Where("source_type = ? AND source_id = ?", caseworkmodel.TodoSourceConsultation, created.ConsultationID).
		Order("id ASC").Find(&todos).Error; err != nil {
		t.Fatal(err)
	}
	if len(todos) != 4 || todos[0].Status != caseworkmodel.TodoStatusSuperseded ||
		todos[1].Status != caseworkmodel.TodoStatusSuperseded || todos[2].Status != caseworkmodel.TodoStatusCompleted ||
		todos[3].Status != caseworkmodel.TodoStatusOpen || todos[3].ActiveSlot == nil {
		t.Fatalf("assignment changes, close and reopen must retain todo history with one active row: %+v", todos)
	}
}

func TestConsultationPermissionStateAndOptionsFailClosed(t *testing.T) {
	fixture := newConsultationFixture(t)
	created, err := fixture.service.CreateClientConsultation(
		fixture.clientCtx,
		fixture.clientIdentity,
		"permission-create",
		caseworkreq.CreateConsultation{
			Subject: "联系时间咨询",
			Message: "请确认方便联系的时间。",
			Urgency: caseworkmodel.ConsultationUrgencyExpedited,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.CloseConsultation(
		fixture.stewardCtx,
		created.ConsultationID,
		"close-without-resolution",
		caseworkreq.CloseConsultation{ExpectedVersion: created.Version, CloseReason: "尝试提前关闭。"},
	); consultationDomainCode(err) != caseworkmodel.CodeConsultationResolutionRequired {
		t.Fatalf("close without a resolution should be rejected, got %v", err)
	}
	if _, err = fixture.service.ReplyConsultation(
		fixture.clinicianCtx,
		created.ConsultationID,
		"non-assignee-reply",
		caseworkreq.ReplyConsultation{
			ExpectedVersion: created.Version,
			Message:         "尝试越过当前责任人回复。",
			NextStatus:      caseworkmodel.ConsultationStatusHandling,
		},
	); consultationDomainCode(err) != caseworkmodel.CodeConsultationAssigneeRequired {
		t.Fatalf("non-assignee reply should be rejected, got %v", err)
	}
	if _, err = fixture.service.GetConsultation(fixture.crossCtx, created.ConsultationID); consultationDomainCode(err) != caseworkmodel.CodeAccessScopeDenied {
		t.Fatalf("cross-department detail should fail closed, got %v", err)
	}
	list, total, err := fixture.service.ListConsultations(fixture.crossCtx, caseworkreq.ConsultationSearch{})
	if err != nil || total != 0 || len(list) != 0 {
		t.Fatalf("cross-department list should be empty: total=%d list=%+v err=%v", total, list, err)
	}
	options, err := fixture.service.ListConsultationAssigneeOptions(fixture.stewardCtx, created.ConsultationID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]uint{
		caremodel.AssignmentRoleCareSteward: fixture.steward.ID,
		caremodel.AssignmentRoleClinician:   fixture.clinician.ID,
		caremodel.AuthorityRoleSupervisor:   fixture.supervisor.ID,
	}
	if len(options) != len(want) {
		t.Fatalf("unexpected assignee options: %+v", options)
	}
	for _, option := range options {
		if want[option.RoleType] != option.ID || option.DisplayName == "" {
			t.Fatalf("option escaped the active responsibility chain: %+v", option)
		}
	}
}

func TestConsultationWithoutStewardWaitsForSupervisorAssignment(t *testing.T) {
	fixture := newConsultationFixture(t)
	seedCtx := datascope.WithSystem(context.Background())
	if err := fixture.db.WithContext(seedCtx).
		Where("care_client_id = ? AND role_type = ?", fixture.client.ID, caremodel.AssignmentRoleCareSteward).
		Delete(&caremodel.CareAssignment{}).Error; err != nil {
		t.Fatal(err)
	}
	created, err := fixture.service.CreateClientConsultation(
		fixture.clientCtx,
		fixture.clientIdentity,
		"unassigned-create",
		caseworkreq.CreateConsultation{
			Subject: "服务流程咨询",
			Message: "请协助确认后续服务步骤。",
			Urgency: caseworkmodel.ConsultationUrgencyRoutine,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != caseworkmodel.ConsultationStatusWaitingAssignment || created.Version != 1 {
		t.Fatalf("consultation without a steward should remain waiting for assignment: %+v", created)
	}
	assertConsultationCount(t, fixture.db, &caseworkmodel.TodoItem{}, "source_id = ?", created.ConsultationID, 0)
	assigned, err := fixture.service.AssignConsultation(
		fixture.supervisorCtx,
		created.ConsultationID,
		"supervisor-assignment",
		caseworkreq.AssignConsultation{
			ExpectedVersion:  created.Version,
			TargetAssigneeID: fixture.clinician.ID,
			TargetRole:       caremodel.AssignmentRoleClinician,
			Reason:           "由当前责任医护接收。",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Status != caseworkmodel.ConsultationStatusAssigned || assigned.Version != 2 {
		t.Fatalf("unexpected assignment result: %+v", assigned)
	}
	assertConsultationCount(t, fixture.db, &caseworkmodel.TodoItem{}, "source_id = ?", created.ConsultationID, 1)
}

func newConsultationFixture(t *testing.T) consultationFixture {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.CareAuthorityProfile{},
		&caseworkmodel.Consultation{}, &caseworkmodel.ConsultationInteraction{},
		&caseworkmodel.TodoItem{}, &caseworkmodel.CommandReceipt{},
		&platformoutbox.Event{}, &system.SysUser{},
		testutil.WithDataScopeCallbacks(),
	)
	now := time.Date(2026, time.August, 19, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	enabled := true
	service := &CaseWorkService{DB: db, Now: func() time.Time { return now }, SyntheticFixturesEnabled: &enabled}
	seedCtx := datascope.WithSystem(context.Background())

	client := caremodel.CareClient{
		DisplayCode: "CARE-T001", DisplayName: "林安然", ServiceReason: "日常服务跟进",
		ServicePackageCode: "CARE-TEST", OrganizationID: 100, Status: caremodel.ClientStatusActive,
		SensitivityLevel: caremodel.SensitivitySensitive, Synthetic: true, Version: 1, DeptId: 101,
	}
	if err := db.WithContext(seedCtx).Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	users := []system.SysUser{
		{Username: "consultation-steward", NickName: "健康管家林青", AuthorityId: 502, DeptId: 101, Enable: 1},
		{Username: "consultation-clinician", NickName: "责任医护周宁", AuthorityId: 503, DeptId: 101, Enable: 1},
		{Username: "consultation-supervisor", NickName: "上级医师王宁", AuthorityId: 504, DeptId: 100, Enable: 1},
		{Username: "consultation-cross", NickName: "其他机构上级", AuthorityId: 505, DeptId: 200, Enable: 1},
	}
	if err := db.WithContext(seedCtx).Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: 502, RoleType: caremodel.AuthorityRoleCareSteward, Synthetic: true, Active: true},
		{AuthorityID: 503, RoleType: caremodel.AuthorityRoleClinician, Synthetic: true, Active: true},
		{AuthorityID: 504, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
		{AuthorityID: 505, RoleType: caremodel.AuthorityRoleSupervisor, Synthetic: true, Active: true},
	}
	if err := db.WithContext(seedCtx).Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	assignments := []caremodel.CareAssignment{
		{
			CareClientID: client.ID, OrganizationID: client.OrganizationID, TeamID: client.DeptId,
			AssigneeID: users[0].ID, RoleType: caremodel.AssignmentRoleCareSteward,
			ValidFrom: now.Add(-time.Hour), Reason: "当前服务责任", Synthetic: true, DeptId: client.DeptId,
		},
		{
			CareClientID: client.ID, OrganizationID: client.OrganizationID, TeamID: client.DeptId,
			AssigneeID: users[1].ID, RoleType: caremodel.AssignmentRoleClinician,
			ValidFrom: now.Add(-time.Hour), Reason: "当前服务责任", Synthetic: true, DeptId: client.DeptId,
		},
	}
	if err := db.WithContext(seedCtx).Create(&assignments).Error; err != nil {
		t.Fatal(err)
	}
	clientCtx := datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: client.ID, DeptID: client.DeptId, DeptIDs: []uint{client.DeptId},
		VisibleDeptIDs: []uint{client.DeptId}, Scope: datascope.ScopeDept,
	})
	return consultationFixture{
		db:        db,
		service:   service,
		now:       now,
		client:    client,
		clientCtx: clientCtx,
		clientIdentity: ClientConsultationIdentity{
			CareClientID: client.ID,
			DeptID:       client.DeptId,
			Synthetic:    true,
		},
		steward:       users[0],
		stewardCtx:    consultationStaffContext(users[0], datascope.ScopeDept, []uint{101}),
		clinician:     users[1],
		clinicianCtx:  consultationStaffContext(users[1], datascope.ScopeDept, []uint{101}),
		supervisor:    users[2],
		supervisorCtx: consultationStaffContext(users[2], datascope.ScopeDeptAndChild, []uint{100, 101}),
		crossCtx:      consultationStaffContext(users[3], datascope.ScopeDeptAndChild, []uint{200, 201}),
	}
}

func consultationStaffContext(user system.SysUser, scope int, visible []uint) context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: user.ID, AuthorityID: user.AuthorityId, DeptID: user.DeptId,
		DeptIDs: []uint{user.DeptId}, VisibleDeptIDs: visible, Scope: scope,
	})
}

func consultationDomainCode(err error) int {
	var domainErr *caseworkmodel.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return 0
}

func assertConsultationCount(t *testing.T, db *gorm.DB, model any, where string, arg any, want int64) {
	t.Helper()
	var count int64
	if err := db.WithContext(datascope.WithSystem(context.Background())).Model(model).
		Where(where, arg).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%T count=%d, want %d", model, count, want)
	}
}
