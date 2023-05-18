package driverset

import (
	"io"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws/session"
)

//counterfeiter:generate . StandardRegionDriverSet
type StandardRegionDriverSet interface {
	MachineImageDriver() resources.MachineImageDriver
	CreateSnapshotDriver() resources.SnapshotDriver
	CreateAmiDriver() resources.AmiDriver
	CopyAmiDriver() resources.AmiDriver
}

type standardRegionDriverSet struct {
	machineImageDriver resources.MachineImageDriver
	snapshotDriver     *driver.SDKSnapshotFromImageDriver
	amiDriver          *driver.SDKCreateAmiDriver
	copyAmiDriver      *driver.SDKCopyAmiDriver
}

func NewStandardRegionDriverSet(logDest io.Writer, awsRegionSession *session.Session, creds config.Credentials) StandardRegionDriverSet {
	return &standardRegionDriverSet{
		machineImageDriver: struct {
			*driver.SDKCreateMachineImageDriver
			*driver.SDKDeleteMachineImageDriver
		}{
			driver.NewCreateMachineImageDriver(logDest, awsRegionSession, creds),
			driver.NewDeleteMachineImageDriver(logDest, awsRegionSession, creds),
		},
		snapshotDriver: driver.NewSnapshotFromImageDriver(logDest, awsRegionSession, creds),
		amiDriver:      driver.NewCreateAmiDriver(logDest, awsRegionSession, creds),
		copyAmiDriver:  driver.NewCopyAmiDriver(logDest, awsRegionSession, creds),
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
