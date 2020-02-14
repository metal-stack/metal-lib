package sec

import (
	"errors"
	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/security"
	"strings"
)

type Plugin struct {
	grpr *grp.Grpr
}

func NewPlugin(grpr *grp.Grpr) *Plugin {
	return &Plugin{
		grpr: grpr,
	}
}

// ExtractUserProcessGroups is a implementation of security-extensionpoint
// Groups will reformatted [app]-[]-[]-[role], e.g. "maas-all-all-admin", "kaas-all-all-kaasadmin", "k8s-all-all-admin".
// All groups without or with another the tenant-prefix are filtered.
func (p *Plugin) ExtractUserProcessGroups(claims *security.Claims) (user *security.User, err error) {
	return p.extractUser(claims, p.extractAndProcessGroups)
}

// extractUser returns the User, groups are extracted with the given fn.
func (p *Plugin) extractUser(claims *security.Claims, fn extractGroupsFn) (user *security.User, err error) {

	tenant := ""
	if claims.FederatedClaims == nil {
		return nil, errors.New("Invalid Token, no FederatedClaims")
	}
	cid := claims.FederatedClaims["connector_id"]
	if cid == "" {
		return nil, errors.New("Invalid Token, no connector_id")
	}

	directory := ""
	tenant, directory, err = grp.ParseConnectorId(cid)
	if err != nil {
		return nil, err
	}

	grps, err := fn(tenant, directory, claims.Groups)
	if err != nil {
		return nil, err
	}

	usr := security.User{
		Name:   claims.Name,
		EMail:  claims.EMail,
		Groups: grps,
		Tenant: tenant,
	}
	return &usr, nil
}

// extractGroupsFn declaration for functions that extract groups
type extractGroupsFn func(tenant string, directory string, groups []string) ([]security.ResourceAccess, error)

// extractAndProcessGroups is a implementation of the extractGroupsFn for ExtractUserProcessGroups
func (p *Plugin) extractAndProcessGroups(tenant string, directory string, groups []string) ([]security.ResourceAccess, error) {
	// determine if the user is the operator/provider of the service (fi-ts)
	tenantIsProvider, err := p.grpr.IsProviderTenant(tenant, directory)
	if err != nil {
		return nil, err
	}

	var groupParseFn grp.GroupContextParseFunc
	groupParseFn, err = p.grpr.SelectGroupParseFunc(directory)
	if err != nil {
		return nil, err
	}

	var grps []security.ResourceAccess
	for _, g := range groups {
		// parsed info from group, all lowercased
		grpCtx, err := groupParseFn(g)
		if err != nil {
			continue
		}

		// group is not of tenant -> skip!
		if strings.ToLower(tenant) != grpCtx.TenantPrefix {
			continue
		}
		// skip on behalf group if user tenant is not provider tenant
		if grpCtx.ClusterTenant != "" && !tenantIsProvider {
			continue
		}

		// full groupname, without tenant-prefix
		fullGroupname := grpCtx.ToFullGroupString()
		grps = append(grps, security.ResourceAccess(fullGroupname))
	}

	return grps, nil
}

// UserTenantGroups returns the list of user-groups that the user can do for his tenant.
func (p *Plugin) UserTenantGroups(u *security.User) []security.ResourceAccess {

	var result []security.ResourceAccess
	for i := range u.Groups {
		grpCtx, err := p.grpr.ParseGroupName(string(u.Groups[i]))
		if err == nil && grpCtx.ClusterTenant == "" {
			// returns groupname without cluster tenant
			result = append(result, security.ResourceAccess(grpCtx.ToCanonicalGroupString()))
		}
	}

	return result
}

// GroupsOnBehalf returns the list of groups that the user can do an behalf of the other tenant.
// The groups returned are canonical groups without tenant prefix and cluster-tenant, e.g. "kaas-all-all-admin".
func (p *Plugin) GroupsOnBehalf(u *security.User, tenant string) []security.ResourceAccess {

	var result []security.ResourceAccess
	for i := range u.Groups {
		grpCtx, err := p.grpr.ParseGroupName(string(u.Groups[i]))
		if err == nil && grpCtx.ClusterTenant == tenant {
			// returns groupname without cluster tenant
			result = append(result, security.ResourceAccess(grpCtx.ToCanonicalGroupString()))
		}
	}

	return result
}

// HasOneOfGroups returns, if the given user has one of the the given groups for/"on behalf of" the given tenant.
// The groups to check are canonical groups without tenant prefix, e.g. "kaas-all-all-admin".
// The matches are exact matches, so "kaas-all-all-admin" only matches "kaas-all-all-admin",
// see HasGroupExpression for more flexible queries
func (p *Plugin) HasOneOfGroups(user *security.User, tenant string, groups ...security.ResourceAccess) bool {

	// default to tenant from token
	if tenant == "" {
		tenant = user.Tenant
	}

	// per default, check the user groups for HIS tenant
	userGroups := p.UserTenantGroups(user)

	// if this is a "on behalf"-scenario, then we use the "on behalf"-groups for the tenant
	if tenant != user.Tenant {
		// user wants to act "on behalf" of another tenant, use the groups that allow access to other tenant
		userGroups = p.GroupsOnBehalf(user, tenant)
	}

	acc := accessGroup(userGroups).asSet()
	for _, cgrp := range groups {
		if ok := acc[cgrp]; ok {
			return true
		}
	}
	return false
}

// HasGroupExpression checks if the given user has group permissions that fulfil the group-expression
// which supports "*" as wildcards
func (p *Plugin) HasGroupExpression(user *security.User, tenant string, groupExpression grp.GroupExpression) bool {

	// default to tenant from token
	if tenant == "" {
		tenant = user.Tenant
	}

	// per default, check the user groups for HIS tenant
	userGroups := p.UserTenantGroups(user)

	// if this is a "on behalf"-scenario, then we use the "on behalf"-groups for the tenant
	if tenant != user.Tenant {
		// user wants to act "on behalf" of another tenant, use the groups that allow access to other tenant
		userGroups = p.GroupsOnBehalf(user, tenant)
	}

	// what we have now is the slice of groups that the user has
	// (including "on behalf", with concrete cluster-tenant or wildcard "all")
	// "on behalf"-groups do not have cluster-tenant because it is already evaluated for the concrete tenant to act

	for _, ug := range userGroups {
		currentGroup, err := p.grpr.ParseGroupName(string(ug))
		if err != nil {
			continue
		}
		if groupExpression.Matches(*currentGroup) {
			return true
		}
	}

	return false

}

// TenantsOnBehalf returns the tenants, that the user can act on behalf with one of the given group-permissions.
// If the user is allowed to act on "all" tenants on behalf, only the flag "all" is true and no tenants are returned.
func (p *Plugin) TenantsOnBehalf(user *security.User, groups []security.ResourceAccess) ([]string, bool, error) {

	tenants := make(map[string]bool)
	for _, group := range groups {
		requestedGroupCtx, err := p.grpr.ParseGroupName(string(group))
		if err != nil {
			return nil, false, err
		}

		for i := range user.Groups {
			grpCtx, err := p.grpr.ParseGroupName(string(user.Groups[i]))
			if err != nil {
				continue
			}

			roleok := requestedGroupCtx.Role == grpCtx.Role
			nsok := requestedGroupCtx.Namespace == grpCtx.Namespace || grpCtx.Namespace == grp.All
			clusterok := requestedGroupCtx.ClusterName == grpCtx.ClusterName || grpCtx.ClusterName == grp.All

			if roleok && nsok && clusterok {

				switch grpCtx.ClusterTenant {
				case grp.All:
					// return with all==true
					return []string{}, true, nil
				case "":
					tenants[user.Tenant] = true
				default:
					tenants[grpCtx.ClusterTenant] = true
				}
			}
		}
	}

	return keys(tenants), false, nil
}

func keys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	return keys
}

type accessGroup []security.ResourceAccess
type resourceSet map[security.ResourceAccess]bool

func (ra accessGroup) asSet() resourceSet {
	groupset := make(resourceSet)
	for _, g := range ra {
		groupset[g] = true
	}
	return groupset
}

// ToResourceAccess creates a slice of ResourceAccess for the given groups
func ToResourceAccess(groups ...string) []security.ResourceAccess {
	var result []security.ResourceAccess
	for _, g := range groups {
		result = append(result, security.ResourceAccess(g))
	}
	return result
}

// MergeResourceAccess merges the given slices of ResourceAccess in a single one.
// Duplicates are not filtered.
func MergeResourceAccess(ras ...[]security.ResourceAccess) []security.ResourceAccess {
	var result []security.ResourceAccess
	for _, ra := range ras {
		result = append(result, ra...)
	}

	return result
}
