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
	AppPort  string        `mapstructure:"app_port"`
	DB       DB            `mapstructure:"db"`
	Redis    Redis         `mapstructure:"redis"`
	JWT      JWT           `mapstructure:"jwt"`
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
}

// DB holds MySQL connection settings.
type DB struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

// Redis holds Redis connection settings.
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWT holds token settings.
type JWT struct {
	Secret string        `mapstructure:"secret"`
	TTL    time.Duration `mapstructure:"ttl"`
}

// Load reads configuration with the following precedence:
// environment variables > config file > defaults.
//
// By default Load looks for config.* in the current directory and ./config.
// CONFIG_FILE can point to a specific YAML, JSON or TOML file.
func Load() (Config, error) {
	return load(os.Getenv("CONFIG_FILE"))
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
	if cfg.JWT.Secret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app_port", "8080")

	v.SetDefault("db.host", "mysql")
	v.SetDefault("db.port", "3306")
	v.SetDefault("db.user", "taskuser")
	v.SetDefault("db.password", "taskpass")
	v.SetDefault("db.name", "taskdb")

	v.SetDefault("redis.addr", "redis:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.ttl", 24*time.Hour)

	v.SetDefault("cache_ttl", 5*time.Minute)
}
