package driverset

// You only need **one** of these per package!
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

import (
	"io"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws/session"
)

//counterfeiter:generate . IsolatedRegionDriverSet
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

func NewIsolatedRegionDriverSet(logDest io.Writer, awsRegionSession *session.Session, creds config.Credentials) IsolatedRegionDriverSet {
	return &isolatedRegionDriverSet{
		machineImageDriver: struct {
			*driver.SDKCreateMachineImageManifestDriver
			*driver.SDKDeleteMachineImageDriver
		}{
			driver.NewCreateMachineImageManifestDriver(logDest, awsRegionSession, creds),
			driver.NewDeleteMachineImageDriver(logDest, awsRegionSession, creds),
		},
		volumeDriver: struct {
			*driver.SDKCreateVolumeDriver
			*driver.SDKDeleteVolumeDriver
		}{
			driver.NewCreateVolumeDriver(logDest, awsRegionSession, creds),
			driver.NewDeleteVolumeDriver(logDest, awsRegionSession, creds),
		},
		snapshotDriver:  driver.NewSnapshotFromVolumeDriver(logDest, awsRegionSession, creds),
		createAmiDriver: driver.NewCreateAmiDriver(logDest, awsRegionSession, creds),
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
