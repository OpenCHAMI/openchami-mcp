package httpjson

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientDecodesJSONAndText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"value":42}`))
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("plain-text"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Timeout: time.Second,
	})

	jsonResp, err := client.Do(context.Background(), http.MethodGet, "/json", RequestOptions{})
	if err != nil {
		t.Fatalf("json request failed: %v", err)
	}
	jsonBody, ok := jsonResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map body, got %T", jsonResp.Data)
	}
	if got := jsonBody["ok"]; got != true {
		t.Fatalf("expected ok=true, got %#v", got)
	}

	textResp, err := client.Do(context.Background(), http.MethodGet, "/text", RequestOptions{})
	if err != nil {
		t.Fatalf("text request failed: %v", err)
	}
	textBody, ok := textResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map body for text response, got %T", textResp.Data)
	}
	if got := textBody["text"]; got != "plain-text" {
		t.Fatalf("expected plain text payload, got %#v", got)
	}
}

func TestClientReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Timeout: time.Second,
	})

	_, err := client.Do(context.Background(), http.MethodGet, "/", RequestOptions{})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", apiErr.StatusCode)
	}
}
