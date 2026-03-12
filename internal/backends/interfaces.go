package backends

import (
	"context"
	"net/url"
)

type SMDBackend interface {
	Configured() bool
	BaseURL() string
	Ready(context.Context) (any, error)
	ListComponents(context.Context) (any, error)
	GetComponent(context.Context, string) (any, error)
	ListGroups(context.Context) (any, error)
	GetGroup(context.Context, string) (any, error)
	GetMembership(context.Context, string) (any, error)
	CreateGroup(context.Context, map[string]any) (any, error)
	PatchGroup(context.Context, string, map[string]any) (any, error)
	DeleteGroup(context.Context, string) (any, error)
	GetGroupMembers(context.Context, string) (any, error)
	SetGroupMembers(context.Context, string, []string) (any, error)
	AddGroupMember(context.Context, string, string) (any, error)
	DeleteGroupMember(context.Context, string, string) (any, error)
}

type BSSBackend interface {
	Configured() bool
	BaseURL() string
	ServiceStatus(context.Context) (any, error)
	GetBootParameters(context.Context) (any, error)
	GetBootScript(context.Context, url.Values) (any, error)
	ListHosts(context.Context) (any, error)
	PutBootParameters(context.Context, map[string]any) (any, error)
	PostBootParameters(context.Context, map[string]any) (any, error)
	PatchBootParameters(context.Context, map[string]any) (any, error)
	DeleteBootParameters(context.Context, map[string]any) (any, error)
	PostHosts(context.Context, map[string]any) (any, error)
}

type CloudInitBackend interface {
	Configured() bool
	BaseURL() string
	Version(context.Context) (any, error)
	GetClusterDefaults(context.Context) (any, error)
	SetClusterDefaults(context.Context, map[string]any) (any, error)
	ListGroups(context.Context) (any, error)
	GetGroup(context.Context, string) (any, error)
	CreateGroup(context.Context, map[string]any) (any, error)
	UpdateGroup(context.Context, string, map[string]any) (any, error)
	DeleteGroup(context.Context, string) (any, error)
	UpdateInstanceInfo(context.Context, string, map[string]any) (any, error)
	GetMetaData(context.Context, string) (any, error)
	GetUserData(context.Context, string) (any, error)
	GetVendorData(context.Context, string) (any, error)
	GetGroupUserData(context.Context, string, string) (any, error)
}

type PowerControlBackend interface {
	Configured() bool
	BaseURL() string
	Health(context.Context) (any, error)
	GetPowerStatus(context.Context, map[string]any) (any, error)
	CreateTransition(context.Context, map[string]any) (any, error)
}

type KeaBackend interface {
	Configured() bool
	BaseURL() string
	DefaultServices() []string
	RunCommand(context.Context, string, map[string]any, []string) (any, error)
}

type CoreDHCPBackend interface {
	Configured() bool
	BaseURL() string
	Status(context.Context) (any, error)
}
