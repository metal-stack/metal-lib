package tag

const (
	// MachineID tag to store machine ID
	// Deprecated use github.com/metal-stack/go/tag/MachineID
	MachineID = "machine.metal-stack.io/id"
	// MachineName tag to store machine name
	// Deprecated use github.com/metal-stack/go/tag/MachineName
	MachineName = "machine.metal-stack.io/name"
	// MachineDescription tag to store machine description
	// Deprecated use github.com/metal-stack/go/tag/MachineDescription
	MachineDescription = "machine.metal-stack.io/description"
	// MachineProject tag to store the project where the machine belongs to
	// Deprecated use github.com/metal-stack/go/tag/MachineProject
	MachineProject = "machine.metal-stack.io/project"
	// MachineTenant tag to store the tenant the machine belongs to
	// Deprecated use github.com/metal-stack/go/tag/MachineTenant
	MachineTenant = "machine.metal-stack.io/tenant"
	// MachineNetworkPrimaryASN tag to store the primary BGP ASN the machine announces.
	// Deprecated use github.com/metal-stack/go/tag/MachineNetworkPrimaryASN
	MachineNetworkPrimaryASN = "machine.metal-stack.io/network.primary.asn"
	// MachineRack tag to store the rack that this machine is placed in.
	// Deprecated use github.com/metal-stack/go/tag/MachineRack
	MachineRack = "machine.metal-stack.io/rack"
	// MachineChassis tag to store the machine chassis.
	// Deprecated use github.com/metal-stack/go/tag/MachineChassis
	MachineChassis = "machine.metal-stack.io/chassis"
)
