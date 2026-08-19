package careclient

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	carereq "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/datascope"
)

func TestDataGovernanceReadinessIsSupervisorOnlyAndFailClosed(t *testing.T) {
	service, _ := newCareServiceTest(t)
	fixturesEnabled := true
	service.SyntheticFixturesEnabled = &fixturesEnabled
	service.DataGovernance = &config.DataGovernance{Mode: caremodel.DataGovernanceModeDisabled}

	supervisorCtx := identityContext(testSupervisor, roleSupervisor, testOrgA, datascope.ScopeDeptAndChild, []uint{testOrgA, testTeamA})
	readiness, err := service.GetDataGovernanceReadiness(supervisorCtx)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Mode != caremodel.DataGovernanceModeDisabled || readiness.UsageScope != caremodel.DataGovernanceUsageTestOnly {
		t.Fatalf("unexpected readiness identity: %+v", readiness)
	}
	if readiness.RealDataEnabled || readiness.FormalConsentEnabled || readiness.LifecycleExecutionEnabled || readiness.RequestRecordingEnabled {
		t.Fatalf("disabled governance must fail closed: %+v", readiness)
	}
	if len(readiness.ConsentRequirements) != 4 || len(readiness.ReviewGates) != 10 {
		t.Fatalf("readiness must expose four consent requirements and ten review gates: %+v", readiness)
	}
	for _, requirement := range readiness.ConsentRequirements {
		if requirement.ContentReviewed || requirement.PolicyVersion != "" || requirement.RecordingEnabled {
			t.Fatalf("unapproved consent requirement must stay closed: %+v", requirement)
		}
	}
	for _, required := range []string{
		"REAL_DATA_MODE_UNAVAILABLE",
		"FORMAL_CONSENT_RECORDING_UNAVAILABLE",
		"LIFECYCLE_EXECUTION_UNAVAILABLE",
		"CONTRACT_TEST_RECORDING_DISABLED",
	} {
		if !containsString(readiness.BlockingItems, required) {
			t.Fatalf("blocking item %q missing from %v", required, readiness.BlockingItems)
		}
	}

	stewardCtx := identityContext(testStewardA, roleSteward, testTeamA, datascope.ScopeDept, []uint{testTeamA})
	if _, err = service.GetDataGovernanceReadiness(stewardCtx); !isDomainCode(err, caremodel.CodeAccessScopeDenied) {
		t.Fatalf("non-supervisor readiness access must be denied, got %v", err)
	}
}

func TestDataLifecycleRequestsAreAppendOnlyTestRecords(t *testing.T) {
	service, db := newCareServiceTest(t)
	fixturesEnabled := true
	service.SyntheticFixturesEnabled = &fixturesEnabled
	service.DataGovernance = &config.DataGovernance{Mode: caremodel.DataGovernanceModeContractTest}
	systemCtx := datascope.WithSystem(context.Background())
	supervisorCtx := identityContext(testSupervisor, roleSupervisor, testOrgA, datascope.ScopeDeptAndChild, []uint{testOrgA, testTeamA})
	teamA, teamB := uint(testTeamA), uint(testTeamB)
	clients := []caremodel.CareClient{
		{
			DisplayCode:    "LIFECYCLE-A",
			DisplayName:    "林清远",
			OrganizationID: testOrgA,
			TeamID:         &teamA,
			Status:         caremodel.ClientStatusActive,
			Synthetic:      true,
			Version:        1,
			DeptId:         testTeamA,
		},
		{
			DisplayCode:    "LIFECYCLE-B",
			DisplayName:    "周安和",
			OrganizationID: testOrgB,
			TeamID:         &teamB,
			Status:         caremodel.ClientStatusActive,
			Synthetic:      true,
			Version:        1,
			DeptId:         testTeamB,
		},
		{
			DisplayCode:    "LIFECYCLE-REJECT",
			DisplayName:    "拒绝路径占位",
			OrganizationID: testOrgA,
			TeamID:         &teamA,
			Status:         caremodel.ClientStatusActive,
			Synthetic:      false,
			Version:        1,
			DeptId:         testTeamA,
		},
	}
	if err := db.WithContext(systemCtx).Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	requestTypes := []string{
		caremodel.LifecycleRequestAccessCopy,
		caremodel.LifecycleRequestCorrection,
		caremodel.LifecycleRequestRestriction,
		caremodel.LifecycleRequestErasure,
	}
	firstRequest := carereq.CreateDataLifecycleRequest{
		ExpectedVersion: 1,
		RequestType:     requestTypes[0],
		RequestedAt:     testNow,
		Source:          caremodel.LifecycleRequestSourceStaffRecorded,
		Reason:          "用户提出查看已记录信息的请求",
	}
	first, err := service.CreateDataLifecycleRequest(supervisorCtx, clients[0].ID, "lifecycle-1", firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.CreateDataLifecycleRequest(supervisorCtx, clients[0].ID, "lifecycle-1", firstRequest)
	if err != nil || replayed != first {
		t.Fatalf("idempotent replay=%+v err=%v, want %+v", replayed, err, first)
	}
	changed := firstRequest
	changed.Reason = "同一幂等键的不同请求"
	if _, err = service.CreateDataLifecycleRequest(supervisorCtx, clients[0].ID, "lifecycle-1", changed); !isDomainCode(err, caremodel.CodeIdempotencyConflict) {
		t.Fatalf("changed request with same key must conflict, got %v", err)
	}

	version := first.Version
	for index, requestType := range requestTypes[1:] {
		result, createErr := service.CreateDataLifecycleRequest(
			supervisorCtx,
			clients[0].ID,
			"lifecycle-"+requestType,
			carereq.CreateDataLifecycleRequest{
				ExpectedVersion: version,
				RequestType:     requestType,
				RequestedAt:     testNow.Add(time.Duration(index+1) * time.Minute),
				Source:          caremodel.LifecycleRequestSourceStaffRecorded,
				Reason:          "用户提出数据生命周期请求",
			},
		)
		if createErr != nil {
			t.Fatalf("create %s: %v", requestType, createErr)
		}
		version = result.Version
	}

	items, total, err := service.ListDataLifecycleRequests(
		supervisorCtx,
		clients[0].ID,
		carereq.DataLifecycleRequestSearch{RequestType: caremodel.LifecycleRequestErasure},
	)
	if err != nil || total != 1 || len(items) != 1 || items[0].RequestType != caremodel.LifecycleRequestErasure {
		t.Fatalf("filtered lifecycle requests=%+v total=%d err=%v", items, total, err)
	}
	if items[0].Status != caremodel.LifecycleRequestStatusPendingPolicy ||
		items[0].IdentityVerificationStatus != caremodel.IdentityVerificationStatusNotConfigured ||
		items[0].ExecutionAllowed {
		t.Fatalf("request must remain non-executable and pending policy: %+v", items[0])
	}
	if _, _, err = service.ListDataLifecycleRequests(
		supervisorCtx,
		clients[0].ID,
		carereq.DataLifecycleRequestSearch{RequestType: "UNKNOWN"},
	); !isDomainCode(err, caremodel.CodeLifecycleRequestInvalid) {
		t.Fatalf("unknown request type must be rejected, got %v", err)
	}

	var storedClient caremodel.CareClient
	if err = db.WithContext(systemCtx).First(&storedClient, clients[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedClient.Version != 5 || storedClient.Status != caremodel.ClientStatusActive || storedClient.DeletedAt.Valid {
		t.Fatalf("client must remain active and undeleted with version 5: %+v", storedClient)
	}
	var records []caremodel.DataLifecycleRequest
	if err = db.WithContext(systemCtx).Where("care_client_id = ?", clients[0].ID).Order("id").Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 {
		t.Fatalf("append-only request count=%d, want 4", len(records))
	}
	for _, record := range records {
		if !record.Synthetic || record.DeptId != testTeamA || record.CreatedBy != testSupervisor ||
			record.PolicySnapshotDigest == "" || record.ExecutionAllowed {
			t.Fatalf("unsafe stored lifecycle request: %+v", record)
		}
	}

	if _, err = service.CreateDataLifecycleRequest(
		supervisorCtx,
		clients[1].ID,
		"cross-org",
		firstRequest,
	); !isDomainCode(err, caremodel.CodeResourceNotFound) {
		t.Fatalf("cross-organization request must be hidden, got %v", err)
	}
	if _, err = service.CreateDataLifecycleRequest(
		supervisorCtx,
		clients[2].ID,
		"real-record",
		firstRequest,
	); !isDomainCode(err, caremodel.CodeOperationNotAllowed) {
		t.Fatalf("non-test client request must be rejected, got %v", err)
	}
	stewardCtx := identityContext(testStewardA, roleSteward, testTeamA, datascope.ScopeDept, []uint{testTeamA})
	if _, err = service.CreateDataLifecycleRequest(
		stewardCtx,
		clients[0].ID,
		"steward-record",
		firstRequest,
	); !isDomainCode(err, caremodel.CodeAccessScopeDenied) {
		t.Fatalf("non-supervisor request must be denied, got %v", err)
	}

	service.DataGovernance = &config.DataGovernance{Mode: caremodel.DataGovernanceModeDisabled}
	replayedAfterDisable, err := service.CreateDataLifecycleRequest(supervisorCtx, clients[0].ID, "lifecycle-1", firstRequest)
	if err != nil || replayedAfterDisable != first {
		t.Fatalf("exact replay must remain stable after gate closes: %+v err=%v", replayedAfterDisable, err)
	}
	blocked := firstRequest
	blocked.ExpectedVersion = storedClient.Version
	if _, err = service.CreateDataLifecycleRequest(supervisorCtx, clients[0].ID, "disabled-new", blocked); !isDomainCode(err, caremodel.CodeDataGovernanceDisabled) {
		t.Fatalf("new request must be rejected after gate closes, got %v", err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
