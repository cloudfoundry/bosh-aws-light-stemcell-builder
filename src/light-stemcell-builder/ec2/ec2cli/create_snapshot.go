package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"
)

func CreateSnapshot(c Config, volumeID string) (string, error) {
	createSnapshot := exec.Command(
		"ec2-create-snapshot",
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
		volumeID,
	)
	secondField, err := command.SelectField(2)
	if err != nil {
		return "", err
	}

	snapshotID, err := command.RunPipeline([]*exec.Cmd{createSnapshot, secondField})
	if err != nil {
		return "", fmt.Errorf("waiting for snapshot %s to be ready: %s", snapshotID, err)
	}

	waiterConfig := WaiterConfig{
		ResourceID:    snapshotID,
		DesiredStatus: snapshotCompletedStatus,
		FetcherConfig: c,
	}

	err = WaitForStatus(DescribeSnapshot, waiterConfig)
	if err != nil {
		return "", fmt.Errorf("waiting for snapshot to become available: %s", err)
	}

	return snapshotID, nil
}
