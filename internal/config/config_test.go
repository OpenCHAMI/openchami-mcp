package config

import (
	"testing"
	"time"
)

func TestLoadFromEnvHTTPDefaults(t *testing.T) {
	t.Setenv("OPENCHAMI_MCP_TRANSPORT", "http")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if cfg.Transport != TransportHTTP {
		t.Fatalf("unexpected transport: %q", cfg.Transport)
	}
	if cfg.HTTP.ListenAddr != "127.0.0.1:8081" {
		t.Fatalf("unexpected listen addr: %q", cfg.HTTP.ListenAddr)
	}
	if cfg.HTTP.BasePath != "/mcp" {
		t.Fatalf("unexpected base path: %q", cfg.HTTP.BasePath)
	}
	if cfg.HTTP.HealthPath != "/healthz" {
		t.Fatalf("unexpected health path: %q", cfg.HTTP.HealthPath)
	}
}

func TestLoadFromEnvAllowedOrigins(t *testing.T) {
	t.Setenv("OPENCHAMI_MCP_ALLOWED_ORIGINS", "http://localhost:3000, https://example.com")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if len(cfg.HTTP.AllowedOrigins) != 2 {
		t.Fatalf("unexpected allowed origins: %#v", cfg.HTTP.AllowedOrigins)
	}
}

func TestLoadFromEnvServiceTLSSettings(t *testing.T) {
	t.Setenv("OPENCHAMI_MCP_SMD_TLS_CA_FILE", "/tmp/smd-ca.pem")
	t.Setenv("OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY", "true")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if cfg.SMD.TLSCAFile != "/tmp/smd-ca.pem" {
		t.Fatalf("unexpected SMD CA file: %q", cfg.SMD.TLSCAFile)
	}
	if !cfg.SMD.TLSInsecure {
		t.Fatal("expected SMD TLS insecure skip verify to be enabled")
	}
}

func TestValidateRejectsInvalidHTTPPaths(t *testing.T) {
	cfg := validConfig()
	cfg.Transport = TransportHTTP
	cfg.HTTP.BasePath = "mcp"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsInvalidTransport(t *testing.T) {
	cfg := validConfig()
	cfg.Transport = "ssh"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func validConfig() Config {
	timeout := 10 * time.Second
	return Config{
		Mode:      ModeReadOnly,
		Transport: TransportStdio,
		Framing:   FramingAuto,
		LogLevel:  "info",
		HTTP: HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
		SMD:          HTTPService{Timeout: timeout, Retries: 1},
		BSS:          HTTPService{Timeout: timeout, Retries: 1},
		CloudInit:    HTTPService{Timeout: timeout, Retries: 1},
		PowerControl: HTTPService{Timeout: timeout, Retries: 1},
		Kea:          KeaService{HTTPService: HTTPService{Timeout: timeout, Retries: 1}},
		CoreDHCP:     HTTPService{Timeout: timeout, Retries: 1},
	}
}
