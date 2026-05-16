package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Config, error) {
	cfg := Default()

	resolvedPath := strings.TrimSpace(path)
	if resolvedPath == "" {
		resolvedPath = DefaultPath
	}

	if err := loadFromFile(resolvedPath, &cfg); err != nil {
		return Config{}, err
	}

	if err := overrideFromEnv(&cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func loadFromFile(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config file %s: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("decode config file %s: %w", path, err)
	}

	return nil
}

func overrideFromEnv(cfg *Config) error {
	if v := strings.TrimSpace(os.Getenv("AUTH_HTTP_ADDR")); v != "" {
		cfg.HTTP.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("DATABASE_URL")); v != "" {
		cfg.Postgres.URL = v
	}
	if v := strings.TrimSpace(os.Getenv("JWT_SECRET")); v != "" {
		cfg.JWT.Secret = v
	}
	if v := strings.TrimSpace(os.Getenv("PROJECT_SERVICE_URL")); v != "" {
		cfg.Admin.ProjectServiceURL = v
	}
	if v := strings.TrimSpace(os.Getenv("STATIC_DIR")); v != "" {
		cfg.StaticDir = v
	}
	if v := strings.TrimSpace(os.Getenv("AUTH_SHUTDOWN_TIMEOUT")); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse AUTH_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = parsed
	}

	return nil
}
