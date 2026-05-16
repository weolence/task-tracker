package config

import (
	"errors"
	"fmt"
	"time"
)

const DefaultPath = "configs/config.local.yaml"

type Config struct {
	HTTP            HTTPConfig        `yaml:"http"`
	Postgres        PostgresConfig    `yaml:"postgres"`
	AuthService     AuthServiceConfig `yaml:"auth_service"`
	StaticDir       string            `yaml:"static_dir"`
	ShutdownTimeout time.Duration     `yaml:"shutdown_timeout"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type PostgresConfig struct {
	URL string `yaml:"url"`
}

type AuthServiceConfig struct {
	URL string `yaml:"url"`
}

func Default() Config {
	return Config{
		HTTP:            HTTPConfig{Addr: ":8081"},
		AuthService:     AuthServiceConfig{URL: "http://localhost:8080"},
		StaticDir:       "static",
		ShutdownTimeout: 10 * time.Second,
	}
}

func (c Config) Validate() error {
	switch {
	case c.HTTP.Addr == "":
		return errors.New("http.addr is required")
	case c.Postgres.URL == "":
		return errors.New("postgres.url is required")
	case c.ShutdownTimeout <= 0:
		return errors.New("shutdown_timeout must be greater than zero")
	default:
		return nil
	}
}

func (c Config) String() string {
	return fmt.Sprintf(
		"http=%s postgres=%t auth_service_url=%s shutdown_timeout=%s",
		c.HTTP.Addr,
		c.Postgres.URL != "",
		c.AuthService.URL,
		c.ShutdownTimeout,
	)
}
