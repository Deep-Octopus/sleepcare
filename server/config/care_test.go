package config

import (
	"testing"
	"time"
)

func TestCareNowUsesFixtureInstantOnlyWhenGateIsEnabled(t *testing.T) {
	const fixtureNow = "2026-08-18T10:00:00+08:00"
	want, err := time.Parse(time.RFC3339, fixtureNow)
	if err != nil {
		t.Fatal(err)
	}

	enabled := Care{SyntheticFixturesEnabled: true, FixtureNow: fixtureNow}
	if got := enabled.Now(); !got.Equal(want) {
		t.Fatalf("enabled fixture time = %s, want %s", got, want)
	}

	disabled := Care{SyntheticFixturesEnabled: false, FixtureNow: fixtureNow}
	before := time.Now().Add(-time.Second)
	got := disabled.Now()
	after := time.Now().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatalf("disabled fixture gate returned non-system time %s", got)
	}

	enabledWithoutFixtureTime := Care{SyntheticFixturesEnabled: true}
	before = time.Now().Add(-time.Second)
	got = enabledWithoutFixtureTime.Now()
	after = time.Now().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatalf("empty fixture time returned non-system time %s", got)
	}
}

func TestCareValidateRejectsMalformedFixtureInstant(t *testing.T) {
	if err := (Care{FixtureNow: "2026-08-18 10:00"}).Validate(); err == nil {
		t.Fatal("malformed fixture time was accepted")
	}
	if err := (Care{FixtureNow: "2026-08-18T10:00:00+08:00"}).Validate(); err != nil {
		t.Fatalf("valid fixture time was rejected: %v", err)
	}
}

func TestNotificationProviderConfigRejectsProductionModeAndUnsafeBoundaries(t *testing.T) {
	if err := (Care{NotificationProvider: NotificationProvider{Mode: "PRODUCTION"}}).Validate(); err == nil {
		t.Fatal("production provider mode was accepted")
	}
	if err := (Care{
		SyntheticFixturesEnabled: true,
		NotificationProvider: NotificationProvider{
			Mode: "CONTRACT_TEST", CallbackMaxSkewSeconds: 10,
		},
	}).Validate(); err == nil {
		t.Fatal("unsafe callback skew was accepted")
	}
	if err := (Care{
		SyntheticFixturesEnabled: true,
		NotificationProvider: NotificationProvider{
			Mode: "CONTRACT_TEST", CallbackMaxSkewSeconds: 300,
			CostCurrency: "CNY",
		},
	}).Validate(); err != nil {
		t.Fatalf("contract-test provider config was rejected: %v", err)
	}
}

func TestDataGovernanceConfigRejectsRealModeAndIncompleteReviewedPolicy(t *testing.T) {
	if err := (Care{DataGovernance: DataGovernance{Mode: "PRODUCTION"}}).Validate(); err == nil {
		t.Fatal("real-data governance mode was accepted")
	}
	if err := (Care{DataGovernance: DataGovernance{Mode: "CONTRACT_TEST"}}).Validate(); err == nil {
		t.Fatal("contract-test governance without the fixture gate was accepted")
	}
	if err := (Care{
		SyntheticFixturesEnabled: true,
		DataGovernance: DataGovernance{
			Mode:                  "CONTRACT_TEST",
			ServiceNoticeReviewed: true,
		},
	}).Validate(); err == nil {
		t.Fatal("reviewed service notice without a version was accepted")
	}
	if err := (Care{
		SyntheticFixturesEnabled: true,
		DataGovernance: DataGovernance{
			Mode:                  "CONTRACT_TEST",
			ServiceNoticeVersion:  "SERVICE-NOTICE-TEST-V1",
			ServiceNoticeReviewed: true,
			PrivacyNoticeVersion:  "PRIVACY-NOTICE-TEST-V1",
			PrivacyNoticeReviewed: true,
		},
	}).Validate(); err != nil {
		t.Fatalf("safe contract-test governance config was rejected: %v", err)
	}
}
