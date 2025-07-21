package tag

const (
	// ClusterID tag to store the id of the cluster
	ClusterID = "cluster.metal-stack.io/id"
	// ClusterName tag to store the name of the cluster
	ClusterName = "cluster.metal-stack.io/name"
	// ClusterDescription tag to store the description of the cluster
	ClusterDescription = "cluster.metal-stack.io/description"
	// ClusterProject tag to store the project the cluster belongs to
	ClusterProject = "cluster.metal-stack.io/project"
	// ClusterPartition tag to store the partition of the cluster
	ClusterPartition = "cluster.metal-stack.io/partition"
	// ClusterTenant tag to store the tenant of the cluster
	ClusterTenant = "cluster.metal-stack.io/tenant"
	// ClusterServiceFQN tag to identify a service running in the cluster
	ClusterServiceFQN = "cluster.metal-stack.io/id/namespace/service"
	// ClusterEgress tag to identify egress ips used for a cluster
	ClusterEgress = "cluster.metal-stack.io/id/egress"
	// ClusterOwner tag to store the name of the cluster owner
	ClusterOwner = "cluster.metal-stack.io/owner"
	// Partition tags to store the type of the cluster partition
	RegionalAutospread = "partition.metal-stack.io/type=regional-autospread"
	MultiZonal         = "partition.metal-stack.io/type=multi-zonal"
	SingleZone         = "partition.metal-stack.io/type=single-zone"
)
