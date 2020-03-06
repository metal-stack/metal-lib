package tag

import (
	"fmt"
	"strings"
)

const (
	// MachineProject tag to store the project where the machine belongs to
	MachineProject = "machine.metal-stack.io/project"
	// MachineDescription tag to store machine description
	MachineDescription = "machine.metal-stack.io/description"
	// MachineID tag to store machine ID
	MachineID = "machine.metal-stack.io/id"
	// MachineName tag to store machine name
	MachineName = "machine.metal-stack.io/name"
	// MachineTenant tag to store the tenant the machine belongs to
	MachineTenant = "machine.metal-stack.io/tenant"
	// MachineNetworkPrimaryASN tag to store the primary BGP ASN the machine announces.
	MachineNetworkPrimaryASN = "machine.metal-stack.io/network.primary.asn"

	// ClusterProject tag to store the project the cluster belongs to
	ClusterProject = "cluster.metal-stack.io/project"
	// ClusterDescription tag to store the description of the cluster
	ClusterDescription = "cluster.metal-stack.io/description"
	// ClusterID tag to store the id of the cluster
	ClusterID = "cluster.metal-stack.io/id"
	// ClusterName tag to store the name of the cluster
	ClusterName = "cluster.metal-stack.io/name"
	// ClusterTenant tag to store the tenant of the cluster
	ClusterTenant = "cluster.metal-stack.io/tenant"
	// ClusterServiceFQN the prefix of the tag used to identify services
	ClusterServiceFQN = "cluster.metal-stack.io/" + IDQualifier + "/" + NamespaceQualifier + "/" + ServiceQualifier

	// IDQualifier identifies the cluster or machine
	IDQualifier = "id"

	// ServiceQualifier identifies the service
	ServiceQualifier = "service"

	// NamespaceQualifier identifies the namespace
	NamespaceQualifier = "namespace"
)

// ServiceTag constructs the service tag for the given cluster and service
func ServiceTag(clusterID string, namespace, serviceName string) string {
	return fmt.Sprintf("%s/%s/%s", ServiceTagClusterPrefix(clusterID), namespace, serviceName)
}

// ServiceTagClusterPrefix constructs the prefix of the service tag that identify all services of a cluster
func ServiceTagClusterPrefix(clusterID string) string {
	return fmt.Sprintf("%s=%s", ClusterServiceFQN, clusterID)
}

// IsMachine returns true if the given tag is a machine-tag.
func IsMachine(tag string) bool {
	return strings.HasPrefix(tag, MachinePrefix)
}

// IsMemberOfCluster returns true of the given tag is a cluster-tag and clusterID matches.
func IsMemberOfCluster(tag, clusterID string) bool {
	if strings.HasPrefix(tag, ClusterPrefix) {
		parts := strings.Split(tag, "=")
		if len(parts) != 2 {
			return false
		}
		if strings.HasPrefix(parts[1], clusterID) {
			return true
		}
	}
	return false
}
