package driverset

import (
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"
)

type IsolatedRegionDriverSet struct {
	machineImageDriver *driver.SDKMachineImageManifestDriver
	volumeDriver       *driver.SDKVolumeDriver
	snapshotDriver     *driver.SDKSnapshotFromVolumeDriver
	createAmiDriver    *driver.SDKCreateAmiDriver
}

func NewIsolatedRegionDriverSet(logDest io.Writer, creds config.Credentials) IsolatedRegionDriverSet {
	return IsolatedRegionDriverSet{
		machineImageDriver: driver.NewMachineImageManifestDriver(logDest, creds),
		volumeDriver:       driver.NewVolumeDriver(logDest, creds),
		snapshotDriver:     driver.NewSnapshotFromVolumeDriver(logDest, creds),
		createAmiDriver:    driver.NewCreateAmiDriver(logDest, creds),
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
