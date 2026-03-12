package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/OpenCHAMI/openchami-mcp/internal/clients/httpjson"
	"github.com/OpenCHAMI/openchami-mcp/internal/config"
	"github.com/OpenCHAMI/openchami-mcp/internal/errorsx"
	"github.com/OpenCHAMI/openchami-mcp/internal/tools"
)

const (
	protocolVersion = "2024-11-05"
	serverVersion   = "0.1.0"
)

type Server struct {
	cfg      config.Config
	registry *tools.Registry
	logger   *slog.Logger
	reader   io.Reader
	writer   io.Writer
}

type framingState struct {
	lastInput string
}

func NewServer(cfg config.Config, registry *tools.Registry, logger *slog.Logger, reader io.Reader, writer io.Writer) *Server {
	return &Server{
		cfg:      cfg,
		registry: registry,
		logger:   logger,
		reader:   reader,
		writer:   writer,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	reader := bufio.NewReader(s.reader)
	state := &framingState{}

	for {
		message, err := readMessage(reader, state)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		response, exit, err := s.handle(ctx, message)
		if err != nil {
			return err
		}
		if response != nil {
			if err := writeMessage(s.writer, state, s.cfg.Framing, response); err != nil {
				return err
			}
		}
		if exit {
			return nil
		}
	}
}

func (s *Server) handle(ctx context.Context, message map[string]any) (map[string]any, bool, error) {
	requestID, _ := message["id"]

	method, ok := message["method"].(string)
	if !ok || method == "" {
		if requestID == nil {
			return nil, false, nil
		}
		return errorResponse(requestID, -32600, "Invalid Request", nil), false, nil
	}

	params, _ := message["params"].(map[string]any)

	switch method {
	case "initialize":
		return resultResponse(requestID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo": map[string]any{
				"name":    "openchami-mcp",
				"version": serverVersion,
			},
		}), false, nil
	case "notifications/initialized":
		return nil, false, nil
	case "ping":
		return resultResponse(requestID, map[string]any{}), false, nil
	case "tools/list":
		return resultResponse(requestID, map[string]any{"tools": definitionsAsMCP(s.registry.Definitions())}), false, nil
	case "tools/call":
		if requestID == nil {
			return nil, false, nil
		}
		if params == nil {
			return errorResponse(requestID, -32602, "Invalid params for tools/call", nil), false, nil
		}
		name, ok := params["name"].(string)
		if !ok || name == "" {
			return errorResponse(requestID, -32602, "Missing tool name", nil), false, nil
		}
		arguments, _ := params["arguments"].(map[string]any)
		result, err := s.callTool(ctx, name, arguments)
		if err != nil {
			return errorFromTool(requestID, err), false, nil
		}
		return resultResponse(requestID, map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": encodeToolResult(result),
				},
			},
		}), false, nil
	case "shutdown":
		return resultResponse(requestID, map[string]any{}), false, nil
	case "exit":
		return nil, true, nil
	default:
		if requestID == nil {
			return nil, false, nil
		}
		return errorResponse(requestID, -32601, fmt.Sprintf("Method not found: %s", method), nil), false, nil
	}
}

func (s *Server) callTool(ctx context.Context, name string, arguments map[string]any) (any, error) {
	definition, ok := s.registry.Definition(name)
	if !ok {
		return nil, errorsx.NewInvalidArgument("Unknown tool: %s", name)
	}
	if !definition.ReadOnly {
		if s.cfg.Mode != config.ModeReadWrite {
			return nil, errorsx.NewPermission("Tool '%s' is write-capable and requires %s mode.", name, config.ModeReadWrite)
		}
		if s.cfg.RequireWriteAck && !s.cfg.EnableWrites {
			return nil, errorsx.NewPermission("Write mode requires OPENCHAMI_MCP_ENABLE_WRITES=true to be set.")
		}
	}
	return s.registry.Call(ctx, name, arguments)
}

func definitionsAsMCP(definitions []tools.Definition) []map[string]any {
	out := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		out = append(out, map[string]any{
			"name":        definition.Name,
			"description": definition.Description,
			"inputSchema": definition.InputSchema,
			"annotations": map[string]any{"readOnlyHint": definition.ReadOnly},
		})
	}
	return out
}

func encodeToolResult(result any) string {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fallback, _ := json.Marshal(map[string]any{"error": err.Error()})
		return string(fallback)
	}
	return string(raw)
}

func resultResponse(id any, result any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

func errorResponse(id any, code int, message string, data any) map[string]any {
	errObj := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		errObj["data"] = data
	}
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   errObj,
	}
}

func errorFromTool(id any, err error) map[string]any {
	var permissionErr *errorsx.PermissionError
	if errors.As(err, &permissionErr) {
		return errorResponse(id, -32001, permissionErr.Error(), nil)
	}

	var apiErr *httpjson.APIError
	if errors.As(err, &apiErr) {
		return errorResponse(id, -32002, "OpenCHAMI API request failed", map[string]any{
			"message":       apiErr.Error(),
			"status_code":   apiErr.StatusCode,
			"response_body": apiErr.ResponseBody,
		})
	}

	var invalidErr *errorsx.InvalidArgumentError
	if errors.As(err, &invalidErr) {
		return errorResponse(id, -32602, invalidErr.Error(), nil)
	}

	var notImplementedErr *errorsx.NotImplementedError
	if errors.As(err, &notImplementedErr) {
		return errorResponse(id, -32003, notImplementedErr.Error(), nil)
	}

	return errorResponse(id, -32603, fmt.Sprintf("Internal error: %v", err), nil)
}
