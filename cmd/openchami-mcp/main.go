package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/OpenCHAMI/openchami-mcp/internal/backends"
	"github.com/OpenCHAMI/openchami-mcp/internal/clients/httpjson"
	"github.com/OpenCHAMI/openchami-mcp/internal/config"
	"github.com/OpenCHAMI/openchami-mcp/internal/mcp"
	"github.com/OpenCHAMI/openchami-mcp/internal/tools"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	mode := flag.String("mode", cfg.Mode, "MCP safety mode: read-only or read-write")
	transport := flag.String("transport", cfg.Transport, "MCP transport: stdio or http")
	framing := flag.String("framing", cfg.Framing, "MCP framing mode: auto, jsonl, or content-length")
	logLevel := flag.String("log-level", cfg.LogLevel, "log level: debug, info, warn, error")
	listenAddr := flag.String("listen-addr", cfg.HTTP.ListenAddr, "HTTP listen address when transport=http")
	basePath := flag.String("base-path", cfg.HTTP.BasePath, "HTTP MCP endpoint path when transport=http")
	healthPath := flag.String("health-path", cfg.HTTP.HealthPath, "HTTP health endpoint path when transport=http")
	allowedOrigins := flag.String("allowed-origins", strings.Join(cfg.HTTP.AllowedOrigins, ","), "comma-separated allowed HTTP origins when transport=http")
	flag.Parse()

	cfg.Mode = *mode
	cfg.Transport = *transport
	cfg.Framing = *framing
	cfg.LogLevel = *logLevel
	cfg.HTTP.ListenAddr = *listenAddr
	cfg.HTTP.BasePath = *basePath
	cfg.HTTP.HealthPath = *healthPath
	if strings.TrimSpace(*allowedOrigins) == "" {
		cfg.HTTP.AllowedOrigins = nil
	} else {
		cfg.HTTP.AllowedOrigins = splitCSV(*allowedOrigins)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	smdClient, err := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.SMD.BaseURL,
		Timeout:     cfg.SMD.Timeout,
		Retries:     cfg.SMD.Retries,
		BearerToken: cfg.SMD.BearerToken,
		TLSCAFile:   cfg.SMD.TLSCAFile,
		TLSInsecure: cfg.SMD.TLSInsecure,
	})
	if err != nil {
		logger.Error("failed to create SMD client", "error", err)
		os.Exit(1)
	}
	bssClient, err := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.BSS.BaseURL,
		Timeout:     cfg.BSS.Timeout,
		Retries:     cfg.BSS.Retries,
		BearerToken: cfg.BSS.BearerToken,
		TLSCAFile:   cfg.BSS.TLSCAFile,
		TLSInsecure: cfg.BSS.TLSInsecure,
	})
	if err != nil {
		logger.Error("failed to create BSS client", "error", err)
		os.Exit(1)
	}
	cloudInitClient, err := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.CloudInit.BaseURL,
		Timeout:     cfg.CloudInit.Timeout,
		Retries:     cfg.CloudInit.Retries,
		BearerToken: cfg.CloudInit.BearerToken,
		TLSCAFile:   cfg.CloudInit.TLSCAFile,
		TLSInsecure: cfg.CloudInit.TLSInsecure,
	})
	if err != nil {
		logger.Error("failed to create cloud-init client", "error", err)
		os.Exit(1)
	}
	powerControlClient, err := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.PowerControl.BaseURL,
		Timeout:     cfg.PowerControl.Timeout,
		Retries:     cfg.PowerControl.Retries,
		BearerToken: cfg.PowerControl.BearerToken,
		TLSCAFile:   cfg.PowerControl.TLSCAFile,
		TLSInsecure: cfg.PowerControl.TLSInsecure,
	})
	if err != nil {
		logger.Error("failed to create power-control client", "error", err)
		os.Exit(1)
	}
	keaClient, err := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.Kea.BaseURL,
		Timeout:     cfg.Kea.Timeout,
		Retries:     cfg.Kea.Retries,
		BearerToken: cfg.Kea.BearerToken,
		TLSCAFile:   cfg.Kea.TLSCAFile,
		TLSInsecure: cfg.Kea.TLSInsecure,
	})
	if err != nil {
		logger.Error("failed to create Kea client", "error", err)
		os.Exit(1)
	}
	coredhcpClient, err := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.CoreDHCP.BaseURL,
		Timeout:     cfg.CoreDHCP.Timeout,
		Retries:     cfg.CoreDHCP.Retries,
		BearerToken: cfg.CoreDHCP.BearerToken,
		TLSCAFile:   cfg.CoreDHCP.TLSCAFile,
		TLSInsecure: cfg.CoreDHCP.TLSInsecure,
	})
	if err != nil {
		logger.Error("failed to create CoreDHCP client", "error", err)
		os.Exit(1)
	}

	registry := tools.NewRegistry(tools.Dependencies{
		Config: cfg,
		Logger: logger,
		SMD:    backends.NewSMDBackend(smdClient),
		BSS:    backends.NewBSSBackend(bssClient),
		Cloud:  backends.NewCloudInitBackend(cloudInitClient),
		Power:  backends.NewPowerControlBackend(powerControlClient),
		Kea:    backends.NewKeaBackend(keaClient, cfg.Kea.DefaultServices),
		Core:   backends.NewCoreDHCPBackend(coredhcpClient),
	})

	server := mcp.NewServer(cfg, registry, logger)
	var serveErr error
	switch cfg.Transport {
	case config.TransportHTTP:
		serveErr = server.ListenAndServeHTTP(ctx)
	default:
		serveErr = server.ServeStdio(ctx, os.Stdin, os.Stdout)
	}
	if serveErr != nil {
		logger.Error("server exited with error", "error", serveErr)
		os.Exit(1)
	}
}

func splitCSV(raw string) []string {
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

func newLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slogLevel,
	}))
}
