package mcp

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/OpenCHAMI/openchami-mcp/internal/backends"
	"github.com/OpenCHAMI/openchami-mcp/internal/config"
	"github.com/OpenCHAMI/openchami-mcp/internal/tools"
)

func TestHandleInitializeAndToolsList(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
	})

	response, exit, err := server.handle(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	if exit {
		t.Fatal("initialize should not request exit")
	}
	result := response["result"].(map[string]any)
	if got := result["protocolVersion"]; got != protocolVersion {
		t.Fatalf("unexpected protocolVersion: %#v", got)
	}

	response, exit, err = server.handle(context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if exit {
		t.Fatal("tools/list should not request exit")
	}
	toolsResult := response["result"].(map[string]any)
	toolList := toolsResult["tools"].([]map[string]any)
	if len(toolList) == 0 {
		t.Fatal("expected tool list to be non-empty")
	}
}

func TestWriteGateBlocksWriteTool(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
		EnableWrites:    false,
	})

	response, _, err := server.handle(context.Background(), map[string]any{
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

	errObject := response["error"].(map[string]any)
	if got := errObject["code"]; got != -32001.0 && got != -32001 {
		t.Fatalf("expected permission error code, got %#v", got)
	}
}

func TestReadOnlyToolCallSucceeds(t *testing.T) {
	server := newTestServer(config.Config{
		Mode:            config.ModeReadOnly,
		Framing:         config.FramingAuto,
		LogLevel:        "error",
		RequireWriteAck: true,
	})

	response, _, err := server.handle(context.Background(), map[string]any{
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

	result := response["result"].(map[string]any)
	content := result["content"].([]map[string]any)
	if len(content) != 1 {
		t.Fatalf("expected single content item, got %d", len(content))
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

	return NewServer(cfg, registry, logger, nil, nil)
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

var (
	_ backends.SMDBackend          = fakeSMDBackend{}
	_ backends.BSSBackend          = fakeBSSBackend{}
	_ backends.CloudInitBackend    = fakeCloudBackend{}
	_ backends.PowerControlBackend = fakePowerBackend{}
	_ backends.KeaBackend          = fakeKeaBackend{}
	_ backends.CoreDHCPBackend     = fakeCoreBackend{}
)
