package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

type phaseTwoAcceptanceContract struct {
	SchemaVersion       string                         `json:"schemaVersion"`
	Task                string                         `json:"task"`
	Environment         string                         `json:"environment"`
	DataScope           string                         `json:"dataScope"`
	EngineeringDecision string                         `json:"engineeringDecision"`
	RealTrialEnabled    bool                           `json:"realTrialEnabled"`
	PromotionAllowed    bool                           `json:"promotionAllowed"`
	Capabilities        phaseTwoAcceptanceCapabilities `json:"capabilities"`
	Operations          phaseTwoAcceptanceOperations   `json:"operations"`
	Checks              []phaseTwoAcceptanceCheck      `json:"checks"`
	UATCases            []phaseTwoAcceptanceUATCase    `json:"uatCases"`
	KnownDebts          []phaseTwoAcceptanceKnownDebt  `json:"knownDebts"`
	Gates               []phaseTwoAcceptanceGate       `json:"gates"`
}

type phaseTwoAcceptanceCapabilities struct {
	RealDataEnabled           bool `json:"realDataEnabled"`
	FormalNotificationEnabled bool `json:"formalNotificationEnabled"`
	StaffAIShadowEnabled      bool `json:"staffAIShadowEnabled"`
	UserFacingAIEnabled       bool `json:"userFacingAIEnabled"`
	ExternalCallsEnabled      bool `json:"externalCallsEnabled"`
}

type phaseTwoAcceptanceOperations struct {
	DeploymentExecuted        bool `json:"deploymentExecuted"`
	TrafficSwitchExecuted     bool `json:"trafficSwitchExecuted"`
	DatabaseRestoreExecuted   bool `json:"databaseRestoreExecuted"`
	VolumeResetExecuted       bool `json:"volumeResetExecuted"`
	GitHistoryRewriteExecuted bool `json:"gitHistoryRewriteExecuted"`
	ExternalCallsExecuted     bool `json:"externalCallsExecuted"`
}

type phaseTwoAcceptanceCheck struct {
	ID       string `json:"id"`
	Scope    string `json:"scope"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type phaseTwoAcceptanceUATCase struct {
	ID       string   `json:"id"`
	Scope    string   `json:"scope"`
	Status   string   `json:"status"`
	Evidence []string `json:"evidence"`
}

type phaseTwoAcceptanceKnownDebt struct {
	ID     string `json:"id"`
	Scope  string `json:"scope"`
	Area   string `json:"area"`
	Status string `json:"status"`
}

type phaseTwoAcceptanceGate struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Owner  string `json:"owner"`
}

func TestPhaseTwoAcceptanceContract(t *testing.T) {
	content, err := os.ReadFile("../../docs/contracts/phase2-acceptance.json")
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var contract phaseTwoAcceptanceContract
	if err = decoder.Decode(&contract); err != nil {
		t.Fatalf("phase-two acceptance contract is invalid: %v", err)
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("phase-two acceptance contract has trailing content: %v", err)
	}

	assertPhaseTwoAcceptanceHeader(t, contract)
	assertPhaseTwoAcceptanceChecks(t, contract.Checks)
	assertPhaseTwoAcceptanceDecision(t, contract.EngineeringDecision)
	assertPhaseTwoAcceptanceUAT(t, contract.UATCases)
	assertPhaseTwoAcceptanceDebts(t, contract.KnownDebts)
	assertPhaseTwoAcceptanceGates(t, contract.Gates)
}

func assertPhaseTwoAcceptanceHeader(t *testing.T, contract phaseTwoAcceptanceContract) {
	t.Helper()
	if contract.SchemaVersion != "P2-08-v1" || contract.Task != "P2-08" ||
		contract.Environment != "LOCAL_TEST" || contract.DataScope != "FIXED_TEST_ONLY" {
		t.Fatalf("unexpected acceptance header: %+v", contract)
	}
	capabilities := contract.Capabilities
	operations := contract.Operations
	if contract.RealTrialEnabled || contract.PromotionAllowed || capabilities.RealDataEnabled ||
		capabilities.FormalNotificationEnabled || capabilities.StaffAIShadowEnabled ||
		capabilities.UserFacingAIEnabled || capabilities.ExternalCallsEnabled ||
		operations.DeploymentExecuted || operations.TrafficSwitchExecuted ||
		operations.DatabaseRestoreExecuted || operations.VolumeResetExecuted ||
		operations.GitHistoryRewriteExecuted || operations.ExternalCallsExecuted {
		t.Fatalf("acceptance enabled a restricted capability or operation: %+v", contract)
	}
}

func assertPhaseTwoAcceptanceChecks(t *testing.T, checks []phaseTwoAcceptanceCheck) {
	t.Helper()
	want := []phaseTwoAcceptanceCheck{
		{ID: "P2-CHECK-01", Scope: "PHASE_TWO_AUTOMATION", Status: "PASSED"},
		{ID: "P2-CHECK-02", Scope: "REPOSITORY_VERIFY", Status: "PASSED"},
		{ID: "P2-CHECK-03", Scope: "FULL_BACKEND_SCAN", Status: "KNOWN_REPOSITORY_DEBT"},
		{ID: "P2-CHECK-04", Scope: "COMPOSE_HEALTH_AND_MIGRATION_IDEMPOTENCY", Status: "PASSED"},
		{ID: "P2-CHECK-05", Scope: "RUNTIME_CLOSED_CAPABILITIES", Status: "PASSED"},
		{ID: "P2-CHECK-06", Scope: "CORE_BROWSER_FLOWS", Status: "PASSED"},
	}
	if len(checks) != len(want) {
		t.Fatalf("acceptance check count = %d, want %d", len(checks), len(want))
	}
	for i, expected := range want {
		actual := checks[i]
		if actual.ID != expected.ID || actual.Scope != expected.Scope || actual.Evidence == "" {
			t.Fatalf("unexpected acceptance check: %+v", actual)
		}
		if actual.Status != expected.Status {
			t.Fatalf("acceptance check %s status = %s, want %s", actual.ID, actual.Status, expected.Status)
		}
	}
}

func assertPhaseTwoAcceptanceDecision(t *testing.T, decision string) {
	t.Helper()
	if decision != "PASSED_WITH_KNOWN_REPOSITORY_DEBT" {
		t.Fatalf("unexpected engineering decision %s", decision)
	}
}

func assertPhaseTwoAcceptanceUAT(t *testing.T, cases []phaseTwoAcceptanceUATCase) {
	t.Helper()
	wantIDs := []string{
		"P2-UAT-01", "P2-UAT-02", "P2-UAT-03", "P2-UAT-04", "P2-UAT-05",
		"P2-UAT-06", "P2-UAT-07", "P2-UAT-08", "P2-UAT-09", "P2-UAT-10",
	}
	gotIDs := make([]string, 0, len(cases))
	for _, item := range cases {
		if item.Scope == "" || len(item.Evidence) == 0 {
			t.Fatalf("incomplete UAT case: %+v", item)
		}
		if item.Status != "PASSED" {
			t.Fatalf("UAT case %s status = %s, want PASSED", item.ID, item.Status)
		}
		gotIDs = append(gotIDs, item.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("UAT case IDs = %v, want %v", gotIDs, wantIDs)
	}
}

func assertPhaseTwoAcceptanceDebts(t *testing.T, debts []phaseTwoAcceptanceKnownDebt) {
	t.Helper()
	wantAreas := []string{
		"MCP_CLIENT_LOCAL_SERVICE",
		"AI_PLUGIN_TEXT_ASSERTIONS",
		"AUTO_PLUGIN_COMPAT_ROUTE_ASSERTION",
		"AUTOCODE_FIXTURE_DATABASE_SETUP",
	}
	gotAreas := make([]string, 0, len(debts))
	for i, debt := range debts {
		wantID := fmt.Sprintf("REPO-TEST-%02d", i+1)
		if debt.ID != wantID || debt.Scope != "OUTSIDE_PHASE_TWO" || debt.Status != "OPEN" || debt.Area == "" {
			t.Fatalf("unexpected known debt: %+v", debt)
		}
		gotAreas = append(gotAreas, debt.Area)
	}
	if !reflect.DeepEqual(gotAreas, wantAreas) {
		t.Fatalf("known debt areas = %v, want %v", gotAreas, wantAreas)
	}
}

func assertPhaseTwoAcceptanceGates(t *testing.T, gates []phaseTwoAcceptanceGate) {
	t.Helper()
	want := map[string]string{
		"REAL_TRIAL_SCOPE_APPROVED":               "BLOCKED",
		"FORMAL_CONTENT_APPROVED":                 "BLOCKED",
		"REAL_IDENTITY_AND_CONSENT_APPROVED":      "BLOCKED",
		"FORMAL_NOTIFICATION_APPROVED":            "BLOCKED",
		"SERVICE_SLA_APPROVED":                    "BLOCKED",
		"CAPACITY_TARGETS_APPROVED":               "BLOCKED",
		"MONITORING_AND_INCIDENT_OWNERS_ASSIGNED": "BLOCKED",
		"BACKUP_RPO_RTO_APPROVED":                 "BLOCKED",
		"RESTORE_REHEARSAL_COMPLETED":             "BLOCKED",
		"SECURITY_REVIEW_COMPLETED":               "BLOCKED",
		"AI_SHADOW_SELECTED":                      "NOT_SELECTED",
	}
	if len(gates) != len(want) {
		t.Fatalf("acceptance gate count = %d, want %d", len(gates), len(want))
	}
	seen := make(map[string]struct{}, len(gates))
	for _, gate := range gates {
		status, ok := want[gate.Code]
		if !ok || gate.Status != status || gate.Owner != "UNASSIGNED" {
			t.Fatalf("unexpected acceptance gate: %+v", gate)
		}
		if _, duplicate := seen[gate.Code]; duplicate {
			t.Fatalf("duplicate acceptance gate %s", gate.Code)
		}
		seen[gate.Code] = struct{}{}
	}
}
