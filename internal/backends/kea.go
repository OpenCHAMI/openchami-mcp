package backends

import (
	"context"

	"github.com/OpenCHAMI/openchami-mcp/internal/clients/httpjson"
)

type keaBackend struct {
	client          *httpjson.Client
	defaultServices []string
}

func NewKeaBackend(client *httpjson.Client, defaultServices []string) KeaBackend {
	return &keaBackend{
		client:          client,
		defaultServices: append([]string(nil), defaultServices...),
	}
}

func (b *keaBackend) Configured() bool { return b.client.Configured() }
func (b *keaBackend) BaseURL() string  { return b.client.BaseURL() }

func (b *keaBackend) DefaultServices() []string {
	return append([]string(nil), b.defaultServices...)
}

func (b *keaBackend) RunCommand(ctx context.Context, command string, arguments map[string]any, services []string) (any, error) {
	payload := map[string]any{
		"command": command,
	}
	if arguments != nil {
		payload["arguments"] = arguments
	}
	if len(services) > 0 {
		payload["service"] = services
	}

	resp, err := b.client.Do(ctx, "POST", "/", httpjson.RequestOptions{
		Body: payload,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}
