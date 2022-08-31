package tag

const (
	// MachineID tag to store machine ID
	MachineID = "machine.metal-stack.io/id"
	// MachineName tag to store machine name
	MachineName = "machine.metal-stack.io/name"
	// MachineDescription tag to store machine description
	MachineDescription = "machine.metal-stack.io/description"
	// MachineProject tag to store the project where the machine belongs to
	MachineProject = "machine.metal-stack.io/project"
	// MachineTenant tag to store the tenant the machine belongs to
	MachineTenant = "machine.metal-stack.io/tenant"
	// MachineNetworkPrimaryASN tag to store the primary BGP ASN the machine announces.
	MachineNetworkPrimaryASN = "machine.metal-stack.io/network.primary.asn"
	// MachineRack tag to store the rack that this machine is placed in.
	MachineRack = "machine.metal-stack.io/rack"
	// MachineChassis tag to store the machine chassis.
	MachineChassis = "machine.metal-stack.io/chassis"
)

// MachineCommand defines a command send to a machine
type MachineCommand string

const (
	// MachineOnCmd power on the machine
	MachineOnCmd MachineCommand = "ON"
	// MachineOffCmd power off the machine
	MachineOffCmd MachineCommand = "OFF"
	// MachineResetCmd hard power reset the machine
	MachineResetCmd MachineCommand = "RESET"
	// MachineCycleCmd worm power cycle the machine
	MachineCycleCmd MachineCommand = "CYCLE"
	// MachineBiosCmd boot the machine into bios
	MachineBiosCmd MachineCommand = "BIOS"
	// MachineDiskCmd boot the machine from disk
	MachineDiskCmd MachineCommand = "DISK"
	// MachinePxeCmd boot the machine from pxe
	MachinePxeCmd MachineCommand = "PXE"
	// MachineReinstallCmd reinstall the machine
	MachineReinstallCmd MachineCommand = "REINSTALL"
	// ChassisIdentifyLEDOnCmd turn the identity LED on
	ChassisIdentifyLEDOnCmd MachineCommand = "LED-ON"
	// ChassisIdentifyLEDOnCmd turn the identity LED off
	ChassisIdentifyLEDOffCmd MachineCommand = "LED-OFF"
	// UpdateFirmwareCmd update machine firmware
	UpdateFirmwareCmd MachineCommand = "UPDATE-FIRMWARE"
)
