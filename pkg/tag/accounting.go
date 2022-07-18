package tag

const (
	// AccountingProductID tag to store an accounting product ID
	AccountingProductID = "accounting.metal-stack.io/productid"
	// AccountingContractID tag to store an accounting contract ID
	AccountingContractID = "accounting.metal-stack.io/contractid"
	// AccountingDebtor tag to store an accounting debtor
	AccountingDebtor = "accounting.metal-stack.io/debtor"
	// AccountingNetworkTrafficExternal tag to indicate external network traffic
	AccountingNetworkTrafficExternal = "accounting.metal-stack.io/network-traffic-external"
	// AccountingNetworkTrafficInternal tag to indicate internal network traffic
	AccountingNetworkTrafficInternal = "accounting.metal-stack.io/network-traffic-internal"
	// AccountingVolumeReplicas tag to store accounting volume replicas
	AccountingVolumeReplicas = "accounting.metal-stack.io/volume-replicas"
	// AccountingVolumeEncryption tag to store accounting volume encryption information
	AccountingVolumeEncryption = "accounting.metal-stack.io/volume-encryption"
)

// AccountingTags returns all accounting tags
func AccountingTags() map[string]bool {
	return map[string]bool{
		AccountingProductID:              true,
		AccountingContractID:             true,
		AccountingDebtor:                 true,
		AccountingNetworkTrafficExternal: true,
		AccountingNetworkTrafficInternal: true,
		AccountingVolumeReplicas:         true,
		AccountingVolumeEncryption:       true,
	}
}
