package config

import (
	"bytes"
	"os"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestShippedServerConfigsMatchSchema(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "default", path: "../config.yaml"},
		{name: "compose", path: "../config.compose.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			decoder := yaml.NewDecoder(bytes.NewReader(content))
			decoder.KnownFields(true)
			var cfg Server
			if err := decoder.Decode(&cfg); err != nil {
				t.Fatalf("配置模板与 config.Server 不一致: %v", err)
			}
		})
	}
}

func TestComposeConfigKeepsSafeInitializationSemantics(t *testing.T) {
	content, err := os.ReadFile("../config.compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg Server
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Mysql.Dbname != "" || cfg.Mysql.Path != "" || cfg.Mysql.Password != "" {
		t.Fatal("compose template must keep MySQL empty so initdb can seed roles, menus, Casbin and admin")
	}
	if !cfg.System.UseRedis || cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("compose Redis config = use:%v addr:%q", cfg.System.UseRedis, cfg.Redis.Addr)
	}
	if cfg.System.DisableAutoMigrate {
		t.Fatal("local compose requires auto-migrate until an explicit migration workflow exists")
	}
	if cfg.Care.SyntheticFixturesEnabled || cfg.Care.FixturePassword != "" || cfg.Care.FixtureNow != "" {
		t.Fatal("compose template must keep synthetic fixtures disabled until Compose injects the local-only gate and password")
	}
	if !slices.Contains(cfg.Zap.FileOnlyModules, "http") || !slices.Contains(cfg.Zap.FileOnlyModules, "sql") {
		t.Fatalf("http/sql logs must stay file-only, got %v", cfg.Zap.FileOnlyModules)
	}
}

func TestComposeDefaultsCareClockToSystemTime(t *testing.T) {
	content, err := os.ReadFile("../../compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var compose struct {
		Services struct {
			Server struct {
				Environment map[string]string `yaml:"environment"`
			} `yaml:"server"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatal(err)
	}

	const systemClockDefault = "${GVA_CARE_FIXTURE_NOW:-}"
	if got := compose.Services.Server.Environment["GVA_CARE_FIXTURE_NOW"]; got != systemClockDefault {
		t.Fatalf("compose care clock override = %q, want opt-in value %q", got, systemClockDefault)
	}
}
