package tag

const (
	// PartitionType describes a type of a partition, i.e. a partition that is located in a single zone inside a region or a partition which spreads machines across a region using the metal-stack rack spreading feature.
	PartitionType = "partition.metal-stack.io/type"

	// PartitionTypeSingleZone describes a partition that is located in a single zone inside a region.
	PartitionTypeSingleZone         = "single-zone"

	// PartitionTypeRegionalAutospread describes a partition, which spreads machines across a region using the metal-stack rack spreading feature.
	PartitionTypeRegionalAutospread = "regional-autospread"
)
