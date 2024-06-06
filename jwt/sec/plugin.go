package sec

import (
	"errors"
	"strings"

	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/security"
)

const OidcDirectory = "oidc.metal-stack.io/directory"

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
// All groups without or with another tenant-prefix are filtered.
func (p *Plugin) GenericOIDCExtractUserProcessGroups(ic *security.IssuerConfig, claims *security.GenericOIDCClaims) (user *security.User, err error) {
	if ic == nil {
		return nil, errors.New("issuerConfig must not be nil")
	}
	return genericOidcExtractUser(ic, claims, p.extractAndProcessGroups)
}

// extractUser returns the User, groups are extracted with the given fn.
func genericOidcExtractUser(ic *security.IssuerConfig, claims *security.GenericOIDCClaims, fn extractGroupsFn) (user *security.User, err error) {

	var directory string
	if ic.Annotations != nil {
		directory = ic.Annotations[OidcDirectory]
	}
	grps, err := fn(ic.Tenant, directory, claims.Roles)
	if err != nil {
		return nil, err
	}

	usr := security.User{
		Name:   claims.Username(),
		EMail:  claims.EMail,
		Groups: grps,
		Tenant: ic.Tenant,
	}
	return &usr, nil
}

// ExtractUserProcessGroups is a implementation of security-extensionpoint
// Groups will reformatted [app]-[]-[]-[role], e.g. "maas-all-all-admin", "kaas-all-all-kaasadmin", "k8s-all-all-admin".
// All groups without or with another the tenant-prefix are filtered.
func (p *Plugin) ExtractUserProcessGroups(claims *security.Claims) (user *security.User, err error) {
	return extractUser(claims, p.extractAndProcessGroups)
}

// extractUser returns the User, groups are extracted with the given fn.
func extractUser(claims *security.Claims, fn extractGroupsFn) (user *security.User, err error) {

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
	// determine if the user is the operator/provider of the service
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
		if grpCtx.OnBehalfTenant != "" && !tenantIsProvider {
			continue
		}

		// full groupname, without tenant-prefix
		fullGroupname := grpCtx.ToFullGroupString()
		grps = append(grps, security.ResourceAccess(fullGroupname))
	}

	return grps, nil
}

// NewGroupExpression creates a new GroupExpression with the given values and ensures that they are properly encoded (i.e. '-' are replaced by '$')
func (p *Plugin) NewGroupExpression(appPrefix, firstScope, secondScope, role string) grp.GroupExpression {
	return grp.GroupExpression{
		AppPrefix:   p.grpr.GroupEncodeName(appPrefix),
		FirstScope:  p.grpr.GroupEncodeName(firstScope),
		SecondScope: p.grpr.GroupEncodeName(secondScope),
		Role:        p.grpr.GroupEncodeName(role),
	}
}

// HasGroupExpression checks if the given user has group permissions that fulfil the group-expression
// which supports "*" as wildcards for resourceTenant and groupExpression
func (p *Plugin) HasGroupExpression(user *security.User, resourceTenant string, groupExpression grp.GroupExpression) bool {

	// no resource tenant is not ok, there can be no default on this layer
	if resourceTenant == "" {
		return false
	}

	// what we have now is the slice of groups that the user has
	// (including "on behalf", with concrete cluster-tenant or wildcard "all")
	// "on behalf"-groups do not have cluster-tenant because it is already evaluated for the concrete tenant to act

	for i := range user.Groups {
		grpCtx, err := p.grpr.ParseGroupName(string(user.Groups[i]))
		if err != nil {
			continue
		}

		// check if group matches for any of the tenants
		if resourceTenant == grp.Any {
			if groupExpression.Matches(*grpCtx) {
				return true
			}
			continue
		}
		// resource belongs to own tenant
		if strings.EqualFold(user.Tenant, resourceTenant) && grpCtx.OnBehalfTenant == "" {
			if groupExpression.Matches(*grpCtx) {
				return true
			}
			continue
		}
		// resource belongs to other tenant, access "on behalf": if group is for resource-tenant or for "all" then check
		if strings.EqualFold(grpCtx.OnBehalfTenant, resourceTenant) || grpCtx.OnBehalfTenant == grp.All {
			if groupExpression.Matches(*grpCtx) {
				return true
			}
			continue
		}
	}

	return false

}

// GroupsOnBehalf returns the list of groups that the user can do an behalf of the other tenant.
// The groups returned are canonical groups without tenant prefix and cluster-tenant, e.g. "kaas-all-all-admin".
func (p *Plugin) GroupsOnBehalf(u *security.User, tenant string) []security.ResourceAccess {

	var result []security.ResourceAccess
	for i := range u.Groups {
		grpCtx, err := p.grpr.ParseGroupName(string(u.Groups[i]))
		if err == nil && grpCtx.OnBehalfTenant == tenant {
			// returns groupname without cluster tenant
			result = append(result, security.ResourceAccess(grpCtx.ToCanonicalGroupString()))
		}
	}

	return result
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
			nsok := requestedGroupCtx.SecondScope == grpCtx.SecondScope || grpCtx.SecondScope == grp.All
			clusterok := requestedGroupCtx.FirstScope == grpCtx.FirstScope || grpCtx.FirstScope == grp.All

			if roleok && nsok && clusterok {

				switch grpCtx.OnBehalfTenant {
				case grp.All:
					// return with all==true
					return []string{}, true, nil
				case "":
					tenants[user.Tenant] = true
				default:
					tenants[grpCtx.OnBehalfTenant] = true
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
