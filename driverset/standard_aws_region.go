package driverset

import (
	"io"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"
)

//counterfeiter:generate . StandardRegionDriverSet
type StandardRegionDriverSet interface {
	MachineImageDriver() resources.MachineImageDriver
	CreateSnapshotDriver() resources.SnapshotDriver
	CreateAmiDriver() resources.AmiDriver
	CopyAmiDriver() resources.AmiDriver
	KmsDriver() resources.KmsDriver
}

type standardRegionDriverSet struct {
	machineImageDriver resources.MachineImageDriver
	snapshotDriver     *driver.SDKSnapshotFromImageDriver
	amiDriver          *driver.SDKCreateAmiDriver
	copyAmiDriver      *driver.SDKCopyAmiDriver
	kmsDriver          *driver.SDKKmsDriver
}

func NewStandardRegionDriverSet(logDest io.Writer, creds config.Credentials) StandardRegionDriverSet {
	return &standardRegionDriverSet{
		machineImageDriver: struct {
			*driver.SDKCreateMachineImageDriver
			*driver.SDKDeleteMachineImageDriver
		}{
			driver.NewCreateMachineImageDriver(logDest, creds),
			driver.NewDeleteMachineImageDriver(logDest, creds),
		},
		snapshotDriver: driver.NewSnapshotFromImageDriver(logDest, creds),
		amiDriver:      driver.NewCreateAmiDriver(logDest, creds),
		copyAmiDriver:  driver.NewCopyAmiDriver(logDest, creds),
		kmsDriver:      driver.NewKmsDriver(logDest, creds),
	}
}

func (s *standardRegionDriverSet) MachineImageDriver() resources.MachineImageDriver {
	return s.machineImageDriver
}

func (s *standardRegionDriverSet) CreateSnapshotDriver() resources.SnapshotDriver {
	return s.snapshotDriver
}

func (s *standardRegionDriverSet) CreateAmiDriver() resources.AmiDriver {
	return s.amiDriver
}

func (s *standardRegionDriverSet) CopyAmiDriver() resources.AmiDriver {
	return s.copyAmiDriver
}

func (s *standardRegionDriverSet) KmsDriver() resources.KmsDriver {
	return s.kmsDriver
}
