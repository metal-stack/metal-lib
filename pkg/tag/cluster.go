package tag

const (
	// ClusterID tag to store the id of the cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterID
	ClusterID = "cluster.metal-stack.io/id"
	// ClusterName tag to store the name of the cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterName
	ClusterName = "cluster.metal-stack.io/name"
	// ClusterDescription tag to store the description of the cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterDescription
	ClusterDescription = "cluster.metal-stack.io/description"
	// ClusterProject tag to store the project the cluster belongs to
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterProject
	ClusterProject = "cluster.metal-stack.io/project"
	// ClusterPartition tag to store the partition of the cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterPartition
	ClusterPartition = "cluster.metal-stack.io/partition"
	// ClusterTenant tag to store the tenant of the cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterTenant
	ClusterTenant = "cluster.metal-stack.io/tenant"
	// ClusterServiceFQN tag to identify a service running in the cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterServiceFQN
	ClusterServiceFQN = "cluster.metal-stack.io/id/namespace/service"
	// ClusterEgress tag to identify egress ips used for a cluster
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterEgress
	ClusterEgress = "cluster.metal-stack.io/id/egress"
	// ClusterOwner tag to store the name of the cluster owner
	// Deprecated use github.com/metal-stack/api/go/tag/ClusterOwner
	ClusterOwner = "cluster.metal-stack.io/owner"
)
