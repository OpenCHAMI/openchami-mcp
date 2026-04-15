package httpjson

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

	client, err := NewClient(Config{
		BaseURL: server.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

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

	client, err := NewClient(Config{
		BaseURL: server.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.Do(context.Background(), http.MethodGet, "/", RequestOptions{})
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

func TestClientSupportsHTTPSInsecureSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"secure":true}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:     server.URL,
		Timeout:     time.Second,
		TLSInsecure: true,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	resp, err := client.Do(context.Background(), http.MethodGet, "/", RequestOptions{})
	if err != nil {
		t.Fatalf("https request failed: %v", err)
	}
	body, ok := resp.Data.(map[string]any)
	if !ok || body["secure"] != true {
		t.Fatalf("unexpected https response body: %#v", resp.Data)
	}
}

func TestClientSupportsHTTPSCustomCA(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	caFile := t.TempDir() + "/server-ca.pem"
	if err := os.WriteFile(caFile, certPEM, 0o644); err != nil {
		t.Fatalf("failed to write CA file: %v", err)
	}

	client, err := NewClient(Config{
		BaseURL:   server.URL,
		Timeout:   time.Second,
		TLSCAFile: caFile,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	resp, err := client.Do(context.Background(), http.MethodGet, "/", RequestOptions{})
	if err != nil {
		t.Fatalf("https request with custom CA failed: %v", err)
	}
	body, ok := resp.Data.(map[string]any)
	if !ok || body["text"] != "ok" {
		t.Fatalf("unexpected https custom CA response: %#v", resp.Data)
	}
}

func TestClientRejectsMissingCAFile(t *testing.T) {
	_, err := NewClient(Config{
		BaseURL:   "https://example.invalid",
		Timeout:   time.Second,
		TLSCAFile: "/does/not/exist",
	})
	if err == nil {
		t.Fatal("expected CA file error")
	}
}

func TestClientFailsHTTPSWithoutTrust(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL: server.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.Do(context.Background(), http.MethodGet, "/", RequestOptions{})
	if err == nil {
		t.Fatal("expected TLS verification error")
	}
	var unknownAuthorityErr x509.UnknownAuthorityError
	if !errors.As(err, &unknownAuthorityErr) {
		t.Fatalf("expected unknown authority error, got %T: %v", err, err)
	}
}
