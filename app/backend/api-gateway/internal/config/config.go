package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "configs/config.local.yaml"

type Config struct {
	HTTP            HTTPConfig    `yaml:"http"`
	AuthService     BackendConfig `yaml:"auth_service"`
	ProjectService  BackendConfig `yaml:"project_service"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

// BackendConfig holds the gRPC address of a downstream service.
type BackendConfig struct {
	GRPCAddr string `yaml:"grpc_addr"`
}

func Default() Config {
	return Config{
		HTTP:            HTTPConfig{Addr: ":8000"},
		AuthService:     BackendConfig{GRPCAddr: "localhost:9090"},
		ProjectService:  BackendConfig{GRPCAddr: "localhost:9091"},
		ShutdownTimeout: 10 * time.Second,
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	p := strings.TrimSpace(path)
	if p == "" {
		p = DefaultPath
	}

	f, err := os.Open(p)
	if err != nil {
		return Config{}, fmt.Errorf("open config %s: %w", p, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", p, err)
	}

	overrideFromEnv(&cfg)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("GATEWAY_HTTP_ADDR"); v != "" {
		cfg.HTTP.Addr = ":" + strings.TrimPrefix(strings.TrimSpace(v), ":")
	}
	if v := os.Getenv("AUTH_SERVICE_GRPC_ADDR"); v != "" {
		cfg.AuthService.GRPCAddr = strings.TrimSpace(v)
	}
	if v := os.Getenv("PROJECT_SERVICE_GRPC_ADDR"); v != "" {
		cfg.ProjectService.GRPCAddr = strings.TrimSpace(v)
	}
}

func (c Config) Validate() error {
	switch {
	case c.HTTP.Addr == "":
		return errors.New("http.addr is required")
	case c.AuthService.GRPCAddr == "":
		return errors.New("auth_service.grpc_addr is required")
	case c.ProjectService.GRPCAddr == "":
		return errors.New("project_service.grpc_addr is required")
	default:
		return nil
	}
}
