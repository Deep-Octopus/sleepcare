package config

import (
	"fmt"
	"strings"
	"time"
)

// Care controls sleep-care development capabilities. Synthetic fixtures are
// opt-in and must stay disabled outside an explicitly configured local setup.
type Care struct {
	SyntheticFixturesEnabled bool         `mapstructure:"synthetic-fixtures-enabled" json:"synthetic-fixtures-enabled" yaml:"synthetic-fixtures-enabled"`
	FixturePassword          string       `mapstructure:"fixture-password" json:"-" yaml:"fixture-password"`
	FixtureNow               string       `mapstructure:"fixture-now" json:"fixture-now" yaml:"fixture-now"`
	ClientAccess             ClientAccess `mapstructure:"client-access" json:"client-access" yaml:"client-access"`
}

// Validate rejects a malformed local fixture clock during startup. The clock
// remains inert unless the explicit fixture gate is enabled.
func (c Care) Validate() error {
	value := strings.TrimSpace(c.FixtureNow)
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("care.fixture-now must use RFC3339: %w", err)
	}
	return nil
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
