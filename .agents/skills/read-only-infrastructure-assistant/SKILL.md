---
name: read-only-infrastructure-assistant
description: Inspect OpenCHAMI infrastructure state and explain the platform without making changes. Use this for safe read-only inventory, boot, metadata, power, and DHCP analysis through openchami-mcp.
license: MIT
compatibility: opencode
metadata:
  audience: operators
  safety: read-only
---

## What I do

Use this skill when the task is to understand or inspect an OpenCHAMI
environment without changing cluster state.

OpenCHAMI is a microservice-based HPC cluster provisioning and management
system. In this repository context, the most important services are:

- SMD for inventory and group membership
- BSS for boot parameters and bootscripts
- cloud-init for metadata and rendered instance data
- power-control for node and BMC power state
- Kea and CoreDHCP integration points for DHCP and lease workflows

`openchami-mcp` exposes these systems to an agent over MCP with stable tool
names, so the client can inspect the platform without calling each service API
directly.

## Default operating mode

Prefer read-only access.

Assume these environment settings unless the user explicitly asks for write
operations and confirms that writes are intended:

- `OPENCHAMI_MCP_MODE=read-only`
- `OPENCHAMI_MCP_ENABLE_WRITES=false`
- `OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true`

If a task would require a write-capable tool, stop and state that the current
skill is intentionally read-only.

## How to work

Start with discovery and status:

- check service health first
- inspect current inventory, group, boot, metadata, lease, or power state
- summarize findings in infrastructure terms instead of MCP internals when
  possible

Prefer these behaviors:

- explain the role of each OpenCHAMI service before diving into raw payloads
- call out missing configuration or unconfigured services explicitly
- distinguish verified cluster state from inference
- keep recommendations operationally safe and non-destructive

## When to use me

Use this skill when a user asks for:

- a read-only overview of OpenCHAMI
- safe infrastructure inspection
- cluster status explanations
- help mapping OpenCHAMI concepts to SMD, BSS, cloud-init, power-control, Kea,
  or CoreDHCP

Do not use this skill for change execution, write testing, or mutation
workflows.
