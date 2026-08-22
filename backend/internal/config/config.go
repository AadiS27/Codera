package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Env             string
	Port            string
	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:      os.Getenv("ENV"),
		Port:     os.Getenv("PORT"),
		LogLevel: os.Getenv("LOG_LEVEL"),
	}

	if cfg.Env == "" {
		cfg.Env = "development"
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	timeoutStr := os.Getenv("SHUTDOWN_TIMEOUT")
	if timeoutStr == "" {
		cfg.ShutdownTimeout = 5 * time.Second
	} else {
		duration, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = duration
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		return errors.New("ENV must be development, production, or test")
	}
	return nil
}
