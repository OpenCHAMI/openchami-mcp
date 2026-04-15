# openchami-mcp

`openchami-mcp` is the Go MCP server for OpenCHAMI.

It exposes MCP over stdio and Streamable HTTP, talks directly to per-service
OpenCHAMI and Kea APIs, and keeps tool names stable so the legacy backends can
be replaced later without breaking MCP clients.

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
- `Dockerfile`: container packaging for the MCP server
- `flake.nix`: Nix package and OCI image build definitions
- `.agents/skills/read-only-infrastructure-assistant/`: repo-local read-only
  infrastructure skill for OpenCode-compatible agents

## Configuration

The server is configured entirely through environment variables.

Start from [.env.example](./.env.example) and set the service URLs for the
deployment you want to control.

Important settings:

- `OPENCHAMI_MCP_MODE=read-only|read-write`
- `OPENCHAMI_MCP_TRANSPORT=stdio|http`
- `OPENCHAMI_MCP_ENABLE_WRITES=true|false`
- `OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true|false`
- `OPENCHAMI_MCP_FRAMING=auto|jsonl|content-length`
- `OPENCHAMI_MCP_LISTEN_ADDR=127.0.0.1:8081`
- `OPENCHAMI_MCP_BASE_PATH=/mcp`
- `OPENCHAMI_MCP_HEALTH_PATH=/healthz`
- `OPENCHAMI_MCP_ALLOWED_ORIGINS=http://localhost:3000,...`
- `OPENCHAMI_MCP_HTTP_BEARER_TOKEN=...`
- `OPENCHAMI_MCP_KEA_DEFAULT_SERVICES=dhcp4,dhcp6`

Transport notes:

- `stdio` remains the default and is the right transport for local MCP client
  integrations such as Codex, Claude Code, Gemini CLI, and OpenCode.
- `http` starts a standalone Streamable HTTP service on one MCP endpoint.
- HTTP mode binds to `127.0.0.1:8081` by default.
- HTTP mode rejects non-local requests that do not present an allowed `Origin`.
- Remote `read-write` HTTP mode is refused unless you configure
  `OPENCHAMI_MCP_HTTP_BEARER_TOKEN`.

Per-service settings follow the same pattern:

- `OPENCHAMI_MCP_<SERVICE>_URL`
- `OPENCHAMI_MCP_<SERVICE>_TIMEOUT`
- `OPENCHAMI_MCP_<SERVICE>_RETRIES`
- `OPENCHAMI_MCP_<SERVICE>_BEARER_TOKEN`
- `OPENCHAMI_MCP_<SERVICE>_TLS_CA_FILE`
- `OPENCHAMI_MCP_<SERVICE>_TLS_INSECURE_SKIP_VERIFY`

TLS notes:

- Use `OPENCHAMI_MCP_<SERVICE>_TLS_CA_FILE` to trust a private CA bundle for
  one backend without changing the container or host trust store.
- Use `OPENCHAMI_MCP_<SERVICE>_TLS_INSECURE_SKIP_VERIFY=true` only when that
  backend is deliberately using a self-signed certificate and you do not have a
  CA bundle available yet.
- In the standard `ochami-from-scratch` quadlet deployment, SMD listens on
  `https://localhost:27779` with a self-signed certificate, so the default SMD
  example below uses HTTPS plus
  `OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY=true`.

Supported service prefixes:

- `SMD`
- `BSS`
- `CLOUD_INIT`
- `POWER_CONTROL`
- `KEA`
- `COREDHCP`

## Running

Start the default stdio server:

```bash
go run ./cmd/openchami-mcp
```

Start HTTP mode on the default localhost bind:

```bash
OPENCHAMI_MCP_TRANSPORT=http \
OPENCHAMI_MCP_ALLOWED_ORIGINS=http://localhost:3000 \
go run ./cmd/openchami-mcp
```

Optional flags override the environment:

```bash
go run ./cmd/openchami-mcp \
  -transport http \
  -listen-addr 127.0.0.1:8081 \
  -base-path /mcp \
  -health-path /healthz \
  -allowed-origins http://localhost:3000 \
  -log-level debug
```

Smoke-test the HTTP transport:

```bash
curl -sS \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://localhost:3000' \
  http://127.0.0.1:8081/mcp \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'
```

Build a local binary for MCP clients:

```bash
mkdir -p bin
go build -o ./bin/openchami-mcp ./cmd/openchami-mcp
```

The client configuration examples below assume the binary is available at
`/absolute/path/to/openchami-mcp` and that you want the safer read-only mode.

Protocol notes:

- The server prefers MCP protocol version `2025-03-26`, which is the first
  version that defines Streamable HTTP.
- It also accepts legacy `2024-11-05` initialize requests so existing stdio
  clients do not break during the transport transition.

## Client setup

For all examples below, keep the server in read-only mode unless you are
explicitly testing write-capable tools:

```text
OPENCHAMI_MCP_MODE=read-only
OPENCHAMI_MCP_TRANSPORT=stdio
OPENCHAMI_MCP_ENABLE_WRITES=false
OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true
OPENCHAMI_MCP_SMD_URL=https://localhost:27779
OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY=true
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
  --env OPENCHAMI_MCP_SMD_URL=https://localhost:27779 \
  --env OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY=true \
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
OPENCHAMI_MCP_SMD_URL = "https://localhost:27779"
OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY = "true"
OPENCHAMI_MCP_BSS_URL = "http://localhost:27778"
OPENCHAMI_MCP_CLOUD_INIT_URL = "http://localhost:27777"
OPENCHAMI_MCP_POWER_CONTROL_URL = "http://localhost:28007/v1"
OPENCHAMI_MCP_KEA_URL = "http://localhost:8000"
```

If `openchami-mcp` is running remotely on a host such as `msws` and only
listens on `127.0.0.1:8081`, there are two workable access patterns.

Approach 1: separate SSH tunnel plus native HTTP MCP entry.

Start the tunnel in another terminal:

```bash
ssh -N -L 18081:127.0.0.1:8081 msws
```

Then add the HTTP MCP server:

```bash
codex mcp add openchami-msws --url http://127.0.0.1:18081/mcp
```

Equivalent `~/.codex/config.toml` snippet:

```toml
[mcp_servers.openchami-msws]
enabled = true
url = "http://127.0.0.1:18081/mcp"
```

Approach 2: one self-contained stdio wrapper that opens the SSH tunnel and
bridges to HTTP with `mcp-remote`.

CLI:

```bash
codex mcp add openchami-msws -- bash -lc 'set -euo pipefail
PORT=18081
ssh -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -N -L ${PORT}:127.0.0.1:8081 msws &
ssh_pid=$!
cleanup() { kill "$ssh_pid" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
for _ in $(seq 1 50); do
  curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null && break
  sleep 0.2
done
exec npx -y mcp-remote "http://127.0.0.1:${PORT}/mcp"'
```

Equivalent `~/.codex/config.toml` snippet:

```toml
[mcp_servers.openchami-msws]
enabled = true
command = "bash"
args = ["-lc", """
set -euo pipefail
PORT=18081
ssh -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -N -L ${PORT}:127.0.0.1:8081 msws &
ssh_pid=$!
cleanup() { kill "$ssh_pid" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
for _ in $(seq 1 50); do
  curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null && break
  sleep 0.2
done
exec npx -y mcp-remote "http://127.0.0.1:${PORT}/mcp"
"""]
```

The second approach was verified on this laptop on `2026-04-14` by adding the
Codex entry and successfully calling `bss_service_status` through the SSH
tunnel. It requires `ssh`, `curl`, and `npx` on the local machine.

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
        "OPENCHAMI_MCP_SMD_URL": "https://localhost:27779",
        "OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY": "true",
        "OPENCHAMI_MCP_BSS_URL": "http://localhost:27778",
        "OPENCHAMI_MCP_CLOUD_INIT_URL": "http://localhost:27777",
        "OPENCHAMI_MCP_POWER_CONTROL_URL": "http://localhost:28007/v1",
        "OPENCHAMI_MCP_KEA_URL": "http://localhost:8000"
      }
    }
  }
}
```

For a remote `openchami-mcp` bound to `127.0.0.1:8081` on `msws`, the same two
patterns work in Claude Code.

Approach 1: separate SSH tunnel plus native HTTP MCP entry.

Start the tunnel in another terminal:

```bash
ssh -N -L 18081:127.0.0.1:8081 msws
```

Then add the project-scoped HTTP server:

```bash
claude mcp add --scope project --transport http openchami-msws http://127.0.0.1:18081/mcp
```

Equivalent project-scoped `.mcp.json`:

```json
{
  "mcpServers": {
    "openchami-msws": {
      "type": "http",
      "url": "http://127.0.0.1:18081/mcp"
    }
  }
}
```

Approach 2: self-contained stdio wrapper.

```bash
claude mcp add --scope project openchami-msws -- bash -lc 'set -euo pipefail
PORT=18081
ssh -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -N -L ${PORT}:127.0.0.1:8081 msws &
ssh_pid=$!
cleanup() { kill "$ssh_pid" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
for _ in $(seq 1 50); do
  curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null && break
  sleep 0.2
done
exec npx -y mcp-remote "http://127.0.0.1:${PORT}/mcp"'
```

Equivalent `.mcp.json`:

```json
{
  "mcpServers": {
    "openchami-msws": {
      "type": "stdio",
      "command": "bash",
      "args": [
        "-lc",
        "set -euo pipefail\nPORT=18081\nssh -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -N -L ${PORT}:127.0.0.1:8081 msws &\nssh_pid=$!\ncleanup() { kill \"$ssh_pid\" 2>/dev/null || true; }\ntrap cleanup EXIT INT TERM\nfor _ in $(seq 1 50); do\n  curl -sf \"http://127.0.0.1:${PORT}/healthz\" >/dev/null && break\n  sleep 0.2\ndone\nexec npx -y mcp-remote \"http://127.0.0.1:${PORT}/mcp\""
      ],
      "env": {}
    }
  }
}
```

Both Claude Code variants were validated locally on `2026-04-14`. The CLI
reported `Status: ✓ Connected` for both the HTTP entry and the self-contained
stdio wrapper.

### Gemini CLI

Add the server with the CLI:

```bash
gemini mcp add --scope project \
  -e OPENCHAMI_MCP_MODE=read-only \
  -e OPENCHAMI_MCP_ENABLE_WRITES=false \
  -e OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true \
  -e OPENCHAMI_MCP_SMD_URL=https://localhost:27779 \
  -e OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY=true \
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
        "OPENCHAMI_MCP_SMD_URL": "https://localhost:27779",
        "OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY": "true",
        "OPENCHAMI_MCP_BSS_URL": "http://localhost:27778",
        "OPENCHAMI_MCP_CLOUD_INIT_URL": "http://localhost:27777",
        "OPENCHAMI_MCP_POWER_CONTROL_URL": "http://localhost:28007/v1",
        "OPENCHAMI_MCP_KEA_URL": "http://localhost:8000"
      }
    }
  }
}
```

For a remote `openchami-mcp` bound to `127.0.0.1:8081` on `msws`, Gemini CLI
supports the same two patterns.

Approach 1: separate SSH tunnel plus native HTTP MCP entry.

Start the tunnel in another terminal:

```bash
ssh -N -L 18081:127.0.0.1:8081 msws
```

Then add the project-scoped HTTP server:

```bash
gemini mcp add --scope project --transport http openchami-msws http://127.0.0.1:18081/mcp
```

Equivalent `.gemini/settings.json`:

```json
{
  "mcpServers": {
    "openchami-msws": {
      "url": "http://127.0.0.1:18081/mcp",
      "type": "http"
    }
  }
}
```

Approach 2: self-contained stdio wrapper.

```bash
gemini mcp add --scope project openchami-msws bash -lc 'set -euo pipefail
PORT=18081
ssh -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -N -L ${PORT}:127.0.0.1:8081 msws &
ssh_pid=$!
cleanup() { kill "$ssh_pid" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
for _ in $(seq 1 50); do
  curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null && break
  sleep 0.2
done
exec npx -y mcp-remote "http://127.0.0.1:${PORT}/mcp"'
```

Equivalent `.gemini/settings.json`:

```json
{
  "mcpServers": {
    "openchami-msws": {
      "command": "bash",
      "args": [
        "-lc",
        "set -euo pipefail\nPORT=18081\nssh -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -N -L ${PORT}:127.0.0.1:8081 msws &\nssh_pid=$!\ncleanup() { kill \"$ssh_pid\" 2>/dev/null || true; }\ntrap cleanup EXIT INT TERM\nfor _ in $(seq 1 50); do\n  curl -sf \"http://127.0.0.1:${PORT}/healthz\" >/dev/null && break\n  sleep 0.2\ndone\nexec npx -y mcp-remote \"http://127.0.0.1:${PORT}/mcp\""
      ]
    }
  }
}
```

Both Gemini variants were validated locally on `2026-04-14`. `gemini mcp list`
reported the HTTP and stdio-wrapper entries as `Connected`.

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
        "OPENCHAMI_MCP_SMD_URL": "https://localhost:27779",
        "OPENCHAMI_MCP_SMD_TLS_INSECURE_SKIP_VERIFY": "true",
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
