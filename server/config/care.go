package config

// Care controls sleep-care development capabilities. Synthetic fixtures are
// opt-in and must stay disabled outside an explicitly configured local setup.
type Care struct {
	SyntheticFixturesEnabled bool   `mapstructure:"synthetic-fixtures-enabled" json:"synthetic-fixtures-enabled" yaml:"synthetic-fixtures-enabled"`
	FixturePassword          string `mapstructure:"fixture-password" json:"-" yaml:"fixture-password"`
}
