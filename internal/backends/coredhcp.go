package backends

import (
	"context"

	"github.com/OpenCHAMI/openchami-mcp/internal/clients/httpjson"
	"github.com/OpenCHAMI/openchami-mcp/internal/errorsx"
)

type coreDHCPBackend struct {
	client *httpjson.Client
}

func NewCoreDHCPBackend(client *httpjson.Client) CoreDHCPBackend {
	return &coreDHCPBackend{client: client}
}

func (b *coreDHCPBackend) Configured() bool { return b.client.Configured() }
func (b *coreDHCPBackend) BaseURL() string  { return b.client.BaseURL() }

func (b *coreDHCPBackend) Status(context.Context) (any, error) {
	return map[string]any{
		"configured": b.Configured(),
		"base_url":   b.BaseURL(),
		"status":     "unsupported",
		"reason":     "CoreDHCP does not expose a standard management API in the checked-out upstream repo; a deployment-specific admin surface still needs to be defined.",
	}, errorsx.NewNotImplemented("CoreDHCP integration requires a concrete management API")
}
