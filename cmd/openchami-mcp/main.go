package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

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
	framing := flag.String("framing", cfg.Framing, "MCP framing mode: auto, jsonl, or content-length")
	logLevel := flag.String("log-level", cfg.LogLevel, "log level: debug, info, warn, error")
	flag.Parse()

	cfg.Mode = *mode
	cfg.Framing = *framing
	cfg.LogLevel = *logLevel
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)

	smdClient := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.SMD.BaseURL,
		Timeout:     cfg.SMD.Timeout,
		Retries:     cfg.SMD.Retries,
		BearerToken: cfg.SMD.BearerToken,
	})
	bssClient := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.BSS.BaseURL,
		Timeout:     cfg.BSS.Timeout,
		Retries:     cfg.BSS.Retries,
		BearerToken: cfg.BSS.BearerToken,
	})
	cloudInitClient := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.CloudInit.BaseURL,
		Timeout:     cfg.CloudInit.Timeout,
		Retries:     cfg.CloudInit.Retries,
		BearerToken: cfg.CloudInit.BearerToken,
	})
	powerControlClient := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.PowerControl.BaseURL,
		Timeout:     cfg.PowerControl.Timeout,
		Retries:     cfg.PowerControl.Retries,
		BearerToken: cfg.PowerControl.BearerToken,
	})
	keaClient := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.Kea.BaseURL,
		Timeout:     cfg.Kea.Timeout,
		Retries:     cfg.Kea.Retries,
		BearerToken: cfg.Kea.BearerToken,
	})
	coredhcpClient := httpjson.NewClient(httpjson.Config{
		BaseURL:     cfg.CoreDHCP.BaseURL,
		Timeout:     cfg.CoreDHCP.Timeout,
		Retries:     cfg.CoreDHCP.Retries,
		BearerToken: cfg.CoreDHCP.BearerToken,
	})

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

	server := mcp.NewServer(cfg, registry, logger, os.Stdin, os.Stdout)
	if err := server.Serve(context.Background()); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
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
