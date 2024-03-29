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
	// AccountingVolumeQoSPolicyID tag to store accounting volume qos policy ID
	AccountingVolumeQoSPolicyID = "accounting.metal-stack.io/volume-qos-policy-id"
	// AccountingVolumeQoSPolicyName tag to store accounting volume qos policy name
	AccountingVolumeQoSPolicyName = "accounting.metal-stack.io/volume-qos-policy-name"
	// AccountingVolumeEncryption tag to store accounting volume encryption information
	AccountingVolumeEncryption = "accounting.metal-stack.io/volume-encryption"
	// AccountingVolumeSnapshotSource tag to store accounting volume snapshot source uuid
	AccountingVolumeSnapshotSource = "accounting.metal-stack.io/volume-snapshot-source"
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
		AccountingVolumeQoSPolicyID:      true,
		AccountingVolumeQoSPolicyName:    true,
		AccountingVolumeEncryption:       true,
		AccountingVolumeSnapshotSource:   true,
	}
}
