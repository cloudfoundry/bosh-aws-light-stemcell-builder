package ec2

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"reflect"
	"time"
)

func DeleteAmi(aws AWS, amiInfo ec2ami.Info) error {
	if validationError := amiInfo.InputConfig.Validate(); validationError != nil {
		return validationError
	}
	err := aws.DeregisterImage(amiInfo.InputConfig)
	if err != nil {
		return fmt.Errorf("Error deregistering AMI: %s", err)
	}

	waiterConfig := WaiterConfig{
		Resource:      &amiInfo.InputConfig,
		DesiredStatus: "", // we're abusing the waiter functionality here as a cheap timeout
		PollTimeout:   30 * time.Minute,
	}

	fmt.Printf("waiting for AMI %s to be deleted\n", amiInfo.AmiID)
	_, err = WaitForStatus(aws.DescribeImage, waiterConfig)
	if reflect.TypeOf(err) != reflect.TypeOf(ec2ami.NonAvailableAmiError{}) {
		return err
	}

	err = aws.DeleteSnapshot(amiInfo.SnapshotID, amiInfo.Region)
	waiterConfig = WaiterConfig{
		Resource:      SnapshotResource{
			SnapshotID: amiInfo.SnapshotID,
			SnapshotRegion: amiInfo.Region,
		},
		DesiredStatus: "", // we're abusing the waiter functionality here as a cheap timeout
		PollTimeout:   30 * time.Minute,
	}

	fmt.Printf("waiting for snapshot %s to be deleted\n", amiInfo.SnapshotID)
	_, err = WaitForStatus(aws.DescribeSnapshot, waiterConfig)
	if reflect.TypeOf(err) != reflect.TypeOf(NonCompletedSnapshotError{}) {
		return err
	}

	return nil
}
