package config

import (
	"fmt"
	"strings"
	"time"
)

// Care controls sleep-care development capabilities. Synthetic fixtures are
// opt-in and must stay disabled outside an explicitly configured local setup.
type Care struct {
	SyntheticFixturesEnabled bool                 `mapstructure:"synthetic-fixtures-enabled" json:"synthetic-fixtures-enabled" yaml:"synthetic-fixtures-enabled"`
	FixturePassword          string               `mapstructure:"fixture-password" json:"-" yaml:"fixture-password"`
	FixtureNow               string               `mapstructure:"fixture-now" json:"fixture-now" yaml:"fixture-now"`
	ClientAccess             ClientAccess         `mapstructure:"client-access" json:"client-access" yaml:"client-access"`
	NotificationProvider     NotificationProvider `mapstructure:"notification-provider" json:"notification-provider" yaml:"notification-provider"`
	DataGovernance           DataGovernance       `mapstructure:"data-governance" json:"data-governance" yaml:"data-governance"`
}

// Validate rejects a malformed local fixture clock during startup. The clock
// remains inert unless the explicit fixture gate is enabled.
func (c Care) Validate() error {
	value := strings.TrimSpace(c.FixtureNow)
	if value != "" {
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("care.fixture-now must use RFC3339: %w", err)
		}
	}
	if err := c.NotificationProvider.Validate(c.SyntheticFixturesEnabled); err != nil {
		return err
	}
	return c.DataGovernance.Validate(c.SyntheticFixturesEnabled)
}

// Now returns the configured fixture instant only for the gated local data
// set. Production-safe defaults continue to use the system clock.
func (c Care) Now() time.Time {
	if c.SyntheticFixturesEnabled {
		if value := strings.TrimSpace(c.FixtureNow); value != "" {
			if parsed, err := time.Parse(time.RFC3339, value); err == nil {
				return parsed
			}
		}
	}
	return time.Now()
}

type ClientAccess struct {
	SessionTTLMinutes int      `mapstructure:"session-ttl-minutes" json:"session-ttl-minutes" yaml:"session-ttl-minutes"`
	CookieName        string   `mapstructure:"cookie-name" json:"cookie-name" yaml:"cookie-name"`
	CookiePath        string   `mapstructure:"cookie-path" json:"cookie-path" yaml:"cookie-path"`
	CookieSecure      bool     `mapstructure:"cookie-secure" json:"cookie-secure" yaml:"cookie-secure"`
	AllowedOrigins    []string `mapstructure:"allowed-origins" json:"allowed-origins" yaml:"allowed-origins"`
}

type NotificationProvider struct {
	Mode                          string `mapstructure:"mode" json:"mode" yaml:"mode"`
	ProviderCode                  string `mapstructure:"provider-code" json:"provider-code" yaml:"provider-code"`
	PolicyCode                    string `mapstructure:"policy-code" json:"policy-code" yaml:"policy-code"`
	PolicyVersion                 int    `mapstructure:"policy-version" json:"policy-version" yaml:"policy-version"`
	TemplateCode                  string `mapstructure:"template-code" json:"template-code" yaml:"template-code"`
	RequestSigningSecret          string `mapstructure:"request-signing-secret" json:"-" yaml:"request-signing-secret"`
	CallbackVerificationSecret    string `mapstructure:"callback-verification-secret" json:"-" yaml:"callback-verification-secret"`
	CallbackMaxSkewSeconds        int    `mapstructure:"callback-max-skew-seconds" json:"callback-max-skew-seconds" yaml:"callback-max-skew-seconds"`
	QualificationEvidenceReviewed bool   `mapstructure:"qualification-evidence-reviewed" json:"qualification-evidence-reviewed" yaml:"qualification-evidence-reviewed"`
	TemplateEvidenceReviewed      bool   `mapstructure:"template-evidence-reviewed" json:"template-evidence-reviewed" yaml:"template-evidence-reviewed"`
	ReceiptSemanticsReviewed      bool   `mapstructure:"receipt-semantics-reviewed" json:"receipt-semantics-reviewed" yaml:"receipt-semantics-reviewed"`
	RetryPolicyReviewed           bool   `mapstructure:"retry-policy-reviewed" json:"retry-policy-reviewed" yaml:"retry-policy-reviewed"`
	FallbackReviewed              bool   `mapstructure:"fallback-reviewed" json:"fallback-reviewed" yaml:"fallback-reviewed"`
	CostBoundaryReviewed          bool   `mapstructure:"cost-boundary-reviewed" json:"cost-boundary-reviewed" yaml:"cost-boundary-reviewed"`
	MaxAttemptsPerRequest         int    `mapstructure:"max-attempts-per-request" json:"max-attempts-per-request" yaml:"max-attempts-per-request"`
	RateLimitWindowSeconds        int    `mapstructure:"rate-limit-window-seconds" json:"rate-limit-window-seconds" yaml:"rate-limit-window-seconds"`
	RateLimitCount                int64  `mapstructure:"rate-limit-count" json:"rate-limit-count" yaml:"rate-limit-count"`
	CostCurrency                  string `mapstructure:"cost-currency" json:"cost-currency" yaml:"cost-currency"`
	EstimatedCostMinor            int64  `mapstructure:"estimated-cost-minor" json:"estimated-cost-minor" yaml:"estimated-cost-minor"`
	DailyCostLimitMinor           int64  `mapstructure:"daily-cost-limit-minor" json:"daily-cost-limit-minor" yaml:"daily-cost-limit-minor"`
}

func (c NotificationProvider) Validate(fixtureDataEnabled bool) error {
	mode := strings.ToUpper(strings.TrimSpace(c.Mode))
	if mode == "" {
		mode = "DISABLED"
	}
	if mode != "DISABLED" && mode != "CONTRACT_TEST" {
		return fmt.Errorf("care.notification-provider.mode must be DISABLED or CONTRACT_TEST")
	}
	if mode == "CONTRACT_TEST" && !fixtureDataEnabled {
		return fmt.Errorf("care.notification-provider CONTRACT_TEST requires the local fixture gate")
	}
	if c.PolicyVersion < 0 || c.CallbackMaxSkewSeconds < 0 || c.MaxAttemptsPerRequest < 0 ||
		c.RateLimitWindowSeconds < 0 || c.RateLimitCount < 0 || c.EstimatedCostMinor < 0 || c.DailyCostLimitMinor < 0 {
		return fmt.Errorf("care.notification-provider numeric boundaries cannot be negative")
	}
	if c.CallbackMaxSkewSeconds > 0 && (c.CallbackMaxSkewSeconds < 30 || c.CallbackMaxSkewSeconds > 900) {
		return fmt.Errorf("care.notification-provider callback skew must be between 30 and 900 seconds")
	}
	currency := strings.TrimSpace(c.CostCurrency)
	if currency != "" && (len(currency) != 3 || currency != strings.ToUpper(currency)) {
		return fmt.Errorf("care.notification-provider cost currency must be a three-letter uppercase code")
	}
	return nil
}

type DataGovernance struct {
	Mode                           string `mapstructure:"mode" json:"mode" yaml:"mode"`
	ServiceNoticeVersion           string `mapstructure:"service-notice-version" json:"service-notice-version" yaml:"service-notice-version"`
	PrivacyNoticeVersion           string `mapstructure:"privacy-notice-version" json:"privacy-notice-version" yaml:"privacy-notice-version"`
	NotificationConsentVersion     string `mapstructure:"notification-consent-version" json:"notification-consent-version" yaml:"notification-consent-version"`
	AIProcessingConsentVersion     string `mapstructure:"ai-processing-consent-version" json:"ai-processing-consent-version" yaml:"ai-processing-consent-version"`
	IdentityVerificationReviewed   bool   `mapstructure:"identity-verification-reviewed" json:"identity-verification-reviewed" yaml:"identity-verification-reviewed"`
	ServiceNoticeReviewed          bool   `mapstructure:"service-notice-reviewed" json:"service-notice-reviewed" yaml:"service-notice-reviewed"`
	PrivacyNoticeReviewed          bool   `mapstructure:"privacy-notice-reviewed" json:"privacy-notice-reviewed" yaml:"privacy-notice-reviewed"`
	NotificationConsentReviewed    bool   `mapstructure:"notification-consent-reviewed" json:"notification-consent-reviewed" yaml:"notification-consent-reviewed"`
	AIProcessingConsentReviewed    bool   `mapstructure:"ai-processing-consent-reviewed" json:"ai-processing-consent-reviewed" yaml:"ai-processing-consent-reviewed"`
	ConsentEvidenceReviewed        bool   `mapstructure:"consent-evidence-reviewed" json:"consent-evidence-reviewed" yaml:"consent-evidence-reviewed"`
	WithdrawalPolicyReviewed       bool   `mapstructure:"withdrawal-policy-reviewed" json:"withdrawal-policy-reviewed" yaml:"withdrawal-policy-reviewed"`
	MinimumNecessaryFieldsReviewed bool   `mapstructure:"minimum-necessary-fields-reviewed" json:"minimum-necessary-fields-reviewed" yaml:"minimum-necessary-fields-reviewed"`
	RetentionPolicyReviewed        bool   `mapstructure:"retention-policy-reviewed" json:"retention-policy-reviewed" yaml:"retention-policy-reviewed"`
	CorrectionPolicyReviewed       bool   `mapstructure:"correction-policy-reviewed" json:"correction-policy-reviewed" yaml:"correction-policy-reviewed"`
	ErasurePolicyReviewed          bool   `mapstructure:"erasure-policy-reviewed" json:"erasure-policy-reviewed" yaml:"erasure-policy-reviewed"`
	ExportPolicyReviewed           bool   `mapstructure:"export-policy-reviewed" json:"export-policy-reviewed" yaml:"export-policy-reviewed"`
	SensitiveAccessAuditReviewed   bool   `mapstructure:"sensitive-access-audit-reviewed" json:"sensitive-access-audit-reviewed" yaml:"sensitive-access-audit-reviewed"`
	BackupRestoreReviewed          bool   `mapstructure:"backup-restore-reviewed" json:"backup-restore-reviewed" yaml:"backup-restore-reviewed"`
}

func (c DataGovernance) NormalizedMode() string {
	mode := strings.ToUpper(strings.TrimSpace(c.Mode))
	if mode == "" {
		return "DISABLED"
	}
	return mode
}

func (c DataGovernance) ContractTestEnabled(fixtureDataEnabled bool) bool {
	return fixtureDataEnabled && c.NormalizedMode() == "CONTRACT_TEST"
}

func (c DataGovernance) Validate(fixtureDataEnabled bool) error {
	mode := c.NormalizedMode()
	if mode != "DISABLED" && mode != "CONTRACT_TEST" {
		return fmt.Errorf("care.data-governance.mode must be DISABLED or CONTRACT_TEST")
	}
	if mode == "CONTRACT_TEST" && !fixtureDataEnabled {
		return fmt.Errorf("care.data-governance CONTRACT_TEST requires the local fixture gate")
	}
	versions := []struct {
		name     string
		value    string
		reviewed bool
	}{
		{"service-notice-version", c.ServiceNoticeVersion, c.ServiceNoticeReviewed},
		{"privacy-notice-version", c.PrivacyNoticeVersion, c.PrivacyNoticeReviewed},
		{"notification-consent-version", c.NotificationConsentVersion, c.NotificationConsentReviewed},
		{"ai-processing-consent-version", c.AIProcessingConsentVersion, c.AIProcessingConsentReviewed},
	}
	for _, version := range versions {
		value := strings.TrimSpace(version.value)
		if len([]rune(value)) > 80 {
			return fmt.Errorf("care.data-governance %s cannot exceed 80 characters", version.name)
		}
		if version.reviewed && value == "" {
			return fmt.Errorf("care.data-governance %s is required when its review gate is enabled", version.name)
		}
	}
	return nil
}
