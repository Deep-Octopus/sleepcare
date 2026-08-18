package careclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	carereq "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
	"gorm.io/gorm"
)

const (
	testOrgA       = 10
	testTeamA      = 11
	testOrgB       = 20
	testTeamB      = 21
	testStewardA   = 101
	testStewardA2  = 102
	testSupervisor = 103
	testStewardB   = 104
	roleSteward    = 201
	roleSupervisor = 203
)

var testNow = time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

func newCareServiceTest(t *testing.T) (*CareClientService, *gorm.DB) {
	t.Helper()
	db := testutil.NewMemoryDB(t,
		&caremodel.CareClient{}, &caremodel.CareAssignment{}, &caremodel.ConsentRecord{},
		&caremodel.CareOrgUnitProfile{}, &caremodel.CareAuthorityProfile{}, &caremodel.CareClientCommandReceipt{},
		&system.SysUser{}, &system.SysAuthority{}, &system.SysDepartment{}, &system.SysUserAuthority{}, &system.SysUserDepartment{},
		testutil.WithDataScopeCallbacks(),
	)
	systemCtx := datascope.WithSystem(context.Background())
	active := true
	departments := []system.SysDepartment{
		{GVA_MODEL: modelID(testOrgA), Name: "[测试] 机构A", Status: &active},
		{GVA_MODEL: modelID(testTeamA), Name: "[测试] 团队A", ParentId: testOrgA, Status: &active},
		{GVA_MODEL: modelID(testOrgB), Name: "[测试] 机构B", Status: &active},
		{GVA_MODEL: modelID(testTeamB), Name: "[测试] 团队B", ParentId: testOrgB, Status: &active},
	}
	if err := db.WithContext(systemCtx).Create(&departments).Error; err != nil {
		t.Fatal(err)
	}
	authorities := []system.SysAuthority{
		{AuthorityId: roleSteward, AuthorityName: "管家", DataScope: datascope.ScopeDept},
		{AuthorityId: roleSupervisor, AuthorityName: "督导", DataScope: datascope.ScopeDeptAndChild},
	}
	if err := db.WithContext(systemCtx).Create(&authorities).Error; err != nil {
		t.Fatal(err)
	}
	profiles := []caremodel.CareAuthorityProfile{
		{AuthorityID: roleSteward, RoleType: caremodel.AuthorityRoleCareSteward, Active: true, Synthetic: true},
		{AuthorityID: roleSupervisor, RoleType: caremodel.AuthorityRoleSupervisor, Active: true, Synthetic: true},
	}
	if err := db.WithContext(systemCtx).Create(&profiles).Error; err != nil {
		t.Fatal(err)
	}
	orgProfiles := []caremodel.CareOrgUnitProfile{
		{DepartmentID: testOrgA, OrganizationID: testOrgA, Code: "ORG-A", UnitType: caremodel.OrgUnitTypeOrganization, Active: true, Synthetic: true, DeptId: testOrgA},
		{DepartmentID: testTeamA, OrganizationID: testOrgA, Code: "TEAM-A", UnitType: caremodel.OrgUnitTypeTeam, Active: true, Synthetic: true, DeptId: testTeamA},
		{DepartmentID: testOrgB, OrganizationID: testOrgB, Code: "ORG-B", UnitType: caremodel.OrgUnitTypeOrganization, Active: true, Synthetic: true, DeptId: testOrgB},
		{DepartmentID: testTeamB, OrganizationID: testOrgB, Code: "TEAM-B", UnitType: caremodel.OrgUnitTypeTeam, Active: true, Synthetic: true, DeptId: testTeamB},
	}
	if err := db.WithContext(systemCtx).Create(&orgProfiles).Error; err != nil {
		t.Fatal(err)
	}
	users := []system.SysUser{
		{GVA_MODEL: modelID(testStewardA), Username: "steward-a", NickName: "管家甲", AuthorityId: roleSteward, DeptId: testTeamA, Enable: 1},
		{GVA_MODEL: modelID(testStewardA2), Username: "steward-a2", NickName: "管家乙", AuthorityId: roleSteward, DeptId: testTeamA, Enable: 1},
		{GVA_MODEL: modelID(testSupervisor), Username: "supervisor", NickName: "督导", AuthorityId: roleSupervisor, DeptId: testOrgA, Enable: 1},
		{GVA_MODEL: modelID(testStewardB), Username: "steward-b", NickName: "跨机构管家", AuthorityId: roleSteward, DeptId: testTeamB, Enable: 1},
	}
	if err := db.WithContext(systemCtx).Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	return &CareClientService{DB: db, Now: func() time.Time { return testNow }}, db
}

func TestCareClientAccessPolicyIsFailClosedAndResponsibilityScoped(t *testing.T) {
	service, db := newCareServiceTest(t)
	systemCtx := datascope.WithSystem(context.Background())
	teamA, teamB := uint(testTeamA), uint(testTeamB)
	clients := []caremodel.CareClient{
		{DisplayCode: "SYN-A1", DisplayName: "[测试] A1", OrganizationID: testOrgA, TeamID: &teamA, Status: caremodel.ClientStatusActive, Synthetic: true, Version: 1, DeptId: testTeamA},
		{DisplayCode: "SYN-A2", DisplayName: "[测试] A2", OrganizationID: testOrgA, TeamID: &teamA, Status: caremodel.ClientStatusActive, Synthetic: true, Version: 1, DeptId: testTeamA},
		{DisplayCode: "SYN-B1", DisplayName: "[测试] B1", OrganizationID: testOrgB, TeamID: &teamB, Status: caremodel.ClientStatusActive, Synthetic: true, Version: 1, DeptId: testTeamB},
	}
	if err := db.WithContext(systemCtx).Create(&clients).Error; err != nil {
		t.Fatal(err)
	}
	assignments := []caremodel.CareAssignment{
		{CareClientID: clients[0].ID, OrganizationID: testOrgA, TeamID: testTeamA, AssigneeID: testStewardA, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: testNow.Add(-time.Hour), Reason: "测试", Synthetic: true, DeptId: testTeamA},
		{CareClientID: clients[1].ID, OrganizationID: testOrgA, TeamID: testTeamA, AssigneeID: testStewardA2, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: testNow.Add(-time.Hour), Reason: "测试", Synthetic: true, DeptId: testTeamA},
		{CareClientID: clients[2].ID, OrganizationID: testOrgB, TeamID: testTeamB, AssigneeID: testStewardB, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: testNow.Add(-time.Hour), Reason: "测试", Synthetic: true, DeptId: testTeamB},
	}
	if err := db.WithContext(systemCtx).Create(&assignments).Error; err != nil {
		t.Fatal(err)
	}

	stewardCtx := identityContext(testStewardA, roleSteward, testTeamA, datascope.ScopeDept, []uint{testTeamA})
	list, total, err := service.List(stewardCtx, carereq.CareClientSearch{})
	if err != nil || total != 1 || len(list) != 1 || list[0].DisplayCode != "SYN-A1" {
		t.Fatalf("steward responsibility scope list=%v total=%d err=%v", list, total, err)
	}
	if _, err = service.Get(stewardCtx, clients[1].ID); !isDomainCode(err, caremodel.CodeResourceNotFound) {
		t.Fatalf("same-team unassigned client should be hidden, got %v", err)
	}

	adminCtx := identityContext(1, 888, 0, datascope.ScopeAll, nil)
	if _, _, err = service.List(adminCtx, carereq.CareClientSearch{}); !isDomainCode(err, caremodel.CodeAccessScopeDenied) {
		t.Fatalf("plain admin must fail closed, got %v", err)
	}

	supervisorCtx := identityContext(testSupervisor, roleSupervisor, testOrgA, datascope.ScopeDeptAndChild, []uint{testOrgA, testTeamA})
	list, total, err = service.List(supervisorCtx, carereq.CareClientSearch{})
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("supervisor should see only org A subtree, total=%d len=%d err=%v", total, len(list), err)
	}
}

func TestCareClientWritesAreIdempotentVersionedAndAppendHistory(t *testing.T) {
	service, db := newCareServiceTest(t)
	ctx := identityContext(testSupervisor, roleSupervisor, testOrgA, datascope.ScopeDeptAndChild, []uint{testOrgA, testTeamA})
	teamA := uint(testTeamA)
	createReq := carereq.CreateCareClient{DisplayCode: "SYN-NEW", DisplayName: "[测试] 新用户", OrganizationID: testOrgA, TeamID: &teamA, Synthetic: true}
	created, err := service.Create(ctx, "create-key", createReq)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Create(ctx, "create-key", createReq)
	if err != nil || replayed != created {
		t.Fatalf("idempotent replay = %+v err=%v, want %+v", replayed, err, created)
	}
	changed := createReq
	changed.DisplayName = "[测试] 另一个请求"
	if _, err = service.Create(ctx, "create-key", changed); !isDomainCode(err, caremodel.CodeIdempotencyConflict) {
		t.Fatalf("different payload with same key should conflict, got %v", err)
	}

	realClient := caremodel.CareClient{DisplayCode: "REAL-LOCKED", DisplayName: "正式数据占位", OrganizationID: testOrgA, TeamID: &teamA, Status: caremodel.ClientStatusActive, Version: 1, DeptId: testTeamA}
	if err = db.WithContext(datascope.WithSystem(context.Background())).Create(&realClient).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = service.Update(ctx, realClient.ID, "real-update", carereq.UpdateCareClient{ExpectedVersion: 1, DisplayName: "[测试] 不应写入", TeamID: &teamA, Status: caremodel.ClientStatusActive}); !isDomainCode(err, caremodel.CodeOperationNotAllowed) {
		t.Fatalf("P1-02 must reject writes to non-synthetic clients, got %v", err)
	}

	initial := caremodel.CareAssignment{CareClientID: created.CareClientID, OrganizationID: testOrgA, TeamID: testTeamA, AssigneeID: testStewardA, RoleType: caremodel.AssignmentRoleCareSteward, ValidFrom: testNow.Add(-time.Hour), Reason: "测试初始责任", Synthetic: true, DeptId: testTeamA}
	if err = db.WithContext(datascope.WithSystem(context.Background())).Create(&initial).Error; err != nil {
		t.Fatal(err)
	}
	crossTeamReq := carereq.CreateAssignment{ExpectedVersion: 1, RoleType: caremodel.AssignmentRoleCareSteward, AssigneeID: testStewardB, TeamID: testTeamB, ValidFrom: testNow, ReplacesAssignmentID: &initial.ID, Reason: "不允许的跨团队转交"}
	if _, err = service.CreateAssignment(ctx, created.CareClientID, "cross-team", crossTeamReq); !isDomainCode(err, caremodel.CodeOperationNotAllowed) {
		t.Fatalf("P1-02 must reject cross-team assignment, got %v", err)
	}
	assignmentReq := carereq.CreateAssignment{ExpectedVersion: 1, RoleType: caremodel.AssignmentRoleCareSteward, AssigneeID: testStewardA2, TeamID: testTeamA, ValidFrom: testNow, ReplacesAssignmentID: &initial.ID, Reason: "固定测试转交"}
	assigned, err := service.CreateAssignment(ctx, created.CareClientID, "assignment-key", assignmentReq)
	if err != nil || assigned.Version != 2 {
		t.Fatalf("assignment result=%+v err=%v", assigned, err)
	}
	var old caremodel.CareAssignment
	if err = db.WithContext(datascope.WithSystem(context.Background())).First(&old, initial.ID).Error; err != nil || old.EndedAt == nil {
		t.Fatalf("replaced assignment must remain with end fact: %+v err=%v", old, err)
	}
	var count int64
	db.WithContext(datascope.WithSystem(context.Background())).Model(&caremodel.CareAssignment{}).Where("care_client_id = ?", created.CareClientID).Count(&count)
	if count != 2 {
		t.Fatalf("assignment history count=%d, want 2", count)
	}

	consentReq := carereq.CreateConsentRecord{ExpectedVersion: 2, ConsentType: caremodel.ConsentTypeSyntheticTestParticipation, Action: caremodel.ConsentActionGrant, TextVersion: "SYNTHETIC-V1", OccurredAt: testNow, Source: caremodel.ConsentSourceStaffRecorded, Reason: "固定测试授权"}
	consented, err := service.CreateConsent(ctx, created.CareClientID, "consent-key", consentReq)
	if err != nil || consented.Version != 3 {
		t.Fatalf("consent result=%+v err=%v", consented, err)
	}
	if _, err = service.CreateConsent(ctx, created.CareClientID, "second-grant", consentReq); !isDomainCode(err, caremodel.CodeVersionConflict) {
		t.Fatalf("stale version must conflict before duplicate grant, got %v", err)
	}
	consentReq.ExpectedVersion = 3
	if _, err = service.CreateConsent(ctx, created.CareClientID, "second-grant-current", consentReq); !isDomainCode(err, caremodel.CodeOperationNotAllowed) {
		t.Fatalf("active grant cannot be duplicated, got %v", err)
	}
	consentReq.ExpectedVersion = 3
	consentReq.Action = caremodel.ConsentActionWithdraw
	consentReq.OccurredAt = testNow.Add(time.Minute)
	withdrawn, err := service.CreateConsent(ctx, created.CareClientID, "withdraw-key", consentReq)
	if err != nil || withdrawn.Version != 4 {
		t.Fatalf("withdraw result=%+v err=%v", withdrawn, err)
	}
	db.WithContext(datascope.WithSystem(context.Background())).Model(&caremodel.ConsentRecord{}).Where("care_client_id = ?", created.CareClientID).Count(&count)
	if count != 2 {
		t.Fatalf("consent history count=%d, want 2", count)
	}
}

func identityContext(userID, authorityID, deptID uint, scope int, visible []uint) context.Context {
	return datascope.WithIdentity(context.Background(), &datascope.Identity{
		UserID: userID, AuthorityID: authorityID, DeptID: deptID, DeptIDs: []uint{deptID}, VisibleDeptIDs: visible, Scope: scope,
	})
}

func isDomainCode(err error, code int) bool {
	var domainErr *caremodel.DomainError
	return errors.As(err, &domainErr) && domainErr.Code == code
}

func modelID(id uint) global.GVA_MODEL { return global.GVA_MODEL{ID: id} }
