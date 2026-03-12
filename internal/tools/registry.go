package tools

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"slices"

	"github.com/OpenCHAMI/openchami-mcp/internal/backends"
	"github.com/OpenCHAMI/openchami-mcp/internal/config"
	"github.com/OpenCHAMI/openchami-mcp/internal/errorsx"
)

type Definition struct {
	Name        string
	Description string
	InputSchema map[string]any
	ReadOnly    bool
}

type tool struct {
	definition Definition
	handler    func(context.Context, map[string]any) (any, error)
}

type Dependencies struct {
	Config config.Config
	Logger *slog.Logger
	SMD    backends.SMDBackend
	BSS    backends.BSSBackend
	Cloud  backends.CloudInitBackend
	Power  backends.PowerControlBackend
	Kea    backends.KeaBackend
	Core   backends.CoreDHCPBackend
}

type Registry struct {
	definitions []Definition
	tools       map[string]tool
	deps        Dependencies
}

func NewRegistry(deps Dependencies) *Registry {
	r := &Registry{
		deps:  deps,
		tools: make(map[string]tool),
	}

	r.registerOpenCHAMI()
	r.registerSMD()
	r.registerBSS()
	r.registerCloudInit()
	r.registerPowerControl()
	r.registerKea()

	return r
}

func (r *Registry) Definitions() []Definition {
	return append([]Definition(nil), r.definitions...)
}

func (r *Registry) Definition(name string) (Definition, bool) {
	t, ok := r.tools[name]
	if !ok {
		return Definition{}, false
	}
	return t.definition, true
}

func (r *Registry) Call(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, errorsx.NewInvalidArgument("Unknown tool: %s", name)
	}
	if args == nil {
		args = map[string]any{}
	}
	return t.handler(ctx, args)
}

func (r *Registry) registerOpenCHAMI() {
	r.add(Definition{
		Name:        "openchami_health",
		Description: "Check readiness or status across the configured OpenCHAMI, Kea, and CoreDHCP integrations.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		out := map[string]any{
			"smd":           r.checkHealth(ctx, r.deps.SMD.Configured(), r.deps.SMD.BaseURL(), r.deps.SMD.Ready),
			"bss":           r.checkHealth(ctx, r.deps.BSS.Configured(), r.deps.BSS.BaseURL(), r.deps.BSS.ServiceStatus),
			"cloud_init":    r.checkHealth(ctx, r.deps.Cloud.Configured(), r.deps.Cloud.BaseURL(), r.deps.Cloud.Version),
			"power_control": r.checkHealth(ctx, r.deps.Power.Configured(), r.deps.Power.BaseURL(), r.deps.Power.Health),
			"kea": r.checkHealth(ctx, r.deps.Kea.Configured(), r.deps.Kea.BaseURL(), func(ctx context.Context) (any, error) {
				return r.deps.Kea.RunCommand(ctx, "status-get", nil, r.deps.Kea.DefaultServices())
			}),
			"coredhcp": r.checkHealth(ctx, r.deps.Core.Configured(), r.deps.Core.BaseURL(), r.deps.Core.Status),
		}
		return out, nil
	})
}

func (r *Registry) registerSMD() {
	r.add(Definition{
		Name:        "hsm_list_components",
		Description: "List HSM components from SMD.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.SMD.ListComponents(ctx)
	})
	r.add(Definition{
		Name:        "hsm_get_component",
		Description: "Get a single HSM component by xname.",
		InputSchema: objectSchema(map[string]any{
			"xname": stringSchema(),
		}, "xname"),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		xname, err := requiredString(args, "xname")
		if err != nil {
			return nil, err
		}
		return r.deps.SMD.GetComponent(ctx, xname)
	})
	r.add(Definition{
		Name:        "hsm_list_groups",
		Description: "List HSM groups.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.SMD.ListGroups(ctx)
	})
	r.add(Definition{
		Name:        "hsm_get_group",
		Description: "Get an HSM group by label.",
		InputSchema: objectSchema(map[string]any{
			"label": stringSchema(),
		}, "label"),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		label, err := requiredString(args, "label")
		if err != nil {
			return nil, err
		}
		return r.deps.SMD.GetGroup(ctx, label)
	})
	r.add(Definition{
		Name:        "hsm_get_group_members",
		Description: "Get the members of an HSM group.",
		InputSchema: objectSchema(map[string]any{
			"label": stringSchema(),
		}, "label"),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		label, err := requiredString(args, "label")
		if err != nil {
			return nil, err
		}
		return r.deps.SMD.GetGroupMembers(ctx, label)
	})
	r.add(Definition{
		Name:        "hsm_get_membership",
		Description: "Get group membership information for an xname.",
		InputSchema: objectSchema(map[string]any{
			"xname": stringSchema(),
		}, "xname"),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		xname, err := requiredString(args, "xname")
		if err != nil {
			return nil, err
		}
		return r.deps.SMD.GetMembership(ctx, xname)
	})
	r.add(Definition{
		Name:        "hsm_create_group",
		Description: "Create an HSM group.",
		InputSchema: objectSchema(map[string]any{
			"label":          stringSchema(),
			"description":    stringSchema(),
			"exclusiveGroup": stringSchema(),
			"tags":           stringListSchema(0),
			"members":        stringListSchema(0),
			"payload":        objectValueSchema(),
		}),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		payload, err := optionalObject(args, "payload")
		if err != nil {
			return nil, err
		}
		if payload == nil {
			label, err := requiredString(args, "label")
			if err != nil {
				return nil, err
			}
			payload = map[string]any{"label": label}
			if description, ok, err := optionalString(args, "description"); err != nil {
				return nil, err
			} else if ok {
				payload["description"] = description
			}
			if exclusiveGroup, ok, err := optionalString(args, "exclusiveGroup"); err != nil {
				return nil, err
			} else if ok {
				payload["exclusiveGroup"] = exclusiveGroup
			}
			if tags, ok, err := optionalStringList(args, "tags"); err != nil {
				return nil, err
			} else if ok {
				payload["tags"] = tags
			}
			if members, ok, err := optionalStringList(args, "members"); err != nil {
				return nil, err
			} else if ok {
				payload["members"] = map[string]any{"ids": members}
			}
		}
		return r.deps.SMD.CreateGroup(ctx, payload)
	})
	r.add(Definition{
		Name:        "hsm_patch_group",
		Description: "Patch an HSM group's description or tags.",
		InputSchema: objectSchema(map[string]any{
			"label":       stringSchema(),
			"description": stringSchema(),
			"tags":        stringListSchema(0),
			"payload":     objectValueSchema(),
		}, "label"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		label, err := requiredString(args, "label")
		if err != nil {
			return nil, err
		}
		payload, err := optionalObject(args, "payload")
		if err != nil {
			return nil, err
		}
		if payload == nil {
			payload = map[string]any{}
			if description, ok, err := optionalString(args, "description"); err != nil {
				return nil, err
			} else if ok {
				payload["description"] = description
			}
			if tags, ok, err := optionalStringList(args, "tags"); err != nil {
				return nil, err
			} else if ok {
				payload["tags"] = tags
			}
			if len(payload) == 0 {
				return nil, errorsx.NewInvalidArgument("one of 'description', 'tags', or 'payload' is required")
			}
		}
		return r.deps.SMD.PatchGroup(ctx, label, payload)
	})
	r.add(Definition{
		Name:        "hsm_delete_group",
		Description: "Delete an HSM group by label.",
		InputSchema: objectSchema(map[string]any{
			"label": stringSchema(),
		}, "label"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		label, err := requiredString(args, "label")
		if err != nil {
			return nil, err
		}
		return r.deps.SMD.DeleteGroup(ctx, label)
	})
	r.add(Definition{
		Name:        "hsm_update_group_members",
		Description: "Set, add, or remove HSM group members. `action` supports replace, add, or remove.",
		InputSchema: objectSchema(map[string]any{
			"label":  stringSchema(),
			"action": stringSchema(),
			"method": stringSchema(),
			"member": stringSchema(),
			"members": map[string]any{
				"type":  "array",
				"items": stringSchema(),
			},
			"payload": objectValueSchema(),
		}, "label"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		label, err := requiredString(args, "label")
		if err != nil {
			return nil, err
		}
		method, err := resolveMembershipMethod(args)
		if err != nil {
			return nil, err
		}
		payload, err := optionalObject(args, "payload")
		if err != nil {
			return nil, err
		}
		members, err := extractMembers(args, payload)
		if err != nil {
			return nil, err
		}

		switch method {
		case "PUT":
			if len(members) == 0 {
				return nil, errorsx.NewInvalidArgument("'members' is required for replace operations")
			}
			return r.deps.SMD.SetGroupMembers(ctx, label, members)
		case "POST":
			if len(members) == 0 {
				return nil, errorsx.NewInvalidArgument("'member' or 'members' is required for add operations")
			}
			if len(members) == 1 {
				return r.deps.SMD.AddGroupMember(ctx, label, members[0])
			}
			out := make([]any, 0, len(members))
			for _, member := range members {
				resp, err := r.deps.SMD.AddGroupMember(ctx, label, member)
				if err != nil {
					return nil, err
				}
				out = append(out, resp)
			}
			return out, nil
		case "DELETE":
			if len(members) == 0 {
				return nil, errorsx.NewInvalidArgument("'member' or 'members' is required for remove operations")
			}
			if len(members) == 1 {
				return r.deps.SMD.DeleteGroupMember(ctx, label, members[0])
			}
			out := make([]any, 0, len(members))
			for _, member := range members {
				resp, err := r.deps.SMD.DeleteGroupMember(ctx, label, member)
				if err != nil {
					return nil, err
				}
				out = append(out, resp)
			}
			return out, nil
		default:
			return nil, errorsx.NewInvalidArgument("unsupported member update method %q", method)
		}
	})
}

func (r *Registry) registerBSS() {
	r.add(Definition{
		Name:        "bss_service_status",
		Description: "Get BSS service status.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.BSS.ServiceStatus(ctx)
	})
	r.add(Definition{
		Name:        "bss_get_bootparameters",
		Description: "Get BSS boot parameters.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.BSS.GetBootParameters(ctx)
	})
	r.add(Definition{
		Name:        "bss_get_bootscript",
		Description: "Fetch a BSS bootscript by MAC, name, or NID.",
		InputSchema: objectSchema(map[string]any{
			"mac":   stringSchema(),
			"name":  stringSchema(),
			"nid":   map[string]any{"type": []string{"string", "integer"}},
			"retry": map[string]any{"type": "integer", "minimum": 0},
		}),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		query := url.Values{}
		for _, key := range []string{"mac", "name"} {
			if value, ok, err := optionalString(args, key); err != nil {
				return nil, err
			} else if ok {
				query.Set(key, value)
			}
		}
		if nid, ok, err := optionalStringish(args, "nid"); err != nil {
			return nil, err
		} else if ok {
			query.Set("nid", nid)
		}
		if retry, ok, err := optionalInt(args, "retry"); err != nil {
			return nil, err
		} else if ok {
			query.Set("retry", fmt.Sprintf("%d", retry))
		}
		if len(query) == 0 {
			return nil, errorsx.NewInvalidArgument("one of 'mac', 'name', or 'nid' is required")
		}
		return r.deps.BSS.GetBootScript(ctx, query)
	})
	r.add(Definition{
		Name:        "bss_list_hosts",
		Description: "List BSS hosts.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.BSS.ListHosts(ctx)
	})
	r.registerBSSWrite("bss_put_bootparameters", "Replace BSS boot parameters.", r.deps.BSS.PutBootParameters)
	r.registerBSSWrite("bss_post_bootparameters", "Create BSS boot parameters.", r.deps.BSS.PostBootParameters)
	r.registerBSSWrite("bss_patch_bootparameters", "Patch BSS boot parameters.", r.deps.BSS.PatchBootParameters)
	r.registerBSSWrite("bss_delete_bootparameters", "Delete BSS boot parameters.", r.deps.BSS.DeleteBootParameters)
	r.registerBSSWrite("bss_hosts_post", "Create or update BSS hosts.", r.deps.BSS.PostHosts)
}

func (r *Registry) registerCloudInit() {
	r.add(Definition{
		Name:        "cloudinit_get_cluster_defaults",
		Description: "Get cloud-init cluster defaults.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.Cloud.GetClusterDefaults(ctx)
	})
	r.add(Definition{
		Name:        "cloudinit_set_cluster_defaults",
		Description: "Set cloud-init cluster defaults.",
		InputSchema: objectSchema(map[string]any{
			"payload": objectValueSchema(),
		}, "payload"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		payload, err := requiredObject(args, "payload")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.SetClusterDefaults(ctx, payload)
	})
	r.add(Definition{
		Name:        "cloudinit_list_groups",
		Description: "List cloud-init groups.",
		InputSchema: objectSchema(map[string]any{}),
		ReadOnly:    true,
	}, func(ctx context.Context, _ map[string]any) (any, error) {
		return r.deps.Cloud.ListGroups(ctx)
	})
	r.add(Definition{
		Name:        "cloudinit_get_group",
		Description: "Get a cloud-init group by ID.",
		InputSchema: objectSchema(map[string]any{
			"id": stringSchema(),
		}, "id"),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		id, err := requiredString(args, "id")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.GetGroup(ctx, id)
	})
	r.add(Definition{
		Name:        "cloudinit_create_group",
		Description: "Create a cloud-init group.",
		InputSchema: objectSchema(map[string]any{
			"payload": objectValueSchema(),
		}, "payload"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		payload, err := requiredObject(args, "payload")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.CreateGroup(ctx, payload)
	})
	r.add(Definition{
		Name:        "cloudinit_update_group",
		Description: "Update a cloud-init group by name.",
		InputSchema: objectSchema(map[string]any{
			"name":    stringSchema(),
			"payload": objectValueSchema(),
		}, "name", "payload"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		name, err := requiredString(args, "name")
		if err != nil {
			return nil, err
		}
		payload, err := requiredObject(args, "payload")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.UpdateGroup(ctx, name, payload)
	})
	r.add(Definition{
		Name:        "cloudinit_delete_group",
		Description: "Delete a cloud-init group.",
		InputSchema: objectSchema(map[string]any{
			"id": stringSchema(),
		}, "id"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		id, err := requiredString(args, "id")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.DeleteGroup(ctx, id)
	})
	r.add(Definition{
		Name:        "cloudinit_update_instance_info",
		Description: "Set cloud-init instance info for a node.",
		InputSchema: objectSchema(map[string]any{
			"id":      stringSchema(),
			"payload": objectValueSchema(),
		}, "id", "payload"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		id, err := requiredString(args, "id")
		if err != nil {
			return nil, err
		}
		payload, err := requiredObject(args, "payload")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.UpdateInstanceInfo(ctx, id, payload)
	})
	r.add(Definition{
		Name:        "cloudinit_get_meta_data",
		Description: "Get cloud-init meta-data, optionally impersonating a specific node ID.",
		InputSchema: objectSchema(map[string]any{
			"id": stringSchema(),
		}),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		id, _, err := optionalString(args, "id")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.GetMetaData(ctx, id)
	})
	r.add(Definition{
		Name:        "cloudinit_get_user_data",
		Description: "Get cloud-init user-data, optionally impersonating a specific node ID.",
		InputSchema: objectSchema(map[string]any{
			"id": stringSchema(),
		}),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		id, _, err := optionalString(args, "id")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.GetUserData(ctx, id)
	})
	r.add(Definition{
		Name:        "cloudinit_get_vendor_data",
		Description: "Get cloud-init vendor-data, optionally impersonating a specific node ID.",
		InputSchema: objectSchema(map[string]any{
			"id": stringSchema(),
		}),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		id, _, err := optionalString(args, "id")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.GetVendorData(ctx, id)
	})
	r.add(Definition{
		Name:        "cloudinit_get_group_user_data",
		Description: "Get group-specific cloud-init YAML, optionally impersonating a specific node ID.",
		InputSchema: objectSchema(map[string]any{
			"id":    stringSchema(),
			"group": stringSchema(),
		}, "group"),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		group, err := requiredString(args, "group")
		if err != nil {
			return nil, err
		}
		id, _, err := optionalString(args, "id")
		if err != nil {
			return nil, err
		}
		return r.deps.Cloud.GetGroupUserData(ctx, id, group)
	})
}

func (r *Registry) registerPowerControl() {
	r.add(Definition{
		Name:        "pcs_power_status",
		Description: "Get power status for one or more xnames.",
		InputSchema: objectSchema(map[string]any{
			"xnames":                stringListSchema(1),
			"powerStateFilter":      stringSchema(),
			"managementStateFilter": stringSchema(),
			"payload":               objectValueSchema(),
		}),
		ReadOnly: true,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		payload, err := optionalObject(args, "payload")
		if err != nil {
			return nil, err
		}
		if payload == nil {
			xnames, err := requiredStringList(args, "xnames")
			if err != nil {
				return nil, err
			}
			payload = map[string]any{"xname": xnames}
			if filter, ok, err := optionalString(args, "powerStateFilter"); err != nil {
				return nil, err
			} else if ok {
				payload["powerStateFilter"] = filter
			}
			if filter, ok, err := optionalString(args, "managementStateFilter"); err != nil {
				return nil, err
			} else if ok {
				payload["managementStateFilter"] = filter
			}
		}
		return r.deps.Power.GetPowerStatus(ctx, payload)
	})
	r.add(Definition{
		Name:        "pcs_transition",
		Description: "Create a power transition for one or more xnames.",
		InputSchema: objectSchema(map[string]any{
			"operation":           stringSchema(),
			"xnames":              stringListSchema(1),
			"taskDeadlineMinutes": map[string]any{"type": "integer", "minimum": 1},
			"payload":             objectValueSchema(),
		}),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		payload, err := optionalObject(args, "payload")
		if err != nil {
			return nil, err
		}
		if payload == nil {
			operation, err := requiredString(args, "operation")
			if err != nil {
				return nil, err
			}
			xnames, err := requiredStringList(args, "xnames")
			if err != nil {
				return nil, err
			}
			location := make([]map[string]any, 0, len(xnames))
			for _, xname := range xnames {
				location = append(location, map[string]any{"xname": xname})
			}
			payload = map[string]any{
				"operation": operation,
				"location":  location,
			}
			if deadline, ok, err := optionalInt(args, "taskDeadlineMinutes"); err != nil {
				return nil, err
			} else if ok {
				payload["taskDeadlineMinutes"] = deadline
			}
		}
		return r.deps.Power.CreateTransition(ctx, payload)
	})
}

func (r *Registry) registerKea() {
	r.add(Definition{
		Name:        "kea_run_command",
		Description: "Run an arbitrary Kea Control Agent command.",
		InputSchema: objectSchema(map[string]any{
			"command":   stringSchema(),
			"arguments": objectValueSchema(),
			"service":   stringSchema(),
			"services":  stringListSchema(0),
		}, "command"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		command, err := requiredString(args, "command")
		if err != nil {
			return nil, err
		}
		arguments, err := optionalObject(args, "arguments")
		if err != nil {
			return nil, err
		}
		services, err := resolveServices(args, r.deps.Kea.DefaultServices())
		if err != nil {
			return nil, err
		}
		return r.deps.Kea.RunCommand(ctx, command, arguments, services)
	})
	r.registerKeaCommand("kea_list_commands", "List available Kea commands.", true, "list-commands", nil)
	r.registerKeaCommand("kea_status_get", "Get Kea daemon status.", true, "status-get", nil)
	r.registerKeaCommand("kea_config_get", "Get the current Kea configuration.", true, "config-get", nil)
	r.registerKeaCommand("kea_config_write", "Write the current Kea configuration to disk.", false, "config-write", nil)
	r.add(Definition{
		Name:        "kea_config_set",
		Description: "Replace the current Kea configuration for one or more services.",
		InputSchema: objectSchema(map[string]any{
			"arguments": objectValueSchema(),
			"service":   stringSchema(),
			"services":  stringListSchema(0),
		}, "arguments"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		arguments, err := requiredObject(args, "arguments")
		if err != nil {
			return nil, err
		}
		services, err := resolveServices(args, r.deps.Kea.DefaultServices())
		if err != nil {
			return nil, err
		}
		return r.deps.Kea.RunCommand(ctx, "config-set", arguments, services)
	})
	r.registerKeaLeaseTool("kea_lease4_get", true, "lease4-get", "Query an IPv4 lease.")
	r.registerKeaLeaseTool("kea_lease4_add", false, "lease4-add", "Create an IPv4 lease.")
	r.registerKeaLeaseTool("kea_lease4_update", false, "lease4-update", "Update an IPv4 lease.")
	r.registerKeaLeaseTool("kea_lease4_del", false, "lease4-del", "Delete an IPv4 lease.")
	r.registerKeaLeaseTool("kea_lease6_get", true, "lease6-get", "Query an IPv6 lease.")
	r.registerKeaLeaseTool("kea_lease6_add", false, "lease6-add", "Create an IPv6 lease.")
	r.registerKeaLeaseTool("kea_lease6_update", false, "lease6-update", "Update an IPv6 lease.")
	r.registerKeaLeaseTool("kea_lease6_del", false, "lease6-del", "Delete an IPv6 lease.")
}

func (r *Registry) registerBSSWrite(name, description string, handler func(context.Context, map[string]any) (any, error)) {
	r.add(Definition{
		Name:        name,
		Description: description,
		InputSchema: objectSchema(map[string]any{
			"payload": objectValueSchema(),
		}, "payload"),
		ReadOnly: false,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		payload, err := requiredObject(args, "payload")
		if err != nil {
			return nil, err
		}
		return handler(ctx, payload)
	})
}

func (r *Registry) registerKeaCommand(name, description string, readOnly bool, command string, arguments map[string]any) {
	r.add(Definition{
		Name:        name,
		Description: description,
		InputSchema: objectSchema(map[string]any{
			"service":  stringSchema(),
			"services": stringListSchema(0),
		}),
		ReadOnly: readOnly,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		services, err := resolveServices(args, r.deps.Kea.DefaultServices())
		if err != nil {
			return nil, err
		}
		return r.deps.Kea.RunCommand(ctx, command, arguments, services)
	})
}

func (r *Registry) registerKeaLeaseTool(name string, readOnly bool, command string, description string) {
	r.add(Definition{
		Name:        name,
		Description: description,
		InputSchema: objectSchema(map[string]any{
			"arguments": objectValueSchema(),
			"service":   stringSchema(),
			"services":  stringListSchema(0),
		}, "arguments"),
		ReadOnly: readOnly,
	}, func(ctx context.Context, args map[string]any) (any, error) {
		arguments, err := requiredObject(args, "arguments")
		if err != nil {
			return nil, err
		}
		services, err := resolveServices(args, r.deps.Kea.DefaultServices())
		if err != nil {
			return nil, err
		}
		return r.deps.Kea.RunCommand(ctx, command, arguments, services)
	})
}

func (r *Registry) add(def Definition, handler func(context.Context, map[string]any) (any, error)) {
	r.definitions = append(r.definitions, def)
	r.tools[def.Name] = tool{definition: def, handler: handler}
}

func (r *Registry) checkHealth(ctx context.Context, configured bool, baseURL string, fn func(context.Context) (any, error)) map[string]any {
	if !configured {
		return map[string]any{
			"configured": false,
			"base_url":   baseURL,
		}
	}

	response, err := fn(ctx)
	out := map[string]any{
		"configured": true,
		"base_url":   baseURL,
	}
	if err != nil {
		out["ok"] = false
		out["error"] = err.Error()
		if response != nil {
			out["response"] = response
		}
		return out
	}
	out["ok"] = true
	out["response"] = response
	return out
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func objectValueSchema() map[string]any {
	return map[string]any{"type": "object"}
}

func stringListSchema(minItems int) map[string]any {
	schema := map[string]any{
		"type":  "array",
		"items": stringSchema(),
	}
	if minItems > 0 {
		schema["minItems"] = minItems
	}
	return schema
}

func requiredString(args map[string]any, key string) (string, error) {
	value, ok, err := optionalString(args, key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errorsx.NewInvalidArgument("'%s' must be a non-empty string", key)
	}
	return value, nil
}

func optionalString(args map[string]any, key string) (string, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", false, nil
	}
	value, ok := raw.(string)
	if !ok || value == "" {
		return "", false, errorsx.NewInvalidArgument("'%s' must be a non-empty string", key)
	}
	return value, true, nil
}

func optionalStringish(args map[string]any, key string) (string, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", false, nil
	}
	switch value := raw.(type) {
	case string:
		if value == "" {
			return "", false, errorsx.NewInvalidArgument("'%s' must be a non-empty string", key)
		}
		return value, true, nil
	case float64:
		return fmt.Sprintf("%.0f", value), true, nil
	default:
		return "", false, errorsx.NewInvalidArgument("'%s' must be a string or integer", key)
	}
}

func requiredStringList(args map[string]any, key string) ([]string, error) {
	value, ok, err := optionalStringList(args, key)
	if err != nil {
		return nil, err
	}
	if !ok || len(value) == 0 {
		return nil, errorsx.NewInvalidArgument("'%s' must be a non-empty string array", key)
	}
	return value, nil
}

func optionalStringList(args map[string]any, key string) ([]string, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, false, nil
	}
	items, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]string); ok {
			return append([]string(nil), typed...), true, nil
		}
		return nil, false, errorsx.NewInvalidArgument("'%s' must be an array of strings", key)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok || value == "" {
			return nil, false, errorsx.NewInvalidArgument("'%s' entries must be non-empty strings", key)
		}
		out = append(out, value)
	}
	return out, true, nil
}

func requiredObject(args map[string]any, key string) (map[string]any, error) {
	value, err := optionalObjectValue(args[key], key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errorsx.NewInvalidArgument("'%s' must be a non-empty object", key)
	}
	return value, nil
}

func optionalObject(args map[string]any, key string) (map[string]any, error) {
	return optionalObjectValue(args[key], key)
}

func optionalObjectValue(raw any, key string) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return nil, errorsx.NewInvalidArgument("'%s' must be an object", key)
	}
	if len(value) == 0 {
		return nil, errorsx.NewInvalidArgument("'%s' must be a non-empty object", key)
	}
	return value, nil
}

func optionalInt(args map[string]any, key string) (int, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return 0, false, nil
	}
	switch value := raw.(type) {
	case float64:
		return int(value), true, nil
	case int:
		return value, true, nil
	default:
		return 0, false, errorsx.NewInvalidArgument("'%s' must be an integer", key)
	}
}

func resolveMembershipMethod(args map[string]any) (string, error) {
	if action, ok, err := optionalString(args, "action"); err != nil {
		return "", err
	} else if ok {
		switch action {
		case "replace":
			return "PUT", nil
		case "add":
			return "POST", nil
		case "remove":
			return "DELETE", nil
		default:
			return "", errorsx.NewInvalidArgument("unsupported action %q; expected replace, add, or remove", action)
		}
	}
	if method, ok, err := optionalString(args, "method"); err != nil {
		return "", err
	} else if ok {
		switch method {
		case "GET", "PUT", "POST", "DELETE":
			return method, nil
		default:
			return "", errorsx.NewInvalidArgument("unsupported method %q", method)
		}
	}
	return "PUT", nil
}

func extractMembers(args map[string]any, payload map[string]any) ([]string, error) {
	if member, ok, err := optionalString(args, "member"); err != nil {
		return nil, err
	} else if ok {
		return []string{member}, nil
	}
	if members, ok, err := optionalStringList(args, "members"); err != nil {
		return nil, err
	} else if ok {
		return members, nil
	}
	if payload != nil {
		if member, ok := payload["id"].(string); ok && member != "" {
			return []string{member}, nil
		}
		if membersRaw, ok := payload["ids"].([]any); ok {
			members := make([]string, 0, len(membersRaw))
			for _, item := range membersRaw {
				value, ok := item.(string)
				if !ok || value == "" {
					return nil, errorsx.NewInvalidArgument("'payload.ids' entries must be non-empty strings")
				}
				members = append(members, value)
			}
			return members, nil
		}
		if membersRaw, ok := payload["ids"].([]string); ok {
			return append([]string(nil), membersRaw...), nil
		}
	}
	return nil, nil
}

func resolveServices(args map[string]any, defaults []string) ([]string, error) {
	if service, ok, err := optionalString(args, "service"); err != nil {
		return nil, err
	} else if ok {
		return []string{service}, nil
	}
	if services, ok, err := optionalStringList(args, "services"); err != nil {
		return nil, err
	} else if ok {
		return services, nil
	}
	return slices.Clone(defaults), nil
}
