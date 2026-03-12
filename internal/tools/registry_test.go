package tools

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/OpenCHAMI/openchami-mcp/internal/backends"
	"github.com/OpenCHAMI/openchami-mcp/internal/config"
)

func TestHSMUpdateGroupMembersActionMapping(t *testing.T) {
	smd := &recordingSMDBackend{}
	registry := NewRegistry(Dependencies{
		Config: config.Config{},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SMD:    smd,
		BSS:    fakeBSS{},
		Cloud:  fakeCloud{},
		Power:  fakePower{},
		Kea:    fakeKea{},
		Core:   fakeCore{},
	})

	_, err := registry.Call(context.Background(), "hsm_update_group_members", map[string]any{
		"label":  "compute",
		"action": "replace",
		"members": []any{
			"x0c0s0b0n0",
			"x0c0s0b0n1",
		},
	})
	if err != nil {
		t.Fatalf("replace call failed: %v", err)
	}
	if smd.lastMethod != "set" {
		t.Fatalf("expected set method, got %q", smd.lastMethod)
	}

	_, err = registry.Call(context.Background(), "hsm_update_group_members", map[string]any{
		"label":  "compute",
		"action": "add",
		"member": "x0c0s0b0n2",
	})
	if err != nil {
		t.Fatalf("add call failed: %v", err)
	}
	if smd.lastMethod != "add" || smd.lastMembers[0] != "x0c0s0b0n2" {
		t.Fatalf("unexpected add call: %#v %#v", smd.lastMethod, smd.lastMembers)
	}

	_, err = registry.Call(context.Background(), "hsm_update_group_members", map[string]any{
		"label":  "compute",
		"action": "remove",
		"member": "x0c0s0b0n2",
	})
	if err != nil {
		t.Fatalf("remove call failed: %v", err)
	}
	if smd.lastMethod != "delete" {
		t.Fatalf("expected delete method, got %q", smd.lastMethod)
	}
}

func TestKeaDefaultServicesAreUsed(t *testing.T) {
	kea := &recordingKeaBackend{}
	registry := NewRegistry(Dependencies{
		Config: config.Config{},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SMD:    &recordingSMDBackend{},
		BSS:    fakeBSS{},
		Cloud:  fakeCloud{},
		Power:  fakePower{},
		Kea:    kea,
		Core:   fakeCore{},
	})

	_, err := registry.Call(context.Background(), "kea_status_get", map[string]any{})
	if err != nil {
		t.Fatalf("kea_status_get failed: %v", err)
	}
	if len(kea.services) != 1 || kea.services[0] != "dhcp6" {
		t.Fatalf("expected default dhcp6 service, got %#v", kea.services)
	}
}

type recordingSMDBackend struct {
	lastMethod  string
	lastMembers []string
}

func (b *recordingSMDBackend) Configured() bool                   { return true }
func (b *recordingSMDBackend) BaseURL() string                    { return "http://smd" }
func (b *recordingSMDBackend) Ready(context.Context) (any, error) { return map[string]any{}, nil }
func (b *recordingSMDBackend) ListComponents(context.Context) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) GetComponent(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) ListGroups(context.Context) (any, error) { return map[string]any{}, nil }
func (b *recordingSMDBackend) GetGroup(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) GetMembership(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) CreateGroup(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) PatchGroup(context.Context, string, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) DeleteGroup(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) GetGroupMembers(context.Context, string) (any, error) {
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) SetGroupMembers(_ context.Context, _ string, members []string) (any, error) {
	b.lastMethod = "set"
	b.lastMembers = append([]string(nil), members...)
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) AddGroupMember(_ context.Context, _ string, member string) (any, error) {
	b.lastMethod = "add"
	b.lastMembers = []string{member}
	return map[string]any{}, nil
}
func (b *recordingSMDBackend) DeleteGroupMember(_ context.Context, _ string, member string) (any, error) {
	b.lastMethod = "delete"
	b.lastMembers = []string{member}
	return map[string]any{}, nil
}

type recordingKeaBackend struct {
	services []string
}

func (b *recordingKeaBackend) Configured() bool          { return true }
func (b *recordingKeaBackend) BaseURL() string           { return "http://kea" }
func (b *recordingKeaBackend) DefaultServices() []string { return []string{"dhcp6"} }
func (b *recordingKeaBackend) RunCommand(_ context.Context, _ string, _ map[string]any, services []string) (any, error) {
	b.services = append([]string(nil), services...)
	return map[string]any{}, nil
}

type fakeBSS struct{}
type fakeCloud struct{}
type fakePower struct{}
type fakeKea struct{}
type fakeCore struct{}

func (fakeBSS) Configured() bool                                       { return true }
func (fakeBSS) BaseURL() string                                        { return "http://bss" }
func (fakeBSS) ServiceStatus(context.Context) (any, error)             { return map[string]any{}, nil }
func (fakeBSS) GetBootParameters(context.Context) (any, error)         { return map[string]any{}, nil }
func (fakeBSS) GetBootScript(context.Context, url.Values) (any, error) { return map[string]any{}, nil }
func (fakeBSS) ListHosts(context.Context) (any, error)                 { return map[string]any{}, nil }
func (fakeBSS) PutBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSS) PostBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSS) PatchBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSS) DeleteBootParameters(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeBSS) PostHosts(context.Context, map[string]any) (any, error) { return map[string]any{}, nil }

func (fakeCloud) Configured() bool                                { return true }
func (fakeCloud) BaseURL() string                                 { return "http://cloud-init" }
func (fakeCloud) Version(context.Context) (any, error)            { return map[string]any{}, nil }
func (fakeCloud) GetClusterDefaults(context.Context) (any, error) { return map[string]any{}, nil }
func (fakeCloud) SetClusterDefaults(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloud) ListGroups(context.Context) (any, error)       { return map[string]any{}, nil }
func (fakeCloud) GetGroup(context.Context, string) (any, error) { return map[string]any{}, nil }
func (fakeCloud) CreateGroup(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloud) UpdateGroup(context.Context, string, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloud) DeleteGroup(context.Context, string) (any, error) { return map[string]any{}, nil }
func (fakeCloud) UpdateInstanceInfo(context.Context, string, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakeCloud) GetMetaData(context.Context, string) (any, error)   { return map[string]any{}, nil }
func (fakeCloud) GetUserData(context.Context, string) (any, error)   { return map[string]any{}, nil }
func (fakeCloud) GetVendorData(context.Context, string) (any, error) { return map[string]any{}, nil }
func (fakeCloud) GetGroupUserData(context.Context, string, string) (any, error) {
	return map[string]any{}, nil
}

func (fakePower) Configured() bool                    { return true }
func (fakePower) BaseURL() string                     { return "http://power-control" }
func (fakePower) Health(context.Context) (any, error) { return map[string]any{}, nil }
func (fakePower) GetPowerStatus(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}
func (fakePower) CreateTransition(context.Context, map[string]any) (any, error) {
	return map[string]any{}, nil
}

func (fakeKea) Configured() bool          { return true }
func (fakeKea) BaseURL() string           { return "http://kea" }
func (fakeKea) DefaultServices() []string { return []string{"dhcp4"} }
func (fakeKea) RunCommand(context.Context, string, map[string]any, []string) (any, error) {
	return map[string]any{}, nil
}

func (fakeCore) Configured() bool                    { return false }
func (fakeCore) BaseURL() string                     { return "" }
func (fakeCore) Status(context.Context) (any, error) { return map[string]any{}, nil }

var (
	_ backends.SMDBackend          = &recordingSMDBackend{}
	_ backends.KeaBackend          = &recordingKeaBackend{}
	_ backends.BSSBackend          = fakeBSS{}
	_ backends.CloudInitBackend    = fakeCloud{}
	_ backends.PowerControlBackend = fakePower{}
	_ backends.KeaBackend          = fakeKea{}
	_ backends.CoreDHCPBackend     = fakeCore{}
)
