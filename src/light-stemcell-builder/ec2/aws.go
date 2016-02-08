package ec2

import (
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2instance"
)

// AWS defines any methods that must be implemented at the low level for the stemcell building process
//go:generate counterfeiter -o fakes/fake_aws.go . AWS
type AWS interface {
	Configure(c Config)
	GetConfig() Config

	ImportVolume(imagePath string) (string, error)
	ResumeImport(taskID string, imagePath string) error
	DeleteVolume(volumeID string) error
	DeleteDiskImage(taskID string) error

	DescribeConversionTask(taskResource StatusResource) (StatusInfo, error)
	DescribeVolume(volumeResource StatusResource) (StatusInfo, error)
	DescribeImage(amiResource StatusResource) (StatusInfo, error)
	DescribeSnapshot(snapshotResource StatusResource) (StatusInfo, error)

	RegisterImage(amiConfig ec2ami.Config, snapshotID string) (string, error)
	CopyImage(amiConfig ec2ami.Config, destination string) (string, error)
	MakeImagePublic(amiConfig ec2ami.Config) error
	DeregisterImage(amiConfig ec2ami.Config) error

	CreateSnapshot(volumeID string) (string, error)
	DeleteSnapshot(snapshotID string, region string) error

	RunInstance(config ec2instance.Config) (ec2instance.Info, error)
	DescribeInstance(instance StatusResource) (StatusInfo, error)
	TerminateInstance(instance ec2instance.Info) error
}
