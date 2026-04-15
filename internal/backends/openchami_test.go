package backends

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/OpenCHAMI/openchami-mcp/internal/clients/httpjson"
	testingutil "github.com/OpenCHAMI/openchami-mcp/internal/testing"
)

func TestSMDGroupMemberRequests(t *testing.T) {
	server := testingutil.NewFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	client, err := httpjson.NewClient(httpjson.Config{
		BaseURL: server.URL(),
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	backend := NewSMDBackend(client)

	ctx := context.Background()

	if _, err := backend.SetGroupMembers(ctx, "compute", []string{"x0c0s0b0n0"}); err != nil {
		t.Fatalf("SetGroupMembers failed: %v", err)
	}
	request := server.LastRequest()
	if request.Method != http.MethodPut || request.Path != "/hsm/v2/groups/compute/members" {
		t.Fatalf("unexpected set-group-members request: %#v", request)
	}

	var payload map[string]any
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	if got := payload["group"]; got != "compute" {
		t.Fatalf("expected group label in request body, got %#v", got)
	}

	if _, err := backend.AddGroupMember(ctx, "compute", "x0c0s0b0n1"); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}
	request = server.LastRequest()
	if request.Method != http.MethodPost || request.Path != "/hsm/v2/groups/compute/members" {
		t.Fatalf("unexpected add-group-member request: %#v", request)
	}

	if _, err := backend.DeleteGroupMember(ctx, "compute", "x0c0s0b0n1"); err != nil {
		t.Fatalf("DeleteGroupMember failed: %v", err)
	}
	request = server.LastRequest()
	if request.Method != http.MethodDelete || request.Path != "/hsm/v2/groups/compute/members/x0c0s0b0n1" {
		t.Fatalf("unexpected delete-group-member request: %#v", request)
	}
}

func TestCloudInitImpersonationPaths(t *testing.T) {
	server := testingutil.NewFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	client, err := httpjson.NewClient(httpjson.Config{
		BaseURL: server.URL(),
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	backend := NewCloudInitBackend(client)

	ctx := context.Background()

	if _, err := backend.GetMetaData(ctx, ""); err != nil {
		t.Fatalf("GetMetaData failed: %v", err)
	}
	request := server.LastRequest()
	if request.Path != "/meta-data" {
		t.Fatalf("unexpected meta-data path: %s", request.Path)
	}

	if _, err := backend.GetUserData(ctx, "x1000c0s0b0n0"); err != nil {
		t.Fatalf("GetUserData failed: %v", err)
	}
	request = server.LastRequest()
	if request.Path != "/admin/impersonation/x1000c0s0b0n0/user-data" {
		t.Fatalf("unexpected impersonation user-data path: %s", request.Path)
	}

	if _, err := backend.GetGroupUserData(ctx, "x1000c0s0b0n0", "compute"); err != nil {
		t.Fatalf("GetGroupUserData failed: %v", err)
	}
	request = server.LastRequest()
	if request.Path != "/admin/impersonation/x1000c0s0b0n0/compute.yaml" {
		t.Fatalf("unexpected impersonation group yaml path: %s", request.Path)
	}
}
