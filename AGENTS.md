# openchami-mcp agent notes

## Purpose

`openchami-mcp` is the Go MCP server for OpenCHAMI.

Use this repository for MCP integration work that exposes stable tool names
over stdio while talking to OpenCHAMI and Kea backends directly.

## Scope

- implement and maintain the Go MCP transport and JSON-RPC handling
- keep tool contracts stable even when backend implementations change
- integrate with:
  - SMD
  - BSS
  - cloud-init
  - power-control
  - Kea
- preserve an explicit seam for CoreDHCP until a real management API exists

## Repository structure

- `cmd/openchami-mcp`: binary entrypoint
- `internal/config`: env-driven configuration
- `internal/clients`: shared HTTP client helpers
- `internal/backends`: backend interfaces and adapters
- `internal/mcp`: MCP framing and request handling
- `internal/tools`: tool registry and argument validation
- `internal/testing`: shared test helpers

## Working rules

- keep tool names backend-agnostic so Fabrica adapters can be added later
- prefer backend interfaces over wiring transport details into tool handlers
- do not invent a CoreDHCP admin API that has not been verified
- keep write operations guarded by MCP mode plus explicit write acknowledgment
- add tests for protocol behavior and backend request construction when adding tools

## Local workflow

- use `.env.example` as the baseline config template
- run `go test ./...` before finishing changes
- keep the service runnable with `go run ./cmd/openchami-mcp`

## Alias files

`CLAUDE.md` and `GEMINI.md` should remain symlinks to this file.
