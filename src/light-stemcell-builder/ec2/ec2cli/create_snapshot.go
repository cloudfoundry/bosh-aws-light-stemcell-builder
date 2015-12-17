package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"
)

func (e *EC2Cli) CreateSnapshot(volumeID string) (string, error) {
	createSnapshot := exec.Command(
		"ec2-create-snapshot",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		volumeID,
	)
	secondField, err := command.SelectField(2)
	if err != nil {
		return "", err
	}

	snapshotID, err := command.RunPipeline([]*exec.Cmd{createSnapshot, secondField})
	if err != nil {
		return "", fmt.Errorf("Error waiting for snapshot %s to be ready: %s", snapshotID, err)
	}

	return snapshotID, nil
}
