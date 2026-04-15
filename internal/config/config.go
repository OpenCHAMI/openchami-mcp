package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ModeReadOnly  = "read-only"
	ModeReadWrite = "read-write"

	TransportStdio = "stdio"
	TransportHTTP  = "http"

	FramingAuto          = "auto"
	FramingJSONL         = "jsonl"
	FramingContentLength = "content-length"
)

var trueValues = map[string]struct{}{
	"1":    {},
	"true": {},
	"yes":  {},
	"on":   {},
}

type HTTPService struct {
	BaseURL     string
	Timeout     time.Duration
	Retries     int
	BearerToken string
	TLSCAFile   string
	TLSInsecure bool
}

type KeaService struct {
	HTTPService
	DefaultServices []string
}

type HTTPTransport struct {
	ListenAddr     string
	BasePath       string
	HealthPath     string
	AllowedOrigins []string
	BearerToken    string
}

type Config struct {
	Mode            string
	Transport       string
	Framing         string
	LogLevel        string
	RequireWriteAck bool
	EnableWrites    bool
	HTTP            HTTPTransport

	SMD          HTTPService
	BSS          HTTPService
	CloudInit    HTTPService
	PowerControl HTTPService
	Kea          KeaService
	CoreDHCP     HTTPService
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		Mode:            envString("OPENCHAMI_MCP_MODE", ModeReadOnly),
		Transport:       envString("OPENCHAMI_MCP_TRANSPORT", TransportStdio),
		Framing:         envString("OPENCHAMI_MCP_FRAMING", FramingAuto),
		LogLevel:        envString("OPENCHAMI_MCP_LOG_LEVEL", "info"),
		RequireWriteAck: envBool("OPENCHAMI_MCP_REQUIRE_WRITE_ACK", true),
		EnableWrites:    envBool("OPENCHAMI_MCP_ENABLE_WRITES", false),
		HTTP: HTTPTransport{
			ListenAddr:     envString("OPENCHAMI_MCP_LISTEN_ADDR", "127.0.0.1:8081"),
			BasePath:       envString("OPENCHAMI_MCP_BASE_PATH", "/mcp"),
			HealthPath:     envString("OPENCHAMI_MCP_HEALTH_PATH", "/healthz"),
			AllowedOrigins: envCSV("OPENCHAMI_MCP_ALLOWED_ORIGINS"),
			BearerToken:    envString("OPENCHAMI_MCP_HTTP_BEARER_TOKEN", ""),
		},
		SMD:          loadHTTPService("OPENCHAMI_MCP_SMD"),
		BSS:          loadHTTPService("OPENCHAMI_MCP_BSS"),
		CloudInit:    loadHTTPService("OPENCHAMI_MCP_CLOUD_INIT"),
		PowerControl: loadHTTPService("OPENCHAMI_MCP_POWER_CONTROL"),
		Kea: KeaService{
			HTTPService:     loadHTTPService("OPENCHAMI_MCP_KEA"),
			DefaultServices: envCSV("OPENCHAMI_MCP_KEA_DEFAULT_SERVICES"),
		},
		CoreDHCP: loadHTTPService("OPENCHAMI_MCP_COREDHCP"),
	}

	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModeReadOnly, ModeReadWrite:
	default:
		return fmt.Errorf("unsupported OPENCHAMI_MCP_MODE %q", c.Mode)
	}

	switch c.Transport {
	case TransportStdio, TransportHTTP:
	default:
		return fmt.Errorf("unsupported OPENCHAMI_MCP_TRANSPORT %q", c.Transport)
	}

	switch c.Framing {
	case FramingAuto, FramingJSONL, FramingContentLength:
	default:
		return fmt.Errorf("unsupported OPENCHAMI_MCP_FRAMING %q", c.Framing)
	}

	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported OPENCHAMI_MCP_LOG_LEVEL %q", c.LogLevel)
	}

	for _, svc := range []struct {
		name string
		cfg  HTTPService
	}{
		{name: "smd", cfg: c.SMD},
		{name: "bss", cfg: c.BSS},
		{name: "cloud-init", cfg: c.CloudInit},
		{name: "power-control", cfg: c.PowerControl},
		{name: "kea", cfg: c.Kea.HTTPService},
		{name: "coredhcp", cfg: c.CoreDHCP},
	} {
		if svc.cfg.Timeout <= 0 {
			return fmt.Errorf("%s timeout must be positive", svc.name)
		}
		if svc.cfg.Retries < 0 {
			return fmt.Errorf("%s retries cannot be negative", svc.name)
		}
	}

	if c.HTTP.ListenAddr == "" {
		return fmt.Errorf("OPENCHAMI_MCP_LISTEN_ADDR cannot be empty")
	}

	if !strings.HasPrefix(c.HTTP.BasePath, "/") {
		return fmt.Errorf("OPENCHAMI_MCP_BASE_PATH must start with '/'")
	}
	if !strings.HasPrefix(c.HTTP.HealthPath, "/") {
		return fmt.Errorf("OPENCHAMI_MCP_HEALTH_PATH must start with '/'")
	}
	if c.HTTP.BasePath == c.HTTP.HealthPath {
		return fmt.Errorf("OPENCHAMI_MCP_BASE_PATH and OPENCHAMI_MCP_HEALTH_PATH must differ")
	}

	return nil
}

func loadHTTPService(prefix string) HTTPService {
	return HTTPService{
		BaseURL:     envString(prefix+"_URL", ""),
		Timeout:     envDuration(prefix+"_TIMEOUT", 10*time.Second),
		Retries:     envInt(prefix+"_RETRIES", 1),
		BearerToken: envString(prefix+"_BEARER_TOKEN", ""),
		TLSCAFile:   envString(prefix+"_TLS_CA_FILE", ""),
		TLSInsecure: envBool(prefix+"_TLS_INSECURE_SKIP_VERIFY", false),
	}
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	_, ok := trueValues[value]
	return ok
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envCSV(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
