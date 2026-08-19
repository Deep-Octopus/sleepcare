package core

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/core/internal"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Viper 配置
func Viper() *viper.Viper {
	config := getConfigPath()

	v := viper.New()
	v.SetConfigFile(config)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	bindEnvironmentOverrides(v)
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		if err = v.Unmarshal(&global.GVA_CONFIG); err != nil {
			fmt.Println(err)
		}
	})
	if err = v.Unmarshal(&global.GVA_CONFIG); err != nil {
		panic(fmt.Errorf("fatal error unmarshal config: %w", err))
	}
	if err = global.GVA_CONFIG.Care.Validate(); err != nil {
		panic(fmt.Errorf("fatal error validate care config: %w", err))
	}

	// root 适配性 根据root位置去找到对应迁移位置,保证root路径有效
	global.GVA_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	return v
}

// bindEnvironmentOverrides allows container deployments to keep credentials
// outside the tracked YAML file. For example, mysql.password can be provided
// through GVA_MYSQL_PASSWORD. BindEnv is required so overrides also participate
// in Unmarshal; AutomaticEnv alone is not sufficient for nested structs.
func bindEnvironmentOverrides(v *viper.Viper) {
	v.SetEnvPrefix("GVA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	// These keys are supplied by Compose even when an older persisted config
	// file predates them. Bind them explicitly so nested Unmarshal sees them.
	for _, key := range []string{
		"care.synthetic-fixtures-enabled",
		"care.fixture-password",
		"care.fixture-now",
		"care.client-access.session-ttl-minutes",
		"care.client-access.cookie-name",
		"care.client-access.cookie-path",
		"care.client-access.cookie-secure",
		"care.client-access.allowed-origins",
		"redis.addr",
		"redis.password",
		"system.use-redis",
	} {
		_ = v.BindEnv(key)
	}
	for _, key := range v.AllKeys() {
		_ = v.BindEnv(key)
	}
	// Viper normally treats an empty environment value as unset. Compose uses
	// an explicit empty value to clear a persisted acceptance clock and restore
	// the system clock, so this one key must preserve empty-value semantics.
	if value, ok := os.LookupEnv("GVA_CARE_FIXTURE_NOW"); ok {
		v.Set("care.fixture-now", value)
	}
}

// getConfigPath 获取配置文件路径, 优先级: 命令行 > 环境变量 > 默认值
func getConfigPath() (config string) {
	// `-c` flag parse
	flag.StringVar(&config, "c", "", "choose config file.")
	flag.Parse()
	if config != "" { // 命令行参数不为空 将值赋值于config
		fmt.Printf("您正在使用命令行的 '-c' 参数传递的值, config 的路径为 %s\n", config)
		return
	}
	if env := os.Getenv(internal.ConfigEnv); env != "" { // 判断环境变量 GVA_CONFIG
		config = env
		fmt.Printf("您正在使用 %s 环境变量, config 的路径为 %s\n", internal.ConfigEnv, config)
		return
	}

	switch gin.Mode() { // 根据 gin 模式文件名
	case gin.DebugMode:
		config = internal.ConfigDebugFile
	case gin.ReleaseMode:
		config = internal.ConfigReleaseFile
	case gin.TestMode:
		config = internal.ConfigTestFile
	}
	fmt.Printf("您正在使用 gin 的 %s 模式运行, config 的路径为 %s\n", gin.Mode(), config)

	_, err := os.Stat(config)
	if err != nil || os.IsNotExist(err) {
		config = internal.ConfigDefaultFile
		fmt.Printf("配置文件路径不存在, 使用默认配置文件路径: %s\n", config)
	}

	return
}
