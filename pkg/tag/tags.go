package tag

import (
	"fmt"
	"strings"
)

const (
	clusterPrefix = "cluster.metal-pod.io" //TODO replace with "cluster.metal-stack.io"
	machinePrefix = "machine.metal-pod.io" //TODO replace with "machine.metal-stack.io"
)

const (
	// ClusterQualifier identifies the cluster
	ClusterQualifier = "clusterid"

	// ServiceQualifier identifies the service
	ServiceQualifier = "service"

	// NamespaceQualifier identifies the namespace
	NamespaceQualifier = "namespace"

	// ClusterPrefix the prefix of the tag used to identify a cluster
	ClusterPrefix = clusterPrefix + "/" + ClusterQualifier //TODO either remove clusterPrefix or ClusterPrefix

	// ServicePrefix the prefix of the tag used to identify services
	ServicePrefix = ClusterPrefix + "/" + NamespaceQualifier + "/" + ServiceQualifier

	// MachineQualifier identifies the machine
	MachineQualifier = "machineid"

	// MachinePrefix the prefix of the tag used to identify a machine
	MachinePrefix = machinePrefix + "/" + MachineQualifier   //TODO either machinePrefix or MachinePrefix
	MetalPrefix   = "metal.metal-pod.io/" + MachineQualifier //TODO remove entirely. Use MachinePrefix instead

	// PrimaryNetworkASN the primary network ASN tag
	PrimaryNetworkASN = machinePrefix + "/" + "network.primary.asn"

	ClusterProject     = clusterPrefix + "/project"
	ClusterDescription = clusterPrefix + "/description"
	ClusterID          = clusterPrefix + "/id"
	ClusterName        = clusterPrefix + "/name"
	ClusterTenant      = clusterPrefix + "/tenant"

	MachineProjectID = machinePrefix + "/project-id" //TODO remove entirely. Use ClusterProject instead
)

// ServiceTag constructs the service tag for the given cluster and service
func ServiceTag(clusterID string, namespace, serviceName string) string {
	return fmt.Sprintf("%s/%s/%s", ServiceTagClusterPrefix(clusterID), namespace, serviceName)
}

// ServiceTagClusterPrefix constructs the prefix of the service tag that identify all services of a cluster
func ServiceTagClusterPrefix(clusterID string) string {
	return fmt.Sprintf("%s=%s", ServicePrefix, clusterID)
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
