package tag

const (
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
	ClusterServiceFQN = "cluster.metal-stack.io/id/service/namespace"
)
