package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testSecret = "0123456789abcdef0123456789abcdef"

var envKeys = []string{
	"CONFIG_FILE",
	"APP_PORT",
	"APP_READ_HEADER_TIMEOUT",
	"APP_READ_TIMEOUT",
	"APP_WRITE_TIMEOUT",
	"APP_IDLE_TIMEOUT",
	"APP_SHUTDOWN_TIMEOUT",
	"DB_HOST",
	"DB_PORT",
	"DB_USER",
	"DB_PASSWORD",
	"DB_NAME",
	"DB_MAX_OPEN_CONNS",
	"DB_MAX_IDLE_CONNS",
	"DB_CONN_MAX_LIFETIME",
	"DB_CONN_MAX_IDLE_TIME",
	"DB_CONNECT_TIMEOUT",
	"REDIS_ADDR",
	"REDIS_PASSWORD",
	"REDIS_DB",
	"REDIS_POOL_SIZE",
	"REDIS_DIAL_TIMEOUT",
	"REDIS_READ_TIMEOUT",
	"REDIS_WRITE_TIMEOUT",
	"JWT_SECRET",
	"JWT_TTL",
	"CACHE_TTL",
	"LOG_LEVEL",
	"LOG_FORMAT",
}

func isolateEnv(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, key := range envKeys {
		t.Setenv(key, "")
	}
	t.Setenv("JWT_SECRET", testSecret)
}

func TestLoadDefaults(t *testing.T) {
	isolateEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want 8080", cfg.App.Port)
	}
	if cfg.App.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("App.ReadHeaderTimeout = %v, want 5s", cfg.App.ReadHeaderTimeout)
	}
	if cfg.DB.Host != "mysql" || cfg.DB.Port != 3306 {
		t.Errorf("DB = %+v, want mysql:3306", cfg.DB)
	}
	if cfg.DB.MaxOpenConns != 25 || cfg.DB.MaxIdleConns != 25 {
		t.Errorf("DB pool = %d/%d, want 25/25", cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Errorf("Redis.Addr = %q, want redis:6379", cfg.Redis.Addr)
	}
	if cfg.Redis.PoolSize != 10 {
		t.Errorf("Redis.PoolSize = %d, want 10", cfg.Redis.PoolSize)
	}
	if cfg.Redis.DialTimeout != 2*time.Second {
		t.Errorf("Redis.DialTimeout = %v, want 2s", cfg.Redis.DialTimeout)
	}
	if cfg.Redis.ReadTimeout != time.Second || cfg.Redis.WriteTimeout != time.Second {
		t.Errorf(
			"Redis timeouts = %v/%v, want 1s/1s",
			cfg.Redis.ReadTimeout,
			cfg.Redis.WriteTimeout,
		)
	}
	if cfg.JWT.Secret != testSecret {
		t.Errorf("JWT.Secret = %q, want test secret", cfg.JWT.Secret)
	}
	if cfg.JWT.TTL != 24*time.Hour {
		t.Errorf("JWT.TTL = %v, want 24h", cfg.JWT.TTL)
	}
	if cfg.Cache.TTL != 5*time.Minute {
		t.Errorf("Cache.TTL = %v, want 5m", cfg.Cache.TTL)
	}
	if cfg.Log.Level != "info" || cfg.Log.Format != "json" {
		t.Errorf("Log = %+v, want info/json", cfg.Log)
	}
}

func TestEnvOverridesDefaults(t *testing.T) {
	isolateEnv(t)

	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_MAX_OPEN_CONNS", "10")
	t.Setenv("DB_MAX_IDLE_CONNS", "5")
	t.Setenv("REDIS_POOL_SIZE", "5")
	t.Setenv("REDIS_DIAL_TIMEOUT", "3s")
	t.Setenv("REDIS_READ_TIMEOUT", "2s")
	t.Setenv("REDIS_WRITE_TIMEOUT", "2s")
	t.Setenv("JWT_TTL", "2h")
	t.Setenv("CACHE_TTL", "1m")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Port != 9090 {
		t.Errorf("App.Port = %d, want 9090", cfg.App.Port)
	}
	if cfg.DB.Port != 3307 {
		t.Errorf("DB.Port = %d, want 3307", cfg.DB.Port)
	}
	if cfg.DB.MaxOpenConns != 10 || cfg.DB.MaxIdleConns != 5 {
		t.Errorf("DB pool = %d/%d, want 10/5", cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	}
	if cfg.Redis.PoolSize != 5 {
		t.Errorf("Redis.PoolSize = %d, want 5", cfg.Redis.PoolSize)
	}
	if cfg.Redis.DialTimeout != 3*time.Second {
		t.Errorf("Redis.DialTimeout = %v, want 3s", cfg.Redis.DialTimeout)
	}
	if cfg.Redis.ReadTimeout != 2*time.Second || cfg.Redis.WriteTimeout != 2*time.Second {
		t.Errorf(
			"Redis timeouts = %v/%v, want 2s/2s",
			cfg.Redis.ReadTimeout,
			cfg.Redis.WriteTimeout,
		)
	}
	if cfg.JWT.TTL != 2*time.Hour {
		t.Errorf("JWT.TTL = %v, want 2h", cfg.JWT.TTL)
	}
	if cfg.Cache.TTL != time.Minute {
		t.Errorf("Cache.TTL = %v, want 1m", cfg.Cache.TTL)
	}
	if cfg.Log.Level != "debug" || cfg.Log.Format != "text" {
		t.Errorf("Log = %+v, want debug/text", cfg.Log)
	}
}

func TestConfigFileAndEnvPrecedence(t *testing.T) {
	isolateEnv(t)

	path := writeConfig(t, `
app:
  port: 7070
db:
  host: filedb
jwt:
  secret: fedcba9876543210fedcba9876543210
  ttl: 2h
log:
  level: warn
`)
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("APP_PORT", "9090")
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Port != 9090 {
		t.Errorf("App.Port = %d, want env override 9090", cfg.App.Port)
	}
	if cfg.DB.Host != "filedb" {
		t.Errorf("DB.Host = %q, want filedb", cfg.DB.Host)
	}
	if cfg.JWT.Secret != "fedcba9876543210fedcba9876543210" {
		t.Errorf("JWT.Secret = %q, want file value", cfg.JWT.Secret)
	}
	if cfg.JWT.TTL != 2*time.Hour {
		t.Errorf("JWT.TTL = %v, want 2h", cfg.JWT.TTL)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("Log.Level = %q, want warn", cfg.Log.Level)
	}
	if cfg.Cache.TTL != 5*time.Minute {
		t.Errorf("Cache.TTL = %v, want default 5m", cfg.Cache.TTL)
	}
}

func TestConfigFileDiscovery(t *testing.T) {
	isolateEnv(t)

	writeFile(t, filepath.Join(mustGetwd(t), "config.yaml"), `
app:
  port: 6060
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Port != 6060 {
		t.Errorf("App.Port = %d, want 6060", cfg.App.Port)
	}
}

func TestMissingConfigFile(t *testing.T) {
	isolateEnv(t)

	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing config error")
	}
}

func TestInvalidYAML(t *testing.T) {
	isolateEnv(t)

	path := writeConfig(t, "app: [")
	t.Setenv("CONFIG_FILE", path)

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want decode error")
	}
}

func TestValidation(t *testing.T) {
	isolateEnv(t)

	base, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"app port", func(c *Config) { c.App.Port = 0 }, "app.port"},
		{"db host", func(c *Config) { c.DB.Host = "" }, "db.host"},
		{"db password", func(c *Config) { c.DB.Password = "" }, "db.password"},
		{"db pool", func(c *Config) { c.DB.MaxIdleConns = c.DB.MaxOpenConns + 1 }, "db.max_idle_conns"},
		{"redis pool", func(c *Config) { c.Redis.PoolSize = 0 }, "redis.pool_size"},
		{"redis dial timeout", func(c *Config) { c.Redis.DialTimeout = 0 }, "redis.dial_timeout"},
		{"redis read timeout", func(c *Config) { c.Redis.ReadTimeout = 0 }, "redis.read_timeout"},
		{"redis write timeout", func(c *Config) { c.Redis.WriteTimeout = 0 }, "redis.write_timeout"},
		{"jwt secret", func(c *Config) { c.JWT.Secret = "short" }, "jwt.secret"},
		{"cache ttl", func(c *Config) { c.Cache.TTL = 0 }, "cache.ttl"},
		{"log level", func(c *Config) { c.Log.Level = "verbose" }, "log.level"},
		{"log format", func(c *Config) { c.Log.Format = "xml" }, "log.format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %q, want mention of %q", err, tt.want)
			}

			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Errorf("Validate() error type = %T, want *ValidationError", err)
			}
		})
	}
}

func TestLoadRejectsShortSecretAndBadDuration(t *testing.T) {
	isolateEnv(t)
	t.Setenv("JWT_SECRET", "short")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "jwt.secret") {
		t.Fatalf("Load() error = %v, want jwt.secret validation error", err)
	}

	isolateEnv(t)
	t.Setenv("JWT_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want duration decode error")
	}
}

func TestExampleConfigFile(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "config.example.yaml"))
	if err != nil {
		t.Fatalf("resolve example config: %v", err)
	}

	isolateEnv(t)
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want 8080", cfg.App.Port)
	}
	if cfg.DB.Host != "mysql" {
		t.Errorf("DB.Host = %q, want mysql", cfg.DB.Host)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", cfg.Log.Format)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, content)
	return path
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return dir
}
