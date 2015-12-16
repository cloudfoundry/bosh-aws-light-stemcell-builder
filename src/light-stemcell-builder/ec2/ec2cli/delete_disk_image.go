package ec2cli

import (
	"bytes"
	"fmt"
	"os/exec"
)

// DeleteDiskImage deletes a task's disk image from S3
func (e EC2Cli) DeleteDiskImage(taskID string) error {
	fmt.Printf("Region: %s, taskID: %s", e.config.Region, taskID)
	deleteDiskImage := exec.Command(
		"ec2-delete-disk-image",
		"-o", e.config.AccessKey,
		"-w", e.config.SecretKey,
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		"-t", taskID,
	)

	errBuff := &bytes.Buffer{}
	deleteDiskImage.Stderr = errBuff

	err := deleteDiskImage.Run()
	if err != nil {
		return fmt.Errorf("deleting disk image with task id %s: %s, stderr: %s", taskID, err, errBuff.String())
	}

	return nil
}
