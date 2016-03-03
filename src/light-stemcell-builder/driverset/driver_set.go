package driverset

import "light-stemcell-builder/resources"

// A DriverSet represents a collection of resource drivers which ultimately are used to produce one or more AMIs
type DriverSet interface {
	CreateMachineImageDriver() resources.MachineImageDriver
	CreateSnapshotDriver() resources.SnapshotDriver
	CreateAmiDriver() resources.AmiDriver
}
