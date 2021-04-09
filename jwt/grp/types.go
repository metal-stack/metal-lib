package grp

import (
	"errors"
	"fmt"
)

// Group with Context (Tenant)
type GroupContext struct {
	// tenant of the group (example: tnnt of group tnns-all-all-admin)
	TenantPrefix string

	// group
	Group
}

// Group information
type Group struct {
	// Application
	AppPrefix string
	// "On behalf"-Tenant, if is not the same as the tenant prefix (example: ddd of group tnnt_ddd#dev-all-admin)
	OnBehalfTenant string
	// first scope: for kaas name of the project, for k8s name of the cluster
	FirstScope string
	// second scope: for kaas name of the cluster, for k8s namespace in the cluster
	SecondScope string
	// role in the given context
	Role string
}

// NewGroup creates the Group with the given content.
// FirstScope and SecondScope will be groupname-encoded.
func (g *Grpr) NewGroup(app, onBehalfTenant, firstScope, secondScope, role string) *Group {
	return &Group{
		AppPrefix:      app,
		OnBehalfTenant: onBehalfTenant,
		FirstScope:     g.GroupEncodeName(firstScope),
		SecondScope:    g.GroupEncodeName(secondScope),
		Role:           role,
	}
}

// ToFullGroupString returns formatted group [app]-[opt. onBehalfTenant][firstScope]-[secondScope]-[role]
func (g *Group) ToFullGroupString() string {

	firstScope := g.FirstScope
	if g.OnBehalfTenant != "" {
		firstScope = fmt.Sprintf("%s%s%s", g.OnBehalfTenant, onBehalfAndScopeSeparator, g.FirstScope)
	}

	return fmt.Sprintf("%s%s%s%s%s%s%s", g.AppPrefix, innerGroupPartSeparator, firstScope, innerGroupPartSeparator, g.SecondScope, innerGroupPartSeparator, g.Role)
}

// returns formatted group [prefix][secondScope]-[role]
func (g *Group) ToPrefixedGroupString(prefix string) string {

	return fmt.Sprintf("%s%s%s%s", prefix, g.SecondScope, innerGroupPartSeparator, g.Role)
}

// ToCanonicalGroupString returns formatted group [app]-[firstScope]-[secondScope]-[role], the onBehalfTenant is left out!
func (g *Group) ToCanonicalGroupString() string {
	return fmt.Sprintf("%s%s%s%s%s%s%s", g.AppPrefix, innerGroupPartSeparator, g.FirstScope, innerGroupPartSeparator, g.SecondScope, innerGroupPartSeparator, g.Role)
}

var errInvalidFormat = errors.New("invalid group-format")
