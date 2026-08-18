package config

// Care controls sleep-care development capabilities. Synthetic fixtures are
// opt-in and must stay disabled outside an explicitly configured local setup.
type Care struct {
	SyntheticFixturesEnabled bool         `mapstructure:"synthetic-fixtures-enabled" json:"synthetic-fixtures-enabled" yaml:"synthetic-fixtures-enabled"`
	FixturePassword          string       `mapstructure:"fixture-password" json:"-" yaml:"fixture-password"`
	ClientAccess             ClientAccess `mapstructure:"client-access" json:"client-access" yaml:"client-access"`
}

type ClientAccess struct {
	SessionTTLMinutes int      `mapstructure:"session-ttl-minutes" json:"session-ttl-minutes" yaml:"session-ttl-minutes"`
	CookieName        string   `mapstructure:"cookie-name" json:"cookie-name" yaml:"cookie-name"`
	CookiePath        string   `mapstructure:"cookie-path" json:"cookie-path" yaml:"cookie-path"`
	CookieSecure      bool     `mapstructure:"cookie-secure" json:"cookie-secure" yaml:"cookie-secure"`
	AllowedOrigins    []string `mapstructure:"allowed-origins" json:"allowed-origins" yaml:"allowed-origins"`
}
