package ec2cli

import (
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"os/exec"
)

// DeregisterImage deregisters an AMI, does not return an error if the AMI does not exist
func (e *EC2Cli) DeregisterImage(amiConfig ec2ami.Config) error {
	deregisterImage := exec.Command(
		"ec2-deregister",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", amiConfig.Region,
		amiConfig.AmiID,
	)

	errBuff := &bytes.Buffer{}
	deregisterImage.Stderr = errBuff

	err := deregisterImage.Run()
	if err != nil {
		return fmt.Errorf("Error deleting AMI with id %s: %s, stderr: %s", amiConfig.AmiID, err, errBuff.String())
	}

	return nil
}
