package ec2

import (
	"fmt"
	"light-stemcell-builder/command"
	"light-stemcell-builder/ec2/ec2cli"
	"reflect"
)

const (
	importVolumeRetryAttempts = 4
	taskCompletedStatus       = "completed"
)

// ImportVolume creates an EBS volume in AWS from the supplied machine imagePath
// ImportVolume assumes that the root device will be /dev/sda1
func ImportVolume(c ec2cli.Config, imagePath string) (string, error) {

	taskID, err := ec2cli.ImportVolume(c, imagePath)
	if err != nil {
		return "", fmt.Errorf("creating import volume task: %s", err)
	}

	for i := 0; i < importVolumeRetryAttempts; i++ {
		err = ec2cli.ResumeImport(c, taskID, imagePath)
		if err == nil {
			break
		}

		if reflect.TypeOf(err) != reflect.TypeOf(command.TimeoutError{}) {
			return "", fmt.Errorf("uploading machine image: %s", err)
		}
	}

	waiterConfig := ec2cli.WaiterConfig{
		ResourceID:    taskID,
		DesiredStatus: taskCompletedStatus,
		FetcherConfig: c,
	}

	err = ec2cli.WaitForStatus(ec2cli.DescribeConverionTaskStatus, waiterConfig)
	volID, err := ec2cli.DescribeEbsVolumeID(c, taskID)
	if err != nil {
		return "", fmt.Errorf("getting volume id for task: %s", taskID)
	}

	return volID, nil
}
