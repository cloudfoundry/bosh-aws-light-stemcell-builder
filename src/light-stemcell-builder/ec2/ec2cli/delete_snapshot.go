package ec2cli

import (
	"bytes"
	"fmt"
	"os/exec"
)

// DeleteSnapshot deletes a snapshot, does not return an error if the AMI does not exist
func (e *EC2Cli) DeleteSnapshot(snapshotID string, region string) error {
	deleteSnapshot := exec.Command(
		"ec2-delete-snapshot",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", region,
		snapshotID,
	)

	errBuff := &bytes.Buffer{}
	deleteSnapshot.Stderr = errBuff

	err := deleteSnapshot.Run()
	if err != nil {
		return fmt.Errorf("deleting snapshot with id %s: %s, stderr: %s", snapshotID, err, errBuff.String())
	}

	return nil
}
