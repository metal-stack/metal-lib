package tag

const (
	// Partition tags to store the type of the cluster partition

	// SingleZone describes a partition that is located in a single zone inside a region.
	SingleZone         = "partition.metal-stack.io/type=single-zone"
	
	// RegionalAutospread describes a partition, which spreads machines across a region using the metal-stack rack spreading feature.
	RegionalAutospread = "partition.metal-stack.io/type=regional-autospread"
)
