package grp

import (
	"errors"
	"fmt"
	"strings"
)

/*
	Naming conventions for groups:

	ActiveDirectory: 	TnPg_Srv_Appkaas-clustername-namespace-role_full

	TenantPrefix:   	Tn = Tenant-Prefix
	GroupType:			Pg = PermissionGroup
	SecondLevelOU:		Srv
	Reference:			App (App-Permission)
	innerGroupName:		kaas-clustername-namespace-role
	Permission:			Full | Mod | Read

	UNIX-LDAP:			tnnt_kaas-clustername-namespace-role
	TenantPrefix:   	tnnt
	innerGroupName:		kaas-clustername-namespace-role
*/
const (
	// separator for outer parts in a group: TnPg_Srv_[inner group]_full
	outerGroupPartSeparator = "_"

	// separator for inner parts in a group: Appkaas-cluster-namespace-role
	innerGroupPartSeparator = "-"

	// separator within the clustername part: k8s-ddd#cluster-namespace-role
	onBehalfAndScopeSeparator = "#"

	// ReferencePrefix "App"
	adReferencePrefix = "App"

	// defined names of the directories in token "connector_id"
	directoryTypeAD   = "ad"
	directoryTypeLDAP = "ldap"

	// "wildcard" for allowing all variants
	All = "all"
)

// Grpr encapsulates conversion from and to groups.
type Grpr struct {
	config Config
}

type Config struct {
	// tenant-prefixes are dependant on directory-environment
	ProviderTenant string
}

// Init configures the Grpr
func NewGrpr(cfg Config) (*Grpr, error) {
	if cfg.ProviderTenant == "" {
		return nil, errors.New("providerTenant must be specified")
	}
	return &Grpr{config: cfg}, nil
}

// Init configures the Grpr and panics if an error occurs.
func MustNewGrpr(cfg Config) *Grpr {
	grp, err := NewGrpr(cfg)
	if err != nil {
		panic(err)
	}
	return grp
}

// common signature for the GroupContext parsing funcs
type GroupContextParseFunc func(group string) (*GroupContext, error)

// SelectGroupParseFunc selects the parsing func according to the given directoryType, see constants
func (g *Grpr) SelectGroupParseFunc(directoryType string) (GroupContextParseFunc, error) {
	switch strings.ToLower(directoryType) {
	case directoryTypeAD:
		return g.ParseADGroup, nil
	case directoryTypeLDAP:
		return g.ParseUnixLDAPGroup, nil
	default:
		return nil, fmt.Errorf("invalid directoryType %s", directoryType)
	}
}

// IsProviderTenant returns true, if the given tenant is the provider/operator of the service
// i.e. "tnnt" or "Tn" in our case
func (g *Grpr) IsProviderTenant(tenant string, directoryType string) (bool, error) {

	switch strings.ToLower(directoryType) {
	case directoryTypeAD:
		return tenant == g.config.ProviderTenant, nil
	case directoryTypeLDAP:
		return tenant == g.config.ProviderTenant, nil
	default:
		return false, fmt.Errorf("invalid directoryType %s", directoryType)
	}
}

// Parse parses and structurally validates a group.
// The result contains normalized (toLower) results.
// TnPg_Srv_Appkaas-cluster-namespace-role_full
func (g *Grpr) ParseADGroup(groupname string) (*GroupContext, error) {

	groupname = strings.ToLower(groupname)

	outerSplit := strings.Split(groupname, outerGroupPartSeparator)

	if len(outerSplit) != 4 {
		return nil, errInvalidFormat
	}

	prefix := outerSplit[0]
	if len(prefix) != 4 {
		return nil, errInvalidFormat
	}

	tenantPrefix := prefix[:2]

	// outerSplit[1] = Srv, irrelevant

	referencePrefixedInnerGroupname := outerSplit[2]

	// outerSplit[3] = full, irrelevant

	// remove Reference to get inner groupname
	innerGroupname := referencePrefixedInnerGroupname[len(adReferencePrefix):]

	group, err := g.ParseGroupName(innerGroupname)
	if err != nil {
		return nil, err
	}

	return &GroupContext{
		TenantPrefix: tenantPrefix,
		Group:        *group,
	}, nil
}

// Parse parses and structurally validates a group.
// The result contains normalized (toLower) results.
// tnnt_kaas-clustername-namespace-role
func (g *Grpr) ParseUnixLDAPGroup(groupname string) (*GroupContext, error) {

	groupname = strings.ToLower(groupname)

	outerSplit := strings.Split(groupname, outerGroupPartSeparator)

	if len(outerSplit) != 2 {
		return nil, errInvalidFormat
	}

	tenantPrefix := outerSplit[0]
	innerGroupname := outerSplit[1]

	group, err := g.ParseGroupName(innerGroupname)
	if err != nil {
		return nil, err
	}

	return &GroupContext{
		TenantPrefix: tenantPrefix,
		Group:        *group,
	}, nil
}

// parses the "inner" groupname with stripped tenant prefixes and idm-suffixes
// example kaas-clustername-namespace-role
func (g *Grpr) ParseGroupName(groupname string) (*Group, error) {

	innerSplit := strings.Split(groupname, innerGroupPartSeparator)
	if len(innerSplit) != 4 {
		return nil, errInvalidFormat
	}

	var clusterTenant string
	clusterName := innerSplit[1]
	if strings.Contains(clusterName, onBehalfAndScopeSeparator) {
		cn := strings.Split(clusterName, onBehalfAndScopeSeparator)

		clusterTenant = cn[0]
		clusterName = cn[1]
	}

	group := &Group{
		AppPrefix:      innerSplit[0],
		OnBehalfTenant: clusterTenant,
		FirstScope:     clusterName,
		SecondScope:    innerSplit[2],
		Role:           innerSplit[3],
	}

	return group, nil
}

// encodes the name so that it can be used in groups, i.e. "-" are replaced by "$"
func (g *Grpr) GroupEncodeName(name string) string {
	return strings.ReplaceAll(name, "-", "$")
}

// encodes the names so that it can be used in groups, i.e. "-" are replaced by "$"
func (g *Grpr) GroupEncodeNames(names []string) []string {
	var result []string
	for i := range names {
		result = append(result, strings.ReplaceAll(names[i], "-", "$"))
	}
	return result
}
