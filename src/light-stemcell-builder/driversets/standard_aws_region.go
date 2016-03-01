package driversets

import (
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers"
	"light-stemcell-builder/resources"
)

type StandardRegionDriverSet struct {
	machineImageDriver *drivers.SDKMachineImageDriver
	snapshotDriver     *drivers.SDKSnapshotFromImageDriver
	createAmiDriver    *drivers.SDKCreateAmiDriver
	copyAmiDriver      *drivers.SDKCopyAmiDriver
}

func NewStandardRegionDriverSet(logDest io.Writer, creds config.Credentials) StandardRegionDriverSet {
	return StandardRegionDriverSet{
		machineImageDriver: drivers.NewMachineImageDriver(logDest, creds),
		snapshotDriver:     drivers.NewSnapshotFromImageDriver(logDest, creds),
		createAmiDriver:    drivers.NewCreateAmiDriver(logDest, creds),
		copyAmiDriver:      drivers.NewCopyAmiDriver(logDest, creds),
	}
}

func (s *StandardRegionDriverSet) CreateMachineImageDriver() resources.MachineImageDriver {
	return s.machineImageDriver
}

func (s *StandardRegionDriverSet) CreateSnapshotDriver() resources.SnapshotDriver {
	return s.snapshotDriver
}

func (s *StandardRegionDriverSet) CreateAmiDriver() resources.AmiDriver {
	return s.createAmiDriver
}

func (s *StandardRegionDriverSet) CopyAmiDriver() resources.AmiDriver {
	return s.copyAmiDriver
}
