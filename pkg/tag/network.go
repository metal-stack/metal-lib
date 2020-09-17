package tag

const (
	// NetworkDefault indicates a network that can serve as a default network for cluster creation
	// there should only be one default network in a metal control plane, otherwise behavior will be non-deterministic
	NetworkDefault = "network.metal-stack.io/default"
	// NetworkDefaultExternal indicates a network that can serve as a default for IP allocations
	NetworkDefaultExternal = "network.metal-stack.io/default-external"
)
