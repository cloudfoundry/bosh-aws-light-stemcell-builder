package driverset

import (
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"
)

//go:generate counterfeiter -o fakes/fake_isolated_region_driver_set.go . IsolatedRegionDriverSet
type IsolatedRegionDriverSet interface {
	MachineImageDriver() resources.MachineImageDriver
	VolumeDriver() resources.VolumeDriver
	CreateSnapshotDriver() resources.SnapshotDriver
	CreateAmiDriver() resources.AmiDriver
}

type isolatedRegionDriverSet struct {
	machineImageDriver resources.MachineImageDriver
	volumeDriver       resources.VolumeDriver
	snapshotDriver     *driver.SDKSnapshotFromVolumeDriver
	createAmiDriver    *driver.SDKCreateAmiDriver
}

func NewIsolatedRegionDriverSet(logDest io.Writer, creds config.Credentials) IsolatedRegionDriverSet {
	return &isolatedRegionDriverSet{
		machineImageDriver: struct {
			*driver.SDKCreateMachineImageManifestDriver
			*driver.SDKDeleteMachineImageDriver
		}{
			driver.NewCreateMachineImageManifestDriver(logDest, creds),
			driver.NewDeleteMachineImageDriver(logDest, creds),
		},
		volumeDriver: struct {
			*driver.SDKCreateVolumeDriver
			*driver.SDKDeleteVolumeDriver
		}{
			driver.NewCreateVolumeDriver(logDest, creds),
			driver.NewDeleteVolumeDriver(logDest, creds),
		},
		snapshotDriver:  driver.NewSnapshotFromVolumeDriver(logDest, creds),
		createAmiDriver: driver.NewCreateAmiDriver(logDest, creds),
	}
}

func (s *isolatedRegionDriverSet) MachineImageDriver() resources.MachineImageDriver {
	return s.machineImageDriver
}

func (s *isolatedRegionDriverSet) VolumeDriver() resources.VolumeDriver {
	return s.volumeDriver
}

func (s *isolatedRegionDriverSet) CreateSnapshotDriver() resources.SnapshotDriver {
	return s.snapshotDriver
}

func (s *isolatedRegionDriverSet) CreateAmiDriver() resources.AmiDriver {
	return s.createAmiDriver
}
