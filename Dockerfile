# syntax=docker/dockerfile:1.7

FROM golang:1.24 AS builder
ARG CGO_ENABLED=0

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o /out/openchami-mcp ./cmd/openchami-mcp

FROM cgr.dev/chainguard/wolfi-base:latest

RUN apk add --no-cache ca-certificates tini

ENV OPENCHAMI_MCP_MODE=read-only \
    OPENCHAMI_MCP_ENABLE_WRITES=false \
    OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true \
    OPENCHAMI_MCP_FRAMING=auto \
    OPENCHAMI_MCP_LOG_LEVEL=info \
    OPENCHAMI_MCP_SMD_TIMEOUT=10s \
    OPENCHAMI_MCP_SMD_RETRIES=1 \
    OPENCHAMI_MCP_BSS_TIMEOUT=10s \
    OPENCHAMI_MCP_BSS_RETRIES=1 \
    OPENCHAMI_MCP_CLOUD_INIT_TIMEOUT=10s \
    OPENCHAMI_MCP_CLOUD_INIT_RETRIES=1 \
    OPENCHAMI_MCP_POWER_CONTROL_TIMEOUT=10s \
    OPENCHAMI_MCP_POWER_CONTROL_RETRIES=1 \
    OPENCHAMI_MCP_KEA_TIMEOUT=10s \
    OPENCHAMI_MCP_KEA_RETRIES=1 \
    OPENCHAMI_MCP_KEA_DEFAULT_SERVICES=dhcp4 \
    OPENCHAMI_MCP_COREDHCP_TIMEOUT=10s \
    OPENCHAMI_MCP_COREDHCP_RETRIES=1

COPY --from=builder /out/openchami-mcp /usr/local/bin/openchami-mcp

USER 65534:65534

CMD ["/usr/local/bin/openchami-mcp"]
ENTRYPOINT ["/sbin/tini", "--"]
