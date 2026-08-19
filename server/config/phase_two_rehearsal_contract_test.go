package config

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"testing"
)

type phaseTwoRehearsalContract struct {
	SchemaVersion    string                    `json:"schemaVersion"`
	Task             string                    `json:"task"`
	Environment      string                    `json:"environment"`
	DataScope        string                    `json:"dataScope"`
	ExecutionMode    string                    `json:"executionMode"`
	RealTrialEnabled bool                      `json:"realTrialEnabled"`
	PromotionAllowed bool                      `json:"promotionAllowed"`
	Capabilities     rehearsalCapabilities     `json:"capabilities"`
	Gates            []rehearsalGate           `json:"gates"`
	UATCases         []rehearsalUATCase        `json:"uatCases"`
	TrainingModules  []rehearsalTrainingModule `json:"trainingModules"`
	Tabletop         rehearsalTabletop         `json:"tabletop"`
}

type rehearsalCapabilities struct {
	RealDataEnabled           bool `json:"realDataEnabled"`
	FormalNotificationEnabled bool `json:"formalNotificationEnabled"`
	StaffAIShadowEnabled      bool `json:"staffAIShadowEnabled"`
	UserFacingAIEnabled       bool `json:"userFacingAIEnabled"`
	ExternalCallsEnabled      bool `json:"externalCallsEnabled"`
}

type rehearsalGate struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Owner  string `json:"owner"`
}

type rehearsalUATCase struct {
	ID              string `json:"id"`
	Scope           string `json:"scope"`
	ExecutionStatus string `json:"executionStatus"`
}

type rehearsalTrainingModule struct {
	Role           string   `json:"role"`
	Topics         []string `json:"topics"`
	DeliveryStatus string   `json:"deliveryStatus"`
}

type rehearsalTabletop struct {
	InitialState     string                    `json:"initialState"`
	RollbackStrategy string                    `json:"rollbackStrategy"`
	Controls         rehearsalTabletopControls `json:"controls"`
	Steps            []rehearsalTabletopStep   `json:"steps"`
}

type rehearsalTabletopControls struct {
	NetworkCallsAllowed      bool `json:"networkCallsAllowed"`
	DeploymentAllowed        bool `json:"deploymentAllowed"`
	DatabaseWritesAllowed    bool `json:"databaseWritesAllowed"`
	DatabaseRestoreAllowed   bool `json:"databaseRestoreAllowed"`
	VolumeResetAllowed       bool `json:"volumeResetAllowed"`
	GitHistoryRewriteAllowed bool `json:"gitHistoryRewriteAllowed"`
	RetainBusinessData       bool `json:"retainBusinessData"`
	CredentialsRecorded      bool `json:"credentialsRecorded"`
}

type rehearsalTabletopStep struct {
	Order    int    `json:"order"`
	Action   string `json:"action"`
	From     string `json:"from"`
	To       string `json:"to"`
	Expected string `json:"expected"`
}

func TestPhaseTwoRehearsalContract(t *testing.T) {
	content, err := os.ReadFile("../../docs/contracts/phase2-rehearsal.json")
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var contract phaseTwoRehearsalContract
	if err = decoder.Decode(&contract); err != nil {
		t.Fatalf("phase-two rehearsal contract is invalid: %v", err)
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("phase-two rehearsal contract has trailing content: %v", err)
	}

	assertRehearsalHeader(t, contract)
	assertRehearsalGates(t, contract.Gates)
	assertRehearsalUATCases(t, contract.UATCases)
	assertRehearsalTraining(t, contract.TrainingModules)
	assertRehearsalTabletop(t, contract.Tabletop)
}

func assertRehearsalHeader(t *testing.T, contract phaseTwoRehearsalContract) {
	t.Helper()
	if contract.SchemaVersion != "P2-07-v1" || contract.Task != "P2-07" ||
		contract.Environment != "LOCAL_TEST" || contract.DataScope != "FIXED_TEST_ONLY" ||
		contract.ExecutionMode != "TABLETOP" {
		t.Fatalf("unexpected rehearsal header: %+v", contract)
	}
	if contract.RealTrialEnabled || contract.PromotionAllowed ||
		contract.Capabilities.RealDataEnabled || contract.Capabilities.FormalNotificationEnabled ||
		contract.Capabilities.StaffAIShadowEnabled || contract.Capabilities.UserFacingAIEnabled ||
		contract.Capabilities.ExternalCallsEnabled {
		t.Fatalf("rehearsal contract enabled a restricted capability: %+v", contract.Capabilities)
	}
}

func assertRehearsalGates(t *testing.T, gates []rehearsalGate) {
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
		t.Fatalf("rehearsal gate count = %d, want %d", len(gates), len(want))
	}
	seen := make(map[string]struct{}, len(gates))
	for _, gate := range gates {
		status, ok := want[gate.Code]
		if !ok || gate.Status != status || gate.Owner != "UNASSIGNED" {
			t.Fatalf("unexpected rehearsal gate: %+v", gate)
		}
		if _, duplicate := seen[gate.Code]; duplicate {
			t.Fatalf("duplicate rehearsal gate %s", gate.Code)
		}
		seen[gate.Code] = struct{}{}
	}
}

func assertRehearsalUATCases(t *testing.T, cases []rehearsalUATCase) {
	t.Helper()
	want := []string{
		"P2-UAT-01", "P2-UAT-02", "P2-UAT-03", "P2-UAT-04", "P2-UAT-05",
		"P2-UAT-06", "P2-UAT-07", "P2-UAT-08", "P2-UAT-09", "P2-UAT-10",
	}
	got := make([]string, 0, len(cases))
	for _, item := range cases {
		if item.Scope == "" || item.ExecutionStatus != "DEFERRED_TO_P2_08" {
			t.Fatalf("unexpected UAT case: %+v", item)
		}
		got = append(got, item.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UAT case IDs = %v, want %v", got, want)
	}
}

func assertRehearsalTraining(t *testing.T, modules []rehearsalTrainingModule) {
	t.Helper()
	want := []string{
		"CARE_CLIENT", "CARE_STEWARD", "CLINICIAN", "SUPERVISOR", "CONTENT_ADMIN",
		"SYSTEM_ADMIN_NEGATIVE_BOUNDARY",
	}
	got := make([]string, 0, len(modules))
	for _, module := range modules {
		if len(module.Topics) == 0 || module.DeliveryStatus != "PREPARED_NOT_DELIVERED" {
			t.Fatalf("unexpected training module: %+v", module)
		}
		got = append(got, module.Role)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("training roles = %v, want %v", got, want)
	}
}

func assertRehearsalTabletop(t *testing.T, tabletop rehearsalTabletop) {
	t.Helper()
	if tabletop.InitialState != "HOLD" || tabletop.RollbackStrategy != "CODE_ONLY_RETAIN_DATA" {
		t.Fatalf("unexpected tabletop boundary: %+v", tabletop)
	}
	controls := tabletop.Controls
	if controls.NetworkCallsAllowed || controls.DeploymentAllowed || controls.DatabaseWritesAllowed ||
		controls.DatabaseRestoreAllowed || controls.VolumeResetAllowed || controls.GitHistoryRewriteAllowed ||
		controls.CredentialsRecorded || !controls.RetainBusinessData {
		t.Fatalf("unsafe tabletop controls: %+v", controls)
	}
	want := []rehearsalTabletopStep{
		{Order: 1, Action: "BEGIN_DRY_RUN", From: "HOLD", To: "DRY_RUN", Expected: "ALLOWED"},
		{Order: 2, Action: "PROMOTE", From: "DRY_RUN", To: "DRY_RUN", Expected: "DENIED"},
		{Order: 3, Action: "PAUSE", From: "DRY_RUN", To: "DRY_RUN_PAUSED", Expected: "ALLOWED"},
		{Order: 4, Action: "ROLLBACK", From: "DRY_RUN_PAUSED", To: "DRY_RUN_ROLLED_BACK", Expected: "ALLOWED"},
		{Order: 5, Action: "VERIFY_AND_HOLD", From: "DRY_RUN_ROLLED_BACK", To: "HOLD", Expected: "ALLOWED"},
	}
	if !reflect.DeepEqual(tabletop.Steps, want) {
		t.Fatalf("tabletop steps = %+v, want %+v", tabletop.Steps, want)
	}
}
