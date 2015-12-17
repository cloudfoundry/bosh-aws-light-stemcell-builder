package ec2

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"time"
)

// CreateAmi creates a single AMI by creating a snapshot of a provided EBS volume
func CreateAmi(aws AWS, volumeID string, amiConfig ec2ami.Config) (ec2ami.Info, error) {
	if validationError := amiConfig.Validate(); validationError != nil {
		return ec2ami.Info{}, validationError
	}

	snapshotID, err := aws.CreateSnapshot(volumeID)
	if err != nil {
		return ec2ami.Info{}, fmt.Errorf("Error creating snapshot: %s", err)
	}

	waiterConfig := WaiterConfig{
		Resource:      SnapshotResource{
			SnapshotID: snapshotID,
			SnapshotRegion: aws.GetConfig().Region,
		},
		DesiredStatus: SnapshotCompletedStatus,
		PollTimeout:   10 * time.Minute,
	}

	_, err = WaitForStatus(aws.DescribeSnapshot, waiterConfig)
	if err != nil {
		return ec2ami.Info{}, fmt.Errorf("Error waiting for snapshot to become available: %s", err)
	}

	amiID, err := aws.RegisterImage(amiConfig, snapshotID)
	if err != nil {
		return ec2ami.Info{}, err
	}

	amiConfig.AmiID = amiID

	waiterConfig = WaiterConfig{
		Resource:      &amiConfig,
		DesiredStatus: ec2ami.AmiAvailableStatus,
		PollTimeout:   10 * time.Minute,
	}

	statusInfo, err := WaitForStatus(aws.DescribeImage, waiterConfig)
	if err != nil {
		return ec2ami.Info{}, fmt.Errorf("Error waiting for ami %s to be available %s", amiID, err)
	}

	if amiConfig.Public {
		err = aws.MakeImagePublic(amiConfig)
		if err != nil {
			return ec2ami.Info{}, fmt.Errorf("Error making image %s public", amiID)
		}
	}
	amiInfo := statusInfo.(ec2ami.Info)
	if validationError := amiInfo.InputConfig.Validate(); validationError != nil {
		return ec2ami.Info{}, validationError
	}

	return amiInfo, nil
}
