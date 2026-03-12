package backends

import (
	"context"
	"fmt"
	"net/url"

	"github.com/OpenCHAMI/openchami-mcp/internal/clients/httpjson"
)

type smdBackend struct {
	client *httpjson.Client
}

type bssBackend struct {
	client *httpjson.Client
}

type cloudInitBackend struct {
	client *httpjson.Client
}

type powerControlBackend struct {
	client *httpjson.Client
}

func NewSMDBackend(client *httpjson.Client) SMDBackend {
	return &smdBackend{client: client}
}

func NewBSSBackend(client *httpjson.Client) BSSBackend {
	return &bssBackend{client: client}
}

func NewCloudInitBackend(client *httpjson.Client) CloudInitBackend {
	return &cloudInitBackend{client: client}
}

func NewPowerControlBackend(client *httpjson.Client) PowerControlBackend {
	return &powerControlBackend{client: client}
}

func (b *smdBackend) Configured() bool { return b.client.Configured() }
func (b *smdBackend) BaseURL() string  { return b.client.BaseURL() }

func (b *smdBackend) Ready(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/service/ready", nil, nil)
}

func (b *smdBackend) ListComponents(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/State/Components", nil, nil)
}

func (b *smdBackend) GetComponent(ctx context.Context, xname string) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/State/Components/"+url.PathEscape(xname), nil, nil)
}

func (b *smdBackend) ListGroups(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/groups", nil, nil)
}

func (b *smdBackend) GetGroup(ctx context.Context, label string) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/groups/"+url.PathEscape(label), nil, nil)
}

func (b *smdBackend) GetMembership(ctx context.Context, xname string) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/memberships/"+url.PathEscape(xname), nil, nil)
}

func (b *smdBackend) CreateGroup(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/hsm/v2/groups", nil, payload)
}

func (b *smdBackend) PatchGroup(ctx context.Context, label string, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "PATCH", "/hsm/v2/groups/"+url.PathEscape(label), nil, payload)
}

func (b *smdBackend) DeleteGroup(ctx context.Context, label string) (any, error) {
	return doJSON(ctx, b.client, "DELETE", "/hsm/v2/groups/"+url.PathEscape(label), nil, nil)
}

func (b *smdBackend) GetGroupMembers(ctx context.Context, label string) (any, error) {
	return doJSON(ctx, b.client, "GET", "/hsm/v2/groups/"+url.PathEscape(label)+"/members", nil, nil)
}

func (b *smdBackend) SetGroupMembers(ctx context.Context, label string, members []string) (any, error) {
	return doJSON(ctx, b.client, "PUT", "/hsm/v2/groups/"+url.PathEscape(label)+"/members", nil, map[string]any{
		"group": label,
		"ids":   members,
	})
}

func (b *smdBackend) AddGroupMember(ctx context.Context, label string, member string) (any, error) {
	return doJSON(ctx, b.client, "POST", "/hsm/v2/groups/"+url.PathEscape(label)+"/members", nil, map[string]any{
		"id": member,
	})
}

func (b *smdBackend) DeleteGroupMember(ctx context.Context, label string, member string) (any, error) {
	return doJSON(ctx, b.client, "DELETE", "/hsm/v2/groups/"+url.PathEscape(label)+"/members/"+url.PathEscape(member), nil, nil)
}

func (b *bssBackend) Configured() bool { return b.client.Configured() }
func (b *bssBackend) BaseURL() string  { return b.client.BaseURL() }

func (b *bssBackend) ServiceStatus(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/boot/v1/service/status", nil, nil)
}

func (b *bssBackend) GetBootParameters(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/boot/v1/bootparameters", nil, nil)
}

func (b *bssBackend) GetBootScript(ctx context.Context, query url.Values) (any, error) {
	return doJSON(ctx, b.client, "GET", "/boot/v1/bootscript", query, nil)
}

func (b *bssBackend) ListHosts(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/boot/v1/hosts", nil, nil)
}

func (b *bssBackend) PutBootParameters(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "PUT", "/boot/v1/bootparameters", nil, payload)
}

func (b *bssBackend) PostBootParameters(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/boot/v1/bootparameters", nil, payload)
}

func (b *bssBackend) PatchBootParameters(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "PATCH", "/boot/v1/bootparameters", nil, payload)
}

func (b *bssBackend) DeleteBootParameters(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "DELETE", "/boot/v1/bootparameters", nil, payload)
}

func (b *bssBackend) PostHosts(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/boot/v1/hosts", nil, payload)
}

func (b *cloudInitBackend) Configured() bool { return b.client.Configured() }
func (b *cloudInitBackend) BaseURL() string  { return b.client.BaseURL() }

func (b *cloudInitBackend) Version(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/version", nil, nil)
}

func (b *cloudInitBackend) GetClusterDefaults(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/admin/cluster-defaults", nil, nil)
}

func (b *cloudInitBackend) SetClusterDefaults(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/admin/cluster-defaults", nil, payload)
}

func (b *cloudInitBackend) ListGroups(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/admin/groups", nil, nil)
}

func (b *cloudInitBackend) GetGroup(ctx context.Context, id string) (any, error) {
	return doJSON(ctx, b.client, "GET", "/admin/groups/"+url.PathEscape(id), nil, nil)
}

func (b *cloudInitBackend) CreateGroup(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/admin/groups", nil, payload)
}

func (b *cloudInitBackend) UpdateGroup(ctx context.Context, name string, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "PUT", "/admin/groups/"+url.PathEscape(name), nil, payload)
}

func (b *cloudInitBackend) DeleteGroup(ctx context.Context, id string) (any, error) {
	return doJSON(ctx, b.client, "DELETE", "/admin/groups/"+url.PathEscape(id), nil, nil)
}

func (b *cloudInitBackend) UpdateInstanceInfo(ctx context.Context, id string, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "PUT", "/admin/instance-info/"+url.PathEscape(id), nil, payload)
}

func (b *cloudInitBackend) GetMetaData(ctx context.Context, id string) (any, error) {
	return doJSON(ctx, b.client, "GET", cloudInitPath(id, "meta-data"), nil, nil)
}

func (b *cloudInitBackend) GetUserData(ctx context.Context, id string) (any, error) {
	return doJSON(ctx, b.client, "GET", cloudInitPath(id, "user-data"), nil, nil)
}

func (b *cloudInitBackend) GetVendorData(ctx context.Context, id string) (any, error) {
	return doJSON(ctx, b.client, "GET", cloudInitPath(id, "vendor-data"), nil, nil)
}

func (b *cloudInitBackend) GetGroupUserData(ctx context.Context, id string, group string) (any, error) {
	if id == "" {
		return doJSON(ctx, b.client, "GET", "/"+url.PathEscape(group)+".yaml", nil, nil)
	}
	return doJSON(ctx, b.client, "GET", "/admin/impersonation/"+url.PathEscape(id)+"/"+url.PathEscape(group)+".yaml", nil, nil)
}

func (b *powerControlBackend) Configured() bool { return b.client.Configured() }
func (b *powerControlBackend) BaseURL() string  { return b.client.BaseURL() }

func (b *powerControlBackend) Health(ctx context.Context) (any, error) {
	return doJSON(ctx, b.client, "GET", "/health", nil, nil)
}

func (b *powerControlBackend) GetPowerStatus(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/power-status", nil, payload)
}

func (b *powerControlBackend) CreateTransition(ctx context.Context, payload map[string]any) (any, error) {
	return doJSON(ctx, b.client, "POST", "/transitions", nil, payload)
}

func doJSON(ctx context.Context, client *httpjson.Client, method, path string, query url.Values, payload any) (any, error) {
	resp, err := client.Do(ctx, method, path, httpjson.RequestOptions{
		Query: query,
		Body:  payload,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func cloudInitPath(id string, suffix string) string {
	if id == "" {
		return "/" + suffix
	}
	return fmt.Sprintf("/admin/impersonation/%s/%s", url.PathEscape(id), suffix)
}
