package core

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestBindEnvironmentOverrides(t *testing.T) {
	t.Setenv("GVA_REDIS_PASSWORD", "local-redis-secret")
	t.Setenv("GVA_SYSTEM_USE_REDIS", "true")

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(`
redis:
  password: from-file
system:
  use-redis: false
`)); err != nil {
		t.Fatalf("read config: %v", err)
	}

	bindEnvironmentOverrides(v)

	var got struct {
		Redis struct {
			Password string `mapstructure:"password"`
		} `mapstructure:"redis"`
		System struct {
			UseRedis bool `mapstructure:"use-redis"`
		} `mapstructure:"system"`
	}
	if err := v.Unmarshal(&got); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if got.Redis.Password != "local-redis-secret" {
		t.Fatalf("redis password = %q, want environment override", got.Redis.Password)
	}
	if !got.System.UseRedis {
		t.Fatal("system.use-redis was not overridden by environment")
	}
}

func TestBindEnvironmentOverridesWhenPersistedConfigLacksComposeKeys(t *testing.T) {
	t.Setenv("GVA_REDIS_ADDR", "redis:6379")
	t.Setenv("GVA_REDIS_PASSWORD", "runtime-secret")
	t.Setenv("GVA_SYSTEM_USE_REDIS", "true")
	t.Setenv("GVA_CARE_FIXTURE_NOW", "2026-08-18T10:00:00+08:00")

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader("system:\n  addr: 8888\n")); err != nil {
		t.Fatalf("read config: %v", err)
	}

	bindEnvironmentOverrides(v)

	var got struct {
		Redis struct {
			Addr     string `mapstructure:"addr"`
			Password string `mapstructure:"password"`
		} `mapstructure:"redis"`
		System struct {
			UseRedis bool `mapstructure:"use-redis"`
		} `mapstructure:"system"`
		Care struct {
			FixtureNow string `mapstructure:"fixture-now"`
		} `mapstructure:"care"`
	}
	if err := v.Unmarshal(&got); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if got.Redis.Addr != "redis:6379" || got.Redis.Password != "runtime-secret" {
		t.Fatalf("redis override = %#v", got.Redis)
	}
	if !got.System.UseRedis {
		t.Fatal("system.use-redis was not bound when absent from YAML")
	}
	if got.Care.FixtureNow != "2026-08-18T10:00:00+08:00" {
		t.Fatalf("care fixture time override = %q", got.Care.FixtureNow)
	}
}

func TestBindEnvironmentOverridesAllowsClearingPersistedFixtureNow(t *testing.T) {
	t.Setenv("GVA_CARE_FIXTURE_NOW", "")

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(`
care:
  fixture-now: "2026-08-18T10:00:00+08:00"
`)); err != nil {
		t.Fatalf("read config: %v", err)
	}

	bindEnvironmentOverrides(v)

	var got struct {
		Care struct {
			FixtureNow string `mapstructure:"fixture-now"`
		} `mapstructure:"care"`
	}
	if err := v.Unmarshal(&got); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if got.Care.FixtureNow != "" {
		t.Fatalf("care fixture time = %q, want explicit system-clock override", got.Care.FixtureNow)
	}
}
