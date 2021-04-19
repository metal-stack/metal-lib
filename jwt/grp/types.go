package grp

import (
	"errors"
	"fmt"
)

// Group with Context (Tenant)
type GroupContext struct {
	// tenant of the group (example: tnnt of group tnnt_api-all-all-admin)
	TenantPrefix string

	// group
	Group
}

// Group information
type Group struct {
	// AppPrefix is id of the Application, e.g. kaas, k8s,... (example: 'app' for group 'app-ddd#dev-all-admin')
	AppPrefix string
	// OnBehalfTenant is the id of the tenant of the resource this group grants permissions on (example: 'ddd' for group 'app-ddd#dev-all-admin')
	OnBehalfTenant string
	// FirstScope e.g. for app kaas name of the project, for app k8s name of the cluster (example: 'dev' for group 'app-ddd#dev-all-admin')
	FirstScope string
	// SecondScope e.g. for app kaas name of the cluster, for app k8s namespace in the cluster (example: 'all' for group 'app-ddd#dev-all-admin')
	SecondScope string
	// Role is the in the given context (example: 'admin' for group 'app-ddd#dev-all-admin')
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
