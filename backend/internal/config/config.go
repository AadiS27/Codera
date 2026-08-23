package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env             string
	Port            string
	LogLevel        string
	ShutdownTimeout time.Duration
	CompileTimeout  time.Duration
	RunTimeout      time.Duration
	MaxStdoutBytes  int64
	MaxStderrBytes  int64

	SandboxRuntime     string
	JavaSandboxImage   string
	ExecutionMemory    string
	ExecutionCPUs      string
	ExecutionPidsLimit int64
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

	var err error
	cfg.ShutdownTimeout, err = parseDuration("SHUTDOWN_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.CompileTimeout, err = parseDuration("COMPILE_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.RunTimeout, err = parseDuration("RUN_TIMEOUT", 2*time.Second)
	if err != nil {
		return nil, err
	}

	cfg.MaxStdoutBytes, err = parseInt("MAX_STDOUT_BYTES", 65536)
	if err != nil {
		return nil, err
	}
	cfg.MaxStderrBytes, err = parseInt("MAX_STDERR_BYTES", 65536)
	if err != nil {
		return nil, err
	}

	cfg.SandboxRuntime = os.Getenv("SANDBOX_RUNTIME")
	if cfg.SandboxRuntime == "" {
		cfg.SandboxRuntime = "docker"
	}
	cfg.JavaSandboxImage = os.Getenv("JAVA_SANDBOX_IMAGE")
	if cfg.JavaSandboxImage == "" {
		cfg.JavaSandboxImage = "code-executor-java:latest"
	}
	cfg.ExecutionMemory = os.Getenv("EXECUTION_MEMORY")
	if cfg.ExecutionMemory == "" {
		cfg.ExecutionMemory = "256m"
	}
	cfg.ExecutionCPUs = os.Getenv("EXECUTION_CPUS")
	if cfg.ExecutionCPUs == "" {
		cfg.ExecutionCPUs = "1.0"
	}
	cfg.ExecutionPidsLimit, err = parseInt("EXECUTION_PIDS_LIMIT", 64)
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return duration, nil
}

func parseInt(key string, fallback int64) (int64, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func (c *Config) validate() error {
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		return errors.New("ENV must be development, production, or test")
	}
	if c.CompileTimeout <= 0 {
		return errors.New("COMPILE_TIMEOUT must be positive")
	}
	if c.RunTimeout <= 0 {
		return errors.New("RUN_TIMEOUT must be positive")
	}
	if c.MaxStdoutBytes <= 0 {
		return errors.New("MAX_STDOUT_BYTES must be positive")
	}
	if c.MaxStderrBytes <= 0 {
		return errors.New("MAX_STDERR_BYTES must be positive")
	}
	if c.SandboxRuntime == "" {
		return errors.New("SANDBOX_RUNTIME cannot be empty")
	}
	if c.JavaSandboxImage == "" {
		return errors.New("JAVA_SANDBOX_IMAGE cannot be empty")
	}
	if c.ExecutionPidsLimit <= 0 {
		return errors.New("EXECUTION_PIDS_LIMIT must be positive")
	}
	return nil
}
