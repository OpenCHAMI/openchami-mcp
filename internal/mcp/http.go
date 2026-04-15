package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenCHAMI/openchami-mcp/internal/config"
)

const maxHTTPBodyBytes = 1 << 20

func (s *Server) ListenAndServeHTTP(ctx context.Context) error {
	if err := validateHTTPServerConfig(s.cfg); err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:    s.cfg.HTTP.ListenAddr,
		Handler: s.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(s.cfg.HTTP.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc(s.cfg.HTTP.BasePath, s.handleHTTPMCP)
	return s.logRequests(mux)
}

func validateHTTPServerConfig(cfg config.Config) error {
	if cfg.Transport != config.TransportHTTP {
		return nil
	}
	if cfg.Mode == config.ModeReadWrite && !isLoopbackBind(cfg.HTTP.ListenAddr) && strings.TrimSpace(cfg.HTTP.BearerToken) == "" {
		return fmt.Errorf("read-write HTTP mode requires localhost binding or OPENCHAMI_MCP_HTTP_BEARER_TOKEN")
	}
	return nil
}

func (s *Server) handleHTTPMCP(w http.ResponseWriter, r *http.Request) {
	if !s.originAllowed(r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	if !s.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="openchami-mcp"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		http.Error(w, "streaming GET is not enabled", http.StatusMethodNotAllowed)
	case http.MethodPost:
		s.handleHTTPPost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHTTPPost(w http.ResponseWriter, r *http.Request) {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxHTTPBodyBytes))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	message, httpStatus, response := decodeHTTPMessage(body)
	if httpStatus != 0 {
		writeHTTPJSON(w, httpStatus, response)
		return
	}

	result, err := s.HandleMessage(r.Context(), message)
	if err != nil {
		writeHTTPJSON(w, http.StatusInternalServerError, errorResponse(message["id"], -32603, fmt.Sprintf("Internal error: %v", err), nil))
		return
	}

	if len(result.Messages) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if len(result.Messages) == 1 {
		writeHTTPJSON(w, http.StatusOK, result.Messages[0])
		return
	}

	writeHTTPJSON(w, http.StatusOK, result.Messages)
}

func decodeHTTPMessage(body []byte) (map[string]any, int, any) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, http.StatusBadRequest, errorResponse(nil, -32700, "Parse error", nil)
	}
	if strings.HasPrefix(trimmed, "[") {
		return nil, http.StatusBadRequest, errorResponse(nil, -32600, "JSON-RPC batch requests are not supported", nil)
	}

	var message map[string]any
	if err := json.Unmarshal(body, &message); err != nil {
		return nil, http.StatusBadRequest, errorResponse(nil, -32700, "Parse error", nil)
	}

	if message == nil {
		return nil, http.StatusBadRequest, errorResponse(nil, -32600, "Invalid Request", nil)
	}

	if isJSONRPCResponse(message) {
		return message, http.StatusAccepted, nil
	}

	if method, hasMethod := message["method"]; !hasMethod && message["id"] == nil && method == nil {
		return nil, http.StatusBadRequest, errorResponse(nil, -32600, "Invalid Request", nil)
	}

	return message, 0, nil
}

func writeHTTPJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) authorized(r *http.Request) bool {
	if strings.TrimSpace(s.cfg.HTTP.BearerToken) == "" {
		return true
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.HTTP.BearerToken)) == 1
}

func (s *Server) originAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return isLoopbackRemoteAddr(r.RemoteAddr)
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	if len(s.cfg.HTTP.AllowedOrigins) > 0 {
		for _, allowed := range s.cfg.HTTP.AllowedOrigins {
			if sameOrigin(origin, allowed) {
				return true
			}
		}
		return false
	}

	return isLocalOrigin(parsed)
}

func sameOrigin(origin, allowed string) bool {
	left, err := url.Parse(origin)
	if err != nil {
		return false
	}
	right, err := url.Parse(allowed)
	if err != nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func isLocalOrigin(origin *url.URL) bool {
	host := origin.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLoopbackBind(listenAddr string) bool {
	host, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		host = listenAddr
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		s.logger.Info("http request", "remote", remoteHost(r.RemoteAddr), "method", r.Method, "path", r.URL.Path, "status", recorder.status())
	})
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	return r.ResponseWriter.Write(p)
}

func (r *statusRecorder) status() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}
