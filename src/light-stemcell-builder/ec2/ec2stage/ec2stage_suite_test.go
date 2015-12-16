package ec2stage_test

import (
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

// dummyAWS is a dummy implementation of the AWS interface
type dummyAWS struct{}

func (a dummyAWS) Configure(c ec2.Config) {}
func (a dummyAWS) GetConfig() ec2.Config {
	return ec2.Config{}
}

func (a dummyAWS) ImportVolume(imagePath string) (string, error) {
	return "", nil
}

func (a dummyAWS) ResumeImport(taskID string, imagePath string) error {
	return nil
}

func (a dummyAWS) DeleteVolume(volumeID string) error {
	return nil
}

func (a dummyAWS) DeleteDiskImage(taskID string) error {
	return nil
}

func (a dummyAWS) DescribeConversionTask(taskResource ec2.StatusResource) (ec2.StatusInfo, error) {
	return ec2.ConversionTaskInfo{}, nil
}

func (a dummyAWS) DescribeVolume(volumeResource ec2.StatusResource) (ec2.StatusInfo, error) {
	return ec2.VolumeInfo{}, nil
}

func (a dummyAWS) DescribeImage(amiResource ec2.StatusResource) (ec2.StatusInfo, error) {
	return ec2ami.Info{}, nil
}

func (a dummyAWS) DescribeSnapshot(snapshotResource ec2.StatusResource) (ec2.StatusInfo, error) {
	return ec2.SnapshotInfo{}, nil
}

func (a dummyAWS) RegisterImage(amiConfig ec2ami.Config, snapshotID string) (string, error) {
	return "", nil
}
func (a dummyAWS) CopyImage(amiConfig ec2ami.Config, destination string) (string, error) {
	return "", nil
}
func (a dummyAWS) MakeImagePublic(amiConfig ec2ami.Config) error {
	return nil
}
func (a dummyAWS) DeregisterImage(amiConfig ec2ami.Config) error {
	return nil
}

func (a dummyAWS) CreateSnapshot(volumeID string) (string, error) {
	return "", nil
}
func (a dummyAWS) DeleteSnapshot(snapshotID string) error {
	return nil
}

func TestEc2stage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ec2stage Suite")
}
