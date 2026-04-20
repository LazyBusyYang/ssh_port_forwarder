package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadDefaults 测试默认值加载（path 为空）
func TestLoadDefaults(t *testing.T) {
	// 清空可能影响测试的环境变量
	cleanEnvVars()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with empty path should not return error, got: %v", err)
	}

	// 验证 Server 默认值
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host default expected '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port default expected 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Env != "production" {
		t.Errorf("Server.Env default expected 'production', got '%s'", cfg.Server.Env)
	}

	// 验证 Database 默认值
	if cfg.Database.Type != "sqlite" {
		t.Errorf("Database.Type default expected 'sqlite', got '%s'", cfg.Database.Type)
	}
	if cfg.Database.SQLite.Path != "./data/spf.db" {
		t.Errorf("Database.SQLite.Path default expected './data/spf.db', got '%s'", cfg.Database.SQLite.Path)
	}

	// 验证 JWT 默认值
	if cfg.JWT.TokenExpire != 86400 {
		t.Errorf("JWT.TokenExpire default expected 86400, got %d", cfg.JWT.TokenExpire)
	}
	if cfg.JWT.RefreshExpire != 604800 {
		t.Errorf("JWT.RefreshExpire default expected 604800, got %d", cfg.JWT.RefreshExpire)
	}

	// 验证 PortRange 默认值
	if cfg.PortRange.Min != 30000 {
		t.Errorf("PortRange.Min default expected 30000, got %d", cfg.PortRange.Min)
	}
	if cfg.PortRange.Max != 33000 {
		t.Errorf("PortRange.Max default expected 33000, got %d", cfg.PortRange.Max)
	}
}

// TestLoadFromYAML 测试从 YAML 文件加载
func TestLoadFromYAML(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
server:
  host: "127.0.0.1"
  port: 9090
  env: "development"

database:
  type: "mysql"
  sqlite:
    path: "/custom/path/db.sqlite"
  mysql:
    dsn: "user:pass@tcp(localhost:3306)/dbname"

jwt:
  secret_current: "current-secret"
  secret_previous: "previous-secret"
  token_expire: 3600
  refresh_expire: 7200

encryption:
  key: "encryption-key"
  key_previous: "encryption-key-prev"

port_range:
  min: 40000
  max: 45000
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 清空环境变量，确保只测试文件加载
	cleanEnvVars()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load from YAML should not return error, got: %v", err)
	}

	// 验证 Server 配置
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host expected '127.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port expected 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Env != "development" {
		t.Errorf("Server.Env expected 'development', got '%s'", cfg.Server.Env)
	}

	// 验证 Database 配置
	if cfg.Database.Type != "mysql" {
		t.Errorf("Database.Type expected 'mysql', got '%s'", cfg.Database.Type)
	}
	if cfg.Database.SQLite.Path != "/custom/path/db.sqlite" {
		t.Errorf("Database.SQLite.Path expected '/custom/path/db.sqlite', got '%s'", cfg.Database.SQLite.Path)
	}
	if cfg.Database.MySQL.DSN != "user:pass@tcp(localhost:3306)/dbname" {
		t.Errorf("Database.MySQL.DSN expected 'user:pass@tcp(localhost:3306)/dbname', got '%s'", cfg.Database.MySQL.DSN)
	}

	// 验证 JWT 配置
	if cfg.JWT.SecretCurrent != "current-secret" {
		t.Errorf("JWT.SecretCurrent expected 'current-secret', got '%s'", cfg.JWT.SecretCurrent)
	}
	if cfg.JWT.SecretPrevious != "previous-secret" {
		t.Errorf("JWT.SecretPrevious expected 'previous-secret', got '%s'", cfg.JWT.SecretPrevious)
	}
	if cfg.JWT.TokenExpire != 3600 {
		t.Errorf("JWT.TokenExpire expected 3600, got %d", cfg.JWT.TokenExpire)
	}
	if cfg.JWT.RefreshExpire != 7200 {
		t.Errorf("JWT.RefreshExpire expected 7200, got %d", cfg.JWT.RefreshExpire)
	}

	// 验证 Encryption 配置
	if cfg.Encryption.Key != "encryption-key" {
		t.Errorf("Encryption.Key expected 'encryption-key', got '%s'", cfg.Encryption.Key)
	}
	if cfg.Encryption.KeyPrevious != "encryption-key-prev" {
		t.Errorf("Encryption.KeyPrevious expected 'encryption-key-prev', got '%s'", cfg.Encryption.KeyPrevious)
	}

	// 验证 PortRange 配置
	if cfg.PortRange.Min != 40000 {
		t.Errorf("PortRange.Min expected 40000, got %d", cfg.PortRange.Min)
	}
	if cfg.PortRange.Max != 45000 {
		t.Errorf("PortRange.Max expected 45000, got %d", cfg.PortRange.Max)
	}
}

// TestLoadWithEnvOverride 测试环境变量覆盖
func TestLoadWithEnvOverride(t *testing.T) {
	// 设置环境变量
	os.Setenv("SPF_ENCRYPTION_KEY", "env-encryption-key")
	os.Setenv("SPF_ENCRYPTION_KEY_PREVIOUS", "env-encryption-key-prev")
	os.Setenv("JWT_SECRET_CURRENT", "env-jwt-current")
	os.Setenv("JWT_SECRET_PREVIOUS", "env-jwt-previous")
	os.Setenv("SPF_PORT_RANGE_MIN", "50000")
	os.Setenv("SPF_PORT_RANGE_MAX", "55000")

	// 清理环境变量
	defer cleanEnvVars()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with env vars should not return error, got: %v", err)
	}

	// 验证环境变量覆盖
	if cfg.Encryption.Key != "env-encryption-key" {
		t.Errorf("Encryption.Key expected 'env-encryption-key', got '%s'", cfg.Encryption.Key)
	}
	if cfg.Encryption.KeyPrevious != "env-encryption-key-prev" {
		t.Errorf("Encryption.KeyPrevious expected 'env-encryption-key-prev', got '%s'", cfg.Encryption.KeyPrevious)
	}
	if cfg.JWT.SecretCurrent != "env-jwt-current" {
		t.Errorf("JWT.SecretCurrent expected 'env-jwt-current', got '%s'", cfg.JWT.SecretCurrent)
	}
	if cfg.JWT.SecretPrevious != "env-jwt-previous" {
		t.Errorf("JWT.SecretPrevious expected 'env-jwt-previous', got '%s'", cfg.JWT.SecretPrevious)
	}
	if cfg.PortRange.Min != 50000 {
		t.Errorf("PortRange.Min expected 50000, got %d", cfg.PortRange.Min)
	}
	if cfg.PortRange.Max != 55000 {
		t.Errorf("PortRange.Max expected 55000, got %d", cfg.PortRange.Max)
	}
}

// TestLoadFileAndEnvOverride 测试文件和环境变量组合（环境变量优先级更高）
func TestLoadFileAndEnvOverride(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
server:
  port: 9090

encryption:
  key: "file-encryption-key"

port_range:
  min: 40000
  max: 45000
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 设置环境变量覆盖部分配置
	os.Setenv("SPF_ENCRYPTION_KEY", "env-encryption-key")
	os.Setenv("SPF_PORT_RANGE_MIN", "50000")

	// 清理环境变量
	defer cleanEnvVars()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load with file and env vars should not return error, got: %v", err)
	}

	// 验证文件中的配置
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port expected 9090 (from file), got %d", cfg.Server.Port)
	}

	// 验证环境变量覆盖的配置
	if cfg.Encryption.Key != "env-encryption-key" {
		t.Errorf("Encryption.Key expected 'env-encryption-key' (from env), got '%s'", cfg.Encryption.Key)
	}
	if cfg.PortRange.Min != 50000 {
		t.Errorf("PortRange.Min expected 50000 (from env), got %d", cfg.PortRange.Min)
	}

	// 验证文件中未被覆盖的配置
	if cfg.PortRange.Max != 45000 {
		t.Errorf("PortRange.Max expected 45000 (from file), got %d", cfg.PortRange.Max)
	}
}

// TestLoadNonExistentFile 测试加载不存在的文件
func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/non/existent/config.yaml")
	if err == nil {
		t.Error("Load with non-existent file should return error")
	}
}

// cleanEnvVars 清理测试相关的环境变量
func cleanEnvVars() {
	envVars := []string{
		"SPF_ENCRYPTION_KEY",
		"SPF_ENCRYPTION_KEY_PREVIOUS",
		"JWT_SECRET_CURRENT",
		"JWT_SECRET_PREVIOUS",
		"SPF_PORT_RANGE_MIN",
		"SPF_PORT_RANGE_MAX",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}
