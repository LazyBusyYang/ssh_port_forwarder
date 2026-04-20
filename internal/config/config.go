package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 根配置
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	PortRange  PortRangeConfig  `mapstructure:"port_range"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

type DatabaseConfig struct {
	Type   string       `mapstructure:"type"` // sqlite / mysql
	SQLite SQLiteConfig `mapstructure:"sqlite"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
}

type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

type MySQLConfig struct {
	DSN string `mapstructure:"dsn"`
}

type JWTConfig struct {
	SecretCurrent  string `mapstructure:"secret_current"`
	SecretPrevious string `mapstructure:"secret_previous"`
	TokenExpire    int64  `mapstructure:"token_expire"`   // 秒，默认 86400
	RefreshExpire  int64  `mapstructure:"refresh_expire"` // 秒，默认 604800
}

type EncryptionConfig struct {
	Key         string `mapstructure:"key"`
	KeyPrevious string `mapstructure:"key_previous"`
}

type PortRangeConfig struct {
	Min int `mapstructure:"min"` // 默认 30000
	Max int `mapstructure:"max"` // 默认 33000
}

// Load 加载配置文件
// 如果 path 为空字符串，则仅使用默认值和环境变量（不加载文件）
func Load(path string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 绑定环境变量
	bindEnvVariables(v)

	// 如果指定了配置文件路径，则加载文件
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// Server 默认值
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.env", "production")

	// Database 默认值
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.sqlite.path", "./data/spf.db")

	// JWT 默认值
	v.SetDefault("jwt.token_expire", 86400)
	v.SetDefault("jwt.refresh_expire", 604800)

	// PortRange 默认值
	v.SetDefault("port_range.min", 30000)
	v.SetDefault("port_range.max", 33000)
}

// bindEnvVariables 绑定环境变量
func bindEnvVariables(v *viper.Viper) {
	// 设置环境变量前缀（可选，用于区分应用特定的环境变量）
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 显式绑定特定的环境变量
	_ = v.BindEnv("database.mysql.dsn", "SPF_DB_DSN")
	_ = v.BindEnv("encryption.key", "SPF_ENCRYPTION_KEY")
	_ = v.BindEnv("encryption.key_previous", "SPF_ENCRYPTION_KEY_PREVIOUS")
	_ = v.BindEnv("jwt.secret_current", "JWT_SECRET_CURRENT")
	_ = v.BindEnv("jwt.secret_previous", "JWT_SECRET_PREVIOUS")
	_ = v.BindEnv("port_range.min", "SPF_PORT_RANGE_MIN")
	_ = v.BindEnv("port_range.max", "SPF_PORT_RANGE_MAX")
}
