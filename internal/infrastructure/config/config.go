// Package config loads application configuration from defaults, an optional
// config file and environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root application configuration.
type Config struct {
	App   App   `mapstructure:"app"`
	DB    DB    `mapstructure:"db"`
	Redis Redis `mapstructure:"redis"`
	JWT   JWT   `mapstructure:"jwt"`
	Cache Cache `mapstructure:"cache"`
	Log   Log   `mapstructure:"log"`
}

// App holds HTTP server settings.
type App struct {
	Port              int           `mapstructure:"port"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

// DB holds MySQL connection and pool settings.
type DB struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
}

// Redis holds Redis connection settings.
type Redis struct {
	Addr         string        `mapstructure:"addr"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// JWT holds token settings.
type JWT struct {
	Secret string        `mapstructure:"secret"`
	TTL    time.Duration `mapstructure:"ttl"`
}

// Cache holds cache settings.
type Cache struct {
	TTL time.Duration `mapstructure:"ttl"`
}

// Log holds logging settings.
type Log struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// ValidationError aggregates configuration problems.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "invalid configuration: " + strings.Join(e.Problems, "; ")
}

// Load reads configuration with the following precedence:
// environment variables > config file > defaults.
//
// By default Load looks for config.* in the current directory and ./config.
// CONFIG_FILE can point to a specific YAML, JSON or TOML file.
func Load() (Config, error) {
	cfg, err := load(os.Getenv("CONFIG_FILE"))
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func load(configFile string) (Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)

	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	if err := v.ReadInConfig(); err != nil {
		if configFile != "" {
			return Config{}, fmt.Errorf("read config file %q: %w", configFile, err)
		}

		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

// Validate checks required fields and sane ranges.
func (c Config) Validate() error {
	var problems []string

	validatePort(&problems, "app.port", c.App.Port)
	validatePositiveDuration(&problems, "app.read_header_timeout", c.App.ReadHeaderTimeout)
	validatePositiveDuration(&problems, "app.read_timeout", c.App.ReadTimeout)
	validatePositiveDuration(&problems, "app.write_timeout", c.App.WriteTimeout)
	validatePositiveDuration(&problems, "app.idle_timeout", c.App.IdleTimeout)
	validatePositiveDuration(&problems, "app.shutdown_timeout", c.App.ShutdownTimeout)

	if strings.TrimSpace(c.DB.Host) == "" {
		problems = append(problems, "db.host is required")
	}
	validatePort(&problems, "db.port", c.DB.Port)
	if strings.TrimSpace(c.DB.User) == "" {
		problems = append(problems, "db.user is required")
	}
	if c.DB.Password == "" {
		problems = append(problems, "db.password is required")
	}
	if strings.TrimSpace(c.DB.Name) == "" {
		problems = append(problems, "db.name is required")
	}
	if c.DB.MaxOpenConns <= 0 {
		problems = append(problems, "db.max_open_conns must be positive")
	}
	if c.DB.MaxIdleConns < 0 {
		problems = append(problems, "db.max_idle_conns must not be negative")
	}
	if c.DB.MaxOpenConns > 0 && c.DB.MaxIdleConns > c.DB.MaxOpenConns {
		problems = append(problems, "db.max_idle_conns must not exceed db.max_open_conns")
	}
	validatePositiveDuration(&problems, "db.conn_max_lifetime", c.DB.ConnMaxLifetime)
	validatePositiveDuration(&problems, "db.conn_max_idle_time", c.DB.ConnMaxIdleTime)
	validatePositiveDuration(&problems, "db.connect_timeout", c.DB.ConnectTimeout)

	if strings.TrimSpace(c.Redis.Addr) == "" {
		problems = append(problems, "redis.addr is required")
	}
	if c.Redis.DB < 0 {
		problems = append(problems, "redis.db must not be negative")
	}
	if c.Redis.PoolSize <= 0 {
		problems = append(problems, "redis.pool_size must be positive")
	}
	validatePositiveDuration(&problems, "redis.dial_timeout", c.Redis.DialTimeout)
	validatePositiveDuration(&problems, "redis.read_timeout", c.Redis.ReadTimeout)
	validatePositiveDuration(&problems, "redis.write_timeout", c.Redis.WriteTimeout)

	if len(c.JWT.Secret) < 32 {
		problems = append(problems, "jwt.secret must be at least 32 characters")
	}
	validatePositiveDuration(&problems, "jwt.ttl", c.JWT.TTL)
	validatePositiveDuration(&problems, "cache.ttl", c.Cache.TTL)

	switch strings.ToLower(strings.TrimSpace(c.Log.Level)) {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, "log.level must be one of: debug, info, warn, error")
	}
	switch strings.ToLower(strings.TrimSpace(c.Log.Format)) {
	case "json", "text":
	default:
		problems = append(problems, "log.format must be one of: json, text")
	}

	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

func validatePort(problems *[]string, name string, port int) {
	if port < 1 || port > 65535 {
		*problems = append(*problems, name+" must be between 1 and 65535")
	}
}

func validatePositiveDuration(problems *[]string, name string, value time.Duration) {
	if value <= 0 {
		*problems = append(*problems, name+" must be positive")
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.read_header_timeout", 5*time.Second)
	v.SetDefault("app.read_timeout", 15*time.Second)
	v.SetDefault("app.write_timeout", 15*time.Second)
	v.SetDefault("app.idle_timeout", 60*time.Second)
	v.SetDefault("app.shutdown_timeout", 10*time.Second)

	v.SetDefault("db.host", "mysql")
	v.SetDefault("db.port", 3306)
	v.SetDefault("db.user", "taskuser")
	v.SetDefault("db.password", "taskpass")
	v.SetDefault("db.name", "taskdb")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 25)
	v.SetDefault("db.conn_max_lifetime", 5*time.Minute)
	v.SetDefault("db.conn_max_idle_time", 5*time.Minute)
	v.SetDefault("db.connect_timeout", 5*time.Second)

	v.SetDefault("redis.addr", "redis:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.dial_timeout", 2*time.Second)
	v.SetDefault("redis.read_timeout", time.Second)
	v.SetDefault("redis.write_timeout", time.Second)

	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.ttl", 24*time.Hour)

	v.SetDefault("cache.ttl", 5*time.Minute)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}
