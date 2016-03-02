package driversets

import (
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers"
	"light-stemcell-builder/resources"
)

//go:generate counterfeiter -o fakes/fake_standard_region_driver_set.go . StandardRegionDriverSet
type StandardRegionDriverSet interface {
	CreateMachineImageDriver() resources.MachineImageDriver
	CreateSnapshotDriver() resources.SnapshotDriver
	CreateAmiDriver() resources.AmiDriver
	CopyAmiDriver() resources.AmiDriver
}

type standardRegionDriverSet struct {
	machineImageDriver *drivers.SDKMachineImageDriver
	snapshotDriver     *drivers.SDKSnapshotFromImageDriver
	createAmiDriver    *drivers.SDKCreateAmiDriver
	copyAmiDriver      *drivers.SDKCopyAmiDriver
}

func NewStandardRegionDriverSet(logDest io.Writer, creds config.Credentials) StandardRegionDriverSet {
	return &standardRegionDriverSet{
		machineImageDriver: drivers.NewMachineImageDriver(logDest, creds),
		snapshotDriver:     drivers.NewSnapshotFromImageDriver(logDest, creds),
		createAmiDriver:    drivers.NewCreateAmiDriver(logDest, creds),
		copyAmiDriver:      drivers.NewCopyAmiDriver(logDest, creds),
	}
}

func (s *standardRegionDriverSet) CreateMachineImageDriver() resources.MachineImageDriver {
	return s.machineImageDriver
}

func (s *standardRegionDriverSet) CreateSnapshotDriver() resources.SnapshotDriver {
	return s.snapshotDriver
}

func (s *standardRegionDriverSet) CreateAmiDriver() resources.AmiDriver {
	return s.createAmiDriver
}

func (s *standardRegionDriverSet) CopyAmiDriver() resources.AmiDriver {
	return s.copyAmiDriver
}
