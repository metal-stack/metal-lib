package grp

import (
	"errors"
	"fmt"
)

// Group with Context (Tenant)
type GroupContext struct {
	// tenant of the group (example: tnnt of group tnns_all_all_admin)
	TenantPrefix string

	// group
	Group
}

// Group information
type Group struct {
	// Application
	AppPrefix string
	// Tenant of the cluster, if is not the same as the tenant prefix (example: ddd of group tnnt_ddd#dev-all-admin)
	ClusterTenant string
	// name of the cluster
	ClusterName string
	// namespace in the cluster
	Namespace string
	// role in the given context
	Role string
}

// NewGroup creates the Group with the given content.
// Clustername and Namespace will be groupname-encoded.
func (g *Grpr) NewGroup(app, clusterTenant, cluster, namespace, role string) *Group {
	return &Group{
		AppPrefix:     app,
		ClusterTenant: clusterTenant,
		ClusterName:   g.GroupEncodeName(cluster),
		Namespace:     g.GroupEncodeName(namespace),
		Role:          role,
	}
}

// ToFullGroupString returns formatted group [app]-[opt. clustertenant][clustername]-[namespace]-[role]
func (g *Group) ToFullGroupString() string {

	cluster := g.ClusterName
	if g.ClusterTenant != "" {
		cluster = fmt.Sprintf("%s%s%s", g.ClusterTenant, tenantClusterNameSeparator, g.ClusterName)
	}

	return fmt.Sprintf("%s%s%s%s%s%s%s", g.AppPrefix, innerGroupPartSeparator, cluster, innerGroupPartSeparator, g.Namespace, innerGroupPartSeparator, g.Role)
}

// returns formatted group [prefix][namespace]-[role]
func (g *Group) ToPrefixedGroupString(prefix string) string {

	return fmt.Sprintf("%s%s%s%s", prefix, g.Namespace, innerGroupPartSeparator, g.Role)
}

// ToCanonicalGroupString returns formatted group [app]-[clustername]-[namespace]-[role], the clusterTenant is left out!
func (g *Group) ToCanonicalGroupString() string {
	return fmt.Sprintf("%s%s%s%s%s%s%s", g.AppPrefix, innerGroupPartSeparator, g.ClusterName, innerGroupPartSeparator, g.Namespace, innerGroupPartSeparator, g.Role)
}

var errInvalidFormat = errors.New("invalid group-format")
