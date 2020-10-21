package net

const (
	// PrivatePrimaryUnshared is a network which is for machines which is private
	PrivatePrimaryUnshared = "privateprimaryunshared"
	// PrivatePrimaryShared is a network which is for machines which is private and shared for other networks
	PrivatePrimaryShared = "privateprimaryshared"
	// PrivateSecondaryShared is a network which is for machines which is consumed from a other shared network
	PrivateSecondaryShared = "privatesecondaryshared"
	// PrivateSecondaryUnshared is not supported
	PrivateSecondaryUnshared = "privatesecondaryunshared"
	// External is a network which is external to machines
	External = "external"
	// Underlay is a network for the dataplane
	Underlay = "underlay"
)
