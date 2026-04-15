# openchami-mcp HTTP Server Mode Plan

Date: 2026-04-14

## Goal

Add an HTTP transport to `openchami-mcp` so it can run as a normal network
service in addition to the existing stdio mode. The target should be the modern
MCP Streamable HTTP transport, not the deprecated legacy HTTP+SSE transport from
protocol version `2024-11-05`.

## Current State

- `openchami-mcp` only serves MCP over stdio.
- The entrypoint wires the server to `os.Stdin` and `os.Stdout`.
- Request handling and framing are currently coupled to the stdio loop in
  `internal/mcp/server.go` and `internal/mcp/framing.go`.
- The packaged `openchami-mcp.container` on `ochami-tenv1` keeps the process alive with
  a FIFO bridge because systemd is not itself an MCP client.

## Planning Decisions

1. Keep stdio support intact.
2. Implement Streamable HTTP as the HTTP mode.
3. Start with a stateless HTTP implementation that handles MCP over `POST` and
   returns `405 Method Not Allowed` for `GET` when no SSE stream is offered.
4. Bind to `127.0.0.1` by default and require explicit configuration to expose
   the service on other interfaces.
5. Add origin validation before any non-local deployment.
6. Treat session management, resumability, and standalone SSE streams as a
   second phase unless a concrete client requires them.

## Why This Shape

The current server does not emit unsolicited server-to-client messages, so a
full SSE/session implementation is not required to get usable HTTP transport.
The MCP Streamable HTTP transport allows the server to reject `GET` with `405`
when no SSE stream is offered, which keeps the first implementation smaller and
safer.

## Implementation Phases

### Phase 1: Transport Refactor

Refactor the current MCP loop so the transport layer is separate from request
handling.

Work:

- Extract a transport-agnostic request handler from `Server.Serve`.
- Keep the existing tool dispatch, error mapping, and JSON-RPC semantics
  unchanged.
- Move stdio framing into a transport-specific layer.
- Introduce an explicit transport selection in config and flags, for example:
  - `OPENCHAMI_MCP_TRANSPORT=stdio|http`
  - `OPENCHAMI_MCP_LISTEN_ADDR=127.0.0.1:8081`
  - `OPENCHAMI_MCP_BASE_PATH=/mcp`
  - `OPENCHAMI_MCP_ALLOWED_ORIGINS=...`

Deliverable:

- A transport-independent core that can process one decoded JSON-RPC message and
  return zero or more response messages.

### Phase 2: Minimal Streamable HTTP Mode

Add an HTTP server transport using `net/http`.

Work:

- Add a single MCP endpoint, default `/mcp`.
- Support `POST` with `Content-Type: application/json`.
- Accept one JSON-RPC request per `POST` in the first iteration.
- Return one JSON-RPC response as JSON when the request expects a response.
- Return `202` or `204` for notifications with no response body, depending on
  the final transport behavior selected during implementation.
- Reject `GET` with `405 Method Not Allowed` when SSE is not enabled.
- Validate `Origin` on incoming requests.
- Add a small health endpoint such as `/healthz` for systemd and deployment
  checks.

Deliverable:

- `openchami-mcp` can run as a normal HTTP service and serve `initialize`,
  `tools/list`, and `tools/call` over HTTP.

### Phase 3: Protocol Version Alignment

Resolve the protocol-version mismatch between the current server and modern HTTP
transport.

Work:

- Review the impact of moving from `2024-11-05` to the current MCP revision used
  for Streamable HTTP.
- Decide whether to:
  - bump the whole server to the newer MCP protocol version, or
  - gate protocol behavior by transport during a short compatibility window.
- Explicitly decide whether JSON-RPC batch requests need to be supported in the
  first HTTP release.

Recommendation:

- Prefer one protocol version across both stdio and HTTP, rather than letting
  transport determine the advertised protocol version.

### Phase 4: Security and Remote-Use Controls

Make the HTTP mode safe enough to expose deliberately.

Work:

- Enforce origin validation with a configurable allowlist.
- Default bind address to localhost.
- Add optional bearer-token authentication for HTTP mode.
- Consider refusing `read-write` mode on non-local bind addresses unless auth is
  explicitly enabled.
- Log client address, method, path, and response status without leaking tool
  inputs by default.

Deliverable:

- A remote-capable HTTP mode that is not trivially unsafe by default.

### Phase 5: Packaging and Deployment

Replace the current FIFO-based quadlet when HTTP mode is ready.

Work:

- Add an HTTP-oriented quadlet or update the existing
  `deploy/quadlets/containers/openchami-mcp.container`.
- Remove the FIFO bridge for HTTP deployments.
- Wire the service into `openchami.target` as a normal daemon.
- Document the port, firewall rules, and local-only default.
- Add an `ochami-tenv1` deployment path that points the quadlet at HTTP mode for live
  testing.

Deliverable:

- `openchami-mcp` behaves like the other services in the stack when deployed via
  RPM and quadlets.

### Phase 6: Optional SSE and Session Support

Only do this if a real client or registry use case requires it.

Work:

- Add optional SSE streaming on `GET /mcp`.
- Add session IDs and any required session store.
- Add resumability only if needed.
- Add compatibility endpoints for older HTTP+SSE clients only if a concrete
  consumer still depends on them.

Deliverable:

- A richer remote MCP transport, if the simpler stateless mode proves
  insufficient.

## Test Plan

Add or update tests in these layers:

- `internal/mcp` unit tests for transport-independent request handling.
- `httptest` coverage for:
  - `POST /mcp` initialize
  - `POST /mcp` tools/list
  - `POST /mcp` tools/call in read-only mode
  - invalid JSON
  - unsupported method
  - invalid or missing `Origin`
  - `GET /mcp` returning `405` when SSE is disabled
- config tests for new HTTP env vars and defaults.
- a small end-to-end test that boots the HTTP server against fake backends and
  runs an MCP client flow.

## Acceptance Criteria

- `go run ./cmd/openchami-mcp -transport http` starts an HTTP server.
- A local client can complete `initialize`, `tools/list`, and a read-only
  `tools/call` over HTTP.
- Stdio mode still works unchanged.
- The HTTP mode defaults to localhost-only binding.
- Invalid origins are rejected.
- RPM/quadlet deployment can run `openchami-mcp` without the FIFO workaround.

## Open Questions

1. Do we need remote HTTP access immediately, or is localhost plus reverse proxy
   enough for the first release?
2. Do we need backward compatibility for legacy `2024-11-05` HTTP+SSE clients,
   or can we standardize on Streamable HTTP only?
3. Do any planned MCP clients for `ochami-tenv1` require SSE or resumable sessions, or
   is plain request/response enough?
4. Should HTTP auth be built into `openchami-mcp`, or delegated to a reverse
   proxy for the first deployment?

## References

- MCP transports spec: https://modelcontextprotocol.io/specification/2025-11-25/basic/transports
- MCP changelog noting Streamable HTTP replacing legacy HTTP+SSE:
  https://modelcontextprotocol.io/specification/2025-03-26/changelog
