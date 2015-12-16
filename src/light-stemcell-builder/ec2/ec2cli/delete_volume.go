package ec2cli

import (
	"bytes"
	"fmt"
	"os/exec"
)

// DeleteVolume removes a volume from EBS, does not returns an error if the volume does not exist
func (e *EC2Cli) DeleteVolume(volumeID string) error {
	deleteVolume := exec.Command(
		"ec2-delete-volume",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		volumeID,
	)

	errBuff := &bytes.Buffer{}
	deleteVolume.Stderr = errBuff

	err := deleteVolume.Run()
	if err != nil {
		return fmt.Errorf("deleting volume with id %s: %s, stderr: %s", volumeID, err, errBuff.String())
	}

	return nil
}
