package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            int           `env:"PORT" default:"8080"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"10s"`

	LogLevel  string `env:"LOG_LEVEL" default:"info"`
	LogFormat string `env:"LOG_FORMAT" default:"json"`

	K8sTimeout time.Duration `env:"K8S_TIMEOUT" default:"30s"`

	NodeName string `env:"NODE_NAME" default:""`

	EnableMetrics bool `env:"ENABLE_METRICS" default:"true"`

	PodRestartThreshold int `env:"POD_RESTART_THRESHOLD" default:"5"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                getEnvAsInt("PORT", 8080),
		ReadTimeout:         getEnvAsDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:        getEnvAsDuration("WRITE_TIMEOUT", 10*time.Second),
		ShutdownTimeout:     getEnvAsDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		K8sTimeout:          getEnvAsDuration("K8S_TIMEOUT", 30*time.Second),
		NodeName:            getEnv("NODE_NAME", ""),
		EnableMetrics:       getEnvAsBool("ENABLE_METRICS", true),
		PodRestartThreshold: getEnvAsInt("POD_RESTART_THRESHOLD", 5),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.LogLevel != "debug" && c.LogLevel != "info" &&
		c.LogLevel != "warn" && c.LogLevel != "error" {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	if c.LogFormat != "json" && c.LogFormat != "text" {
		return fmt.Errorf("invalid log format: %s", c.LogFormat)
	}

	if c.PodRestartThreshold < 0 {
		return fmt.Errorf("invalid pod restart threshold: %d (must be >= 0)", c.PodRestartThreshold)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
