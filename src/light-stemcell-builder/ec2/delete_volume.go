package ec2

import (
	"fmt"
	"reflect"
	"time"
)

// DeleteVolume removes a volume from EBS, returns an error if the volume does not exist
func DeleteVolume(aws AWS, volumeID string) error {
	err := aws.DeleteVolume(volumeID)
	if err != nil {
		return err
	}

	waiterConfig := WaiterConfig{
		Resource:      VolumeResource{VolumeID: volumeID},
		DesiredStatus: "", // we're abusing the waiter functionality here as a cheap timeout
		PollTimeout:   120 * time.Minute,
	}

	fmt.Printf("waiting for volume %s to be deleted\n", volumeID)
	_, err = WaitForStatus(aws.DescribeVolume, waiterConfig)
	if reflect.TypeOf(err) != reflect.TypeOf(NonAvailableVolumeError{}) {
		return err
	}

	return nil
}
