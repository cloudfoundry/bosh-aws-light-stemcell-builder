package driversets

import (
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers"
	"light-stemcell-builder/resources"
)

type IsolatedRegionDriverSet struct {
	machineImageDriver *drivers.SDKMachineImageManifestDriver
	volumeDriver       *drivers.SDKVolumeDriver
	snapshotDriver     *drivers.SDKSnapshotFromVolumeDriver
	createAmiDriver    *drivers.SDKCreateAmiDriver
}

func NewIsolatedRegionDriverSet(logDest io.Writer, creds config.Credentials) IsolatedRegionDriverSet {
	return IsolatedRegionDriverSet{
		machineImageDriver: drivers.NewMachineImageManifestDriver(logDest, creds),
		volumeDriver:       drivers.NewVolumeDriver(logDest, creds),
		snapshotDriver:     drivers.NewSnapshotFromVolumeDriver(logDest, creds),
		createAmiDriver:    drivers.NewCreateAmiDriver(logDest, creds),
	}
}

func (s *IsolatedRegionDriverSet) CreateMachineImageDriver() resources.MachineImageDriver {
	return s.machineImageDriver
}

func (s *IsolatedRegionDriverSet) CreateVolumeDriver() resources.VolumeDriver {
	return s.volumeDriver
}

func (s *IsolatedRegionDriverSet) CreateSnapshotDriver() resources.SnapshotDriver {
	return s.snapshotDriver
}

func (s *IsolatedRegionDriverSet) CreateAmiDriver() resources.AmiDriver {
	return s.createAmiDriver
}
