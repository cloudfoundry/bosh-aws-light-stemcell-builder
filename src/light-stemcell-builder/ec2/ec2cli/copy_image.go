package ec2cli

import (
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"os/exec"
	"strings"
)

func (e *EC2Cli) CopyImage(amiConfig ec2ami.Config, destination string) (string, error) {
	amiName, err := amiConfig.Name()
	if err != nil {
		return "", fmt.Errorf("creating ami: %s", err)
	}

	copyImage := exec.Command(
		"ec2-copy-image",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--source-region", amiConfig.Region,
		"--region", destination,
		"-s", amiConfig.AmiID,
		"-n", amiName,
		"-d", amiConfig.Description,
	)

	stderr := &bytes.Buffer{}
	copyImage.Stderr = stderr

	fmt.Printf("starting to copy ami %s to %s\n", amiConfig.AmiID, destination)
	rawOutput, err := copyImage.Output()
	if err != nil {
		return "", fmt.Errorf("coping ami: %s error: %s, stderr: %s", amiConfig.AmiID, err, stderr.String())
	}

	outputFields := strings.Fields(string(rawOutput))

	return outputFields[1], nil
}
