# openchami-mcp

`openchami-mcp` is the Go MCP server for OpenCHAMI.

It exposes MCP over stdio, talks directly to per-service OpenCHAMI and Kea
APIs, and keeps tool names stable so the legacy backends can be replaced later
without breaking MCP clients.

## Current scope

- SMD component, group, membership, and group-member tools
- BSS status, boot parameter, bootscript, and host tools
- cloud-init cluster-default, group, instance-info, and metadata tools
- power-control power-status and transition tools
- Kea Control Agent command, config, and lease tools
- explicit CoreDHCP integration seam, with no fake admin API assumed

## Repository files

- `AGENTS.md`: repo-local agent instructions for coding agents
- `CLAUDE.md`: symlink to `AGENTS.md`
- `GEMINI.md`: symlink to `AGENTS.md`
- `.env.example`: local development configuration template

## Configuration

The server is configured entirely through environment variables.

Start from [.env.example](./.env.example) and set the service URLs for the
deployment you want to control.

Important settings:

- `OPENCHAMI_MCP_MODE=read-only|read-write`
- `OPENCHAMI_MCP_ENABLE_WRITES=true|false`
- `OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true|false`
- `OPENCHAMI_MCP_FRAMING=auto|jsonl|content-length`
- `OPENCHAMI_MCP_KEA_DEFAULT_SERVICES=dhcp4,dhcp6`

Per-service settings follow the same pattern:

- `OPENCHAMI_MCP_<SERVICE>_URL`
- `OPENCHAMI_MCP_<SERVICE>_TIMEOUT`
- `OPENCHAMI_MCP_<SERVICE>_RETRIES`
- `OPENCHAMI_MCP_<SERVICE>_BEARER_TOKEN`

Supported service prefixes:

- `SMD`
- `BSS`
- `CLOUD_INIT`
- `POWER_CONTROL`
- `KEA`
- `COREDHCP`

## Running

```bash
go run ./cmd/openchami-mcp
```

Optional flags override the environment:

```bash
go run ./cmd/openchami-mcp -mode read-write -framing jsonl -log-level debug
```

## Notes on CoreDHCP

The checked-out upstream CoreDHCP repo does not expose a standard management
API comparable to Kea Control Agent.

The current implementation therefore keeps a `CoreDHCPBackend` seam and reports
that status through `openchami_health`, but it does not pretend that an admin
API exists where one has not been verified yet.
