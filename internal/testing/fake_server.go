package testingutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

type RecordedRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Headers http.Header
	Body    []byte
}

type FakeServer struct {
	t        testing.TB
	server   *httptest.Server
	mu       sync.Mutex
	requests []RecordedRequest
}

func NewFakeServer(t testing.TB, responder func(http.ResponseWriter, *http.Request)) *FakeServer {
	t.Helper()

	fs := &FakeServer{t: t}
	fs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		_ = r.Body.Close()

		fs.mu.Lock()
		fs.requests = append(fs.requests, RecordedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.Query(),
			Headers: r.Header.Clone(),
			Body:    body,
		})
		fs.mu.Unlock()

		responder(w, r)
	}))

	t.Cleanup(fs.server.Close)
	return fs
}

func (f *FakeServer) URL() string {
	return f.server.URL
}

func (f *FakeServer) Requests() []RecordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]RecordedRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

func (f *FakeServer) LastRequest() RecordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		f.t.Fatalf("expected at least one request")
	}
	return f.requests[len(f.requests)-1]
}
