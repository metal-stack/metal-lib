package grp

import (
	"fmt"
	"strings"
)

// parses the connectorId, convention is "[tenant]_[directory]"
// optionally there can be arbitrary additional fields that are ignored
func ParseConnectorId(connectorId string) (jwtTenant string, directory string, err error) {

	// the tenant is the first part of the connectorId, i.e. "ddd_ldap" -> "ddd"
	connectorParts := strings.Split(connectorId, "_")
	if len(connectorParts) >= 2 {
		jwtTenant = connectorParts[0]
		directory = connectorParts[1]

		return
	}

	return "", "", fmt.Errorf("error parsing connectorId, expected [tenant]_[directory type], got %s", connectorId)
}
