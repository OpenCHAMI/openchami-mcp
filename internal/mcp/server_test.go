package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/OpenCHAMI/openchami-mcp/internal/config"
	"github.com/OpenCHAMI/openchami-mcp/internal/tools"
)

func TestHandleInitializeAndToolsList(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportStdio,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
	})

	response, err := server.HandleMessage(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": legacyProtocolVersion,
		},
	})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	if response.Exit {
		t.Fatal("initialize should not request exit")
	}
	result := response.Messages[0]["result"].(map[string]any)
	if got := result["protocolVersion"]; got != legacyProtocolVersion {
		t.Fatalf("unexpected protocolVersion: %#v", got)
	}

	response, err = server.HandleMessage(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if response.Exit {
		t.Fatal("tools/list should not request exit")
	}
	toolsResult := response.Messages[0]["result"].(map[string]any)
	toolList := toolsResult["tools"].([]map[string]any)
	if len(toolList) == 0 {
		t.Fatal("expected tool list to be non-empty")
	}
}

func TestHandleInitializeDefaultsToPreferredProtocol(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportStdio,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
	})

	response, err := server.HandleMessage(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	result := response.Messages[0]["result"].(map[string]any)
	if got := result["protocolVersion"]; got != preferredProtocolVersion {
		t.Fatalf("unexpected protocolVersion: %#v", got)
	}
}

func TestWriteGateBlocksWriteTool(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportStdio,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
		EnableWrites:    false,
	})

	response, err := server.HandleMessage(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "hsm_create_group",
			"arguments": map[string]any{
				"label": "test-group",
			},
		},
	})
	if err != nil {
		t.Fatalf("tools/call failed: %v", err)
	}

	errObject := response.Messages[0]["error"].(map[string]any)
	if got := errObject["code"]; got != -32001.0 && got != -32001 {
		t.Fatalf("expected permission error code, got %#v", got)
	}
}

func TestReadOnlyToolCallSucceeds(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportStdio,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
	})

	response, err := server.HandleMessage(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      9,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "bss_service_status",
			"arguments": map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("tools/call failed: %v", err)
	}

	result := response.Messages[0]["result"].(map[string]any)
	content := result["content"].([]map[string]any)
	if len(content) != 1 {
		t.Fatalf("expected single content item, got %d", len(content))
	}
}

func TestServeStdioPreservesJSONLResponses(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportStdio,
		Framing:         config.FramingJSONL,
		LogLevel:        "error",
		RequireWriteAck: true,
	})

	var output bytes.Buffer
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\n")
	if err := server.ServeStdio(context.Background(), input, &output); err != nil {
		t.Fatalf("ServeStdio failed: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatalf("failed to decode stdio response: %v", err)
	}
	if response["id"] != float64(1) {
		t.Fatalf("unexpected response id: %#v", response["id"])
	}
}

func TestHTTPHandlerInitialize(t *testing.T) {
	server := newHTTPTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportHTTP,
		LogLevel:        "error",
		RequireWriteAck: true,
		HTTP: config.HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	req.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	result := response["result"].(map[string]any)
	if got := result["protocolVersion"]; got != preferredProtocolVersion {
		t.Fatalf("unexpected protocol version: %#v", got)
	}
}

func TestHTTPHandlerToolsList(t *testing.T) {
	server := newHTTPTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportHTTP,
		LogLevel:        "error",
		RequireWriteAck: true,
		HTTP: config.HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	req.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	result := response["result"].(map[string]any)
	toolList := result["tools"].([]any)
	if len(toolList) == 0 {
		t.Fatal("expected tool list to be non-empty")
	}
}

func TestHTTPHandlerNotificationAccepted(t *testing.T) {
	server := newHTTPTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportHTTP,
		LogLevel:        "error",
		RequireWriteAck: true,
		HTTP: config.HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	req.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", recorder.Body.String())
	}
}

func TestHTTPHandlerRejectsInvalidJSON(t *testing.T) {
	server := newHTTPTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportHTTP,
		LogLevel:        "error",
		RequireWriteAck: true,
		HTTP: config.HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader(`{"jsonrpc":`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	req.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestHTTPHandlerRejectsMissingOriginFromRemoteClient(t *testing.T) {
	server := newHTTPTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportHTTP,
		LogLevel:        "error",
		RequireWriteAck: true,
		HTTP: config.HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.44:12345"
	recorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestHTTPHandlerGetReturnsMethodNotAllowed(t *testing.T) {
	server := newHTTPTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Transport:       config.TransportHTTP,
		LogLevel:        "error",
		RequireWriteAck: true,
		HTTP: config.HTTPTransport{
			ListenAddr: "127.0.0.1:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://localhost/mcp", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()

	server.HTTPHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestValidateHTTPServerConfigRequiresAuthForRemoteReadWrite(t *testing.T) {
	err := validateHTTPServerConfig(config.Config{
		Mode:      config.ModeReadWrite,
		Transport: config.TransportHTTP,
		HTTP: config.HTTPTransport{
			ListenAddr: "0.0.0.0:8081",
			BasePath:   "/mcp",
			HealthPath: "/healthz",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func newTestServer(cfg config.Config) *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := tools.NewRegistry(tools.Dependencies{
		Config: cfg,
		Logger: logger,
		SMD:    fakeSMDBackend{},
		BSS:    fakeBSSBackend{},
		Cloud:  fakeCloudBackend{},
		Power:  fakePowerBackend{},
		Kea:    fakeKeaBackend{},
		Core:   fakeCoreBackend{},
	})

	return NewServer(cfg, registry, logger)
}

func newHTTPTestServer(cfg config.Config) *Server {
	if cfg.Transport == "" {
		cfg.Transport = config.TransportHTTP
	}
	return newTestServer(cfg)
}

type fakeSMDBackend struct{}

func (fakeSMDBackend) Configured() bool                            { return true }
func (fakeSMDBackend) BaseURL() string                             { return "http://smd" }
func (fakeSMDBackend) Ready(context.Context) (any, error)          { return map[string]any{"ready": true}, nil }
func (fakeSMDBackend) ListComponents(context.Context) (any, error) { return []any{}, nil }
func (fakeSMDBackend) GetComponent(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (fakeSMDBackend) ListGroups(context.Context) (any, error)       { return []any{}, nil }
func (fakeSMDBackend) GetGroup(context.Context, string) (any, error) { return map[string]any{}, nil }
func (fakeSMDBackend) GetMembership(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (fakeSMDBackend) CreateGroup(context.Context, map[string]any) (any, error) {
	return map[string]any{"created": true}, nil
}
func (fakeSMDBackend) PatchGroup(context.Context, string, map[string]any) (any, error) {
	return map[string]any{"patched": true}, nil
}
func (fakeSMDBackend) DeleteGroup(context.Context, string) (any, error) {
	return map[string]any{"deleted": true}, nil
}
func (fakeSMDBackend) GetGroupMembers(context.Context, string) (any, error) {
	return map[string]any{"ids": []string{}}, nil
}
func (fakeSMDBackend) SetGroupMembers(context.Context, string, []string) (any, error) {
	return map[string]any{"set": true}, nil
}
func (fakeSMDBackend) AddGroupMember(context.Context, string, string) (any, error) {
	return map[string]any{"added": true}, nil
}
func (fakeSMDBackend) DeleteGroupMember(context.Context, string, string) (any, error) {
	return map[string]any{"deleted": true}, nil
}

type fakeBSSBackend struct{}

func (fakeBSSBackend) Configured() bool { return true }
func (fakeBSSBackend) BaseURL() string  { return "http://bss" }
func (fakeBSSBackend) ServiceStatus(context.Context) (any, error) {
	return map[string]any{"status": "ok"}, nil
}
func (fakeBSSBackend) GetBootParameters(context.Context) (any, error) { return map[string]any{}, nil }
func (fakeBSSBackend) GetBootScript(context.Context, url.Values) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSSBackend) ListHosts(context.Context) (any, error) { return map[string]any{}, nil }
func (fakeBSSBackend) PutBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSSBackend) PostBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSSBackend) PatchBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSSBackend) DeleteBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSSBackend) PostHosts(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}

type fakeCloudBackend struct{}

func (fakeCloudBackend) Configured() bool { return true }
func (fakeCloudBackend) BaseURL() string  { return "http://cloud-init" }
func (fakeCloudBackend) Version(context.Context) (any, error) {
	return map[string]any{"version": "test"}, nil
}
func (fakeCloudBackend) GetClusterDefaults(context.Context) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) SetClusterDefaults(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) ListGroups(context.Context) (any, error)       { return map[string]any{}, nil }
func (fakeCloudBackend) GetGroup(context.Context, string) (any, error) { return map[string]any{}, nil }
func (fakeCloudBackend) CreateGroup(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) UpdateGroup(context.Context, string, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) DeleteGroup(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) UpdateInstanceInfo(context.Context, string, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) GetMetaData(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) GetUserData(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) GetVendorData(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloudBackend) GetGroupUserData(context.Context, string, string) (any, error) {
	return map[string]any{}, nil
}

type fakePowerBackend struct{}

func (fakePowerBackend) Configured() bool { return true }
func (fakePowerBackend) BaseURL() string  { return "http://power-control" }
func (fakePowerBackend) Health(context.Context) (any, error) {
	return map[string]any{"status": "ok"}, nil
}
func (fakePowerBackend) GetPowerStatus(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakePowerBackend) CreateTransition(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}

type fakeKeaBackend struct{}

func (fakeKeaBackend) Configured() bool          { return true }
func (fakeKeaBackend) BaseURL() string           { return "http://kea" }
func (fakeKeaBackend) DefaultServices() []string { return []string{"dhcp4"} }
func (fakeKeaBackend) RunCommand(context.Context, string, map[string]any, []string) (any, error) {
	return map[string]any{"result": 0}, nil
}

type fakeCoreBackend struct{}

func (fakeCoreBackend) Configured() bool                    { return false }
func (fakeCoreBackend) BaseURL() string                     { return "" }
func (fakeCoreBackend) Status(context.Context) (any, error) { return map[string]any{}, nil }
