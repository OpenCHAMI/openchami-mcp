# openchami-mcp

`openchami-mcp` is the Go MCP server for OpenCHAMI.

It exposes MCP over stdio, talks directly to per-service OpenCHAMI and Kea
APIs, and keeps tool names stable so the legacy backends can be replaced later
without breaking MCP clients.

## What OpenCHAMI is

OpenCHAMI is a microservice-based cluster provisioning and management stack for
HPC-style environments.

The services behind this MCP server cover hardware inventory, boot
configuration, node metadata delivery, power control, and DHCP or lease-related
integration points. `openchami-mcp` gives agent clients one stable tool surface
over those services instead of making each client speak every backend API
directly.

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
- `Dockerfile`: container packaging for the stdio MCP server
- `flake.nix`: Nix package and OCI image build definitions
- `.agents/skills/read-only-infrastructure-assistant/`: repo-local read-only
  infrastructure skill for OpenCode-compatible agents

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

Build a local binary for MCP clients:

```bash
mkdir -p bin
go build -o ./bin/openchami-mcp ./cmd/openchami-mcp
```

The client configuration examples below assume the binary is available at
`/absolute/path/to/openchami-mcp` and that you want the safer read-only mode.

## Client setup

For all examples below, keep the server in read-only mode unless you are
explicitly testing write-capable tools:

```text
OPENCHAMI_MCP_MODE=read-only
OPENCHAMI_MCP_ENABLE_WRITES=false
OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true
OPENCHAMI_MCP_SMD_URL=http://localhost:27779
OPENCHAMI_MCP_BSS_URL=http://localhost:27778
OPENCHAMI_MCP_CLOUD_INIT_URL=http://localhost:27777
OPENCHAMI_MCP_POWER_CONTROL_URL=http://localhost:28007/v1
OPENCHAMI_MCP_KEA_URL=http://localhost:8000
```

### Codex

Add the server with the CLI:

```bash
codex mcp add openchami-mcp \
  --env OPENCHAMI_MCP_MODE=read-only \
  --env OPENCHAMI_MCP_ENABLE_WRITES=false \
  --env OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true \
  --env OPENCHAMI_MCP_SMD_URL=http://localhost:27779 \
  --env OPENCHAMI_MCP_BSS_URL=http://localhost:27778 \
  --env OPENCHAMI_MCP_CLOUD_INIT_URL=http://localhost:27777 \
  --env OPENCHAMI_MCP_POWER_CONTROL_URL=http://localhost:28007/v1 \
  --env OPENCHAMI_MCP_KEA_URL=http://localhost:8000 \
  -- /absolute/path/to/openchami-mcp
```

Equivalent `~/.codex/config.toml` snippet:

```toml
[mcp_servers.openchami-mcp]
command = "/absolute/path/to/openchami-mcp"

[mcp_servers.openchami-mcp.env]
OPENCHAMI_MCP_MODE = "read-only"
OPENCHAMI_MCP_ENABLE_WRITES = "false"
OPENCHAMI_MCP_REQUIRE_WRITE_ACK = "true"
OPENCHAMI_MCP_SMD_URL = "http://localhost:27779"
OPENCHAMI_MCP_BSS_URL = "http://localhost:27778"
OPENCHAMI_MCP_CLOUD_INIT_URL = "http://localhost:27777"
OPENCHAMI_MCP_POWER_CONTROL_URL = "http://localhost:28007/v1"
OPENCHAMI_MCP_KEA_URL = "http://localhost:8000"
```

### Claude Code

Add the server with the CLI:

```bash
claude mcp add --scope project --transport stdio \
  openchami-mcp -- /absolute/path/to/openchami-mcp
```

Per the current Claude Code MCP docs, all options must come before the server
name, and `--` separates the server name from the local command to run.

Then add the read-only environment block in the project-scoped `.mcp.json` at
the repo root:

```json
{
  "mcpServers": {
    "openchami-mcp": {
      "type": "stdio",
      "command": "/absolute/path/to/openchami-mcp",
      "args": [],
      "env": {
        "OPENCHAMI_MCP_MODE": "read-only",
        "OPENCHAMI_MCP_ENABLE_WRITES": "false",
        "OPENCHAMI_MCP_REQUIRE_WRITE_ACK": "true",
        "OPENCHAMI_MCP_SMD_URL": "http://localhost:27779",
        "OPENCHAMI_MCP_BSS_URL": "http://localhost:27778",
        "OPENCHAMI_MCP_CLOUD_INIT_URL": "http://localhost:27777",
        "OPENCHAMI_MCP_POWER_CONTROL_URL": "http://localhost:28007/v1",
        "OPENCHAMI_MCP_KEA_URL": "http://localhost:8000"
      }
    }
  }
}
```

### Gemini CLI

Add the server with the CLI:

```bash
gemini mcp add --scope project \
  -e OPENCHAMI_MCP_MODE=read-only \
  -e OPENCHAMI_MCP_ENABLE_WRITES=false \
  -e OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true \
  -e OPENCHAMI_MCP_SMD_URL=http://localhost:27779 \
  -e OPENCHAMI_MCP_BSS_URL=http://localhost:27778 \
  -e OPENCHAMI_MCP_CLOUD_INIT_URL=http://localhost:27777 \
  -e OPENCHAMI_MCP_POWER_CONTROL_URL=http://localhost:28007/v1 \
  -e OPENCHAMI_MCP_KEA_URL=http://localhost:8000 \
  openchami-mcp /absolute/path/to/openchami-mcp
```

Equivalent project-scoped `.gemini/settings.json`:

```json
{
  "mcpServers": {
    "openchami-mcp": {
      "command": "/absolute/path/to/openchami-mcp",
      "args": [],
      "env": {
        "OPENCHAMI_MCP_MODE": "read-only",
        "OPENCHAMI_MCP_ENABLE_WRITES": "false",
        "OPENCHAMI_MCP_REQUIRE_WRITE_ACK": "true",
        "OPENCHAMI_MCP_SMD_URL": "http://localhost:27779",
        "OPENCHAMI_MCP_BSS_URL": "http://localhost:27778",
        "OPENCHAMI_MCP_CLOUD_INIT_URL": "http://localhost:27777",
        "OPENCHAMI_MCP_POWER_CONTROL_URL": "http://localhost:28007/v1",
        "OPENCHAMI_MCP_KEA_URL": "http://localhost:8000"
      }
    }
  }
}
```

### OpenCode

OpenCode can add MCP servers interactively with:

```bash
opencode mcp add
```

For a repo-checked-in example, add this to project `opencode.json`. Current
OpenCode docs also support the same structure in global
`~/.config/opencode/opencode.json`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "openchami-mcp": {
      "type": "local",
      "command": ["/absolute/path/to/openchami-mcp"],
      "enabled": true,
      "environment": {
        "OPENCHAMI_MCP_MODE": "read-only",
        "OPENCHAMI_MCP_ENABLE_WRITES": "false",
        "OPENCHAMI_MCP_REQUIRE_WRITE_ACK": "true",
        "OPENCHAMI_MCP_SMD_URL": "http://localhost:27779",
        "OPENCHAMI_MCP_BSS_URL": "http://localhost:27778",
        "OPENCHAMI_MCP_CLOUD_INIT_URL": "http://localhost:27777",
        "OPENCHAMI_MCP_POWER_CONTROL_URL": "http://localhost:28007/v1",
        "OPENCHAMI_MCP_KEA_URL": "http://localhost:8000"
      }
    }
  }
}
```

## Repo-local skill

This repository includes a project-local OpenCode skill at
`.agents/skills/read-only-infrastructure-assistant/SKILL.md`.

Use it when you want an agent to:

- explain what OpenCHAMI is and how the major services fit together
- inspect inventory, boot, metadata, power, and DHCP state without making
  changes
- stay in read-only MCP mode and avoid write-capable operations by default

## Container builds

Build the runtime image with Docker:

```bash
docker build -t openchami-mcp:dev .
```

Build the binary package with Nix:

```bash
nix build .#default
```

If the flake files are still untracked in a dirty checkout, use
`nix build path:$PWD#default` instead.

Build an OCI archive with Nix:

```bash
nix build .#ociImage
```

If the flake files are still untracked in a dirty checkout, use
`nix build path:$PWD#ociImage` instead.

## Notes on CoreDHCP

The checked-out upstream CoreDHCP repo does not expose a standard management
API comparable to Kea Control Agent.

The current implementation therefore keeps a `CoreDHCPBackend` seam and reports
that status through `openchami_health`, but it does not pretend that an admin
API exists where one has not been verified yet.
