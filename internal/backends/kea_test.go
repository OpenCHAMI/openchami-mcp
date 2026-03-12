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

func TestKeaRunCommandBuildsExpectedPayload(t *testing.T) {
	server := testingutil.NewFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":0}`))
	})

	backend := NewKeaBackend(httpjson.NewClient(httpjson.Config{
		BaseURL: server.URL(),
		Timeout: time.Second,
	}), []string{"dhcp4"})

	_, err := backend.RunCommand(context.Background(), "config-set", map[string]any{
		"Dhcp4": map[string]any{"interfaces-config": map[string]any{}},
	}, []string{"dhcp4"})
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	request := server.LastRequest()
	if request.Method != http.MethodPost || request.Path != "/" {
		t.Fatalf("unexpected Kea request: %#v", request)
	}

	var payload map[string]any
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		t.Fatalf("failed to decode Kea payload: %v", err)
	}
	if got := payload["command"]; got != "config-set" {
		t.Fatalf("expected command config-set, got %#v", got)
	}
	if _, ok := payload["service"]; !ok {
		t.Fatal("expected service selector in Kea payload")
	}
}
