package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"

	"light-stemcell-builder/ec2/ec2ami"
)

func (e *EC2Cli) RegisterImage(amiConfig ec2ami.Config, snapshotID string) (string, error) {
	amiName, err := amiConfig.Name()
	if err != nil {
		return "", fmt.Errorf("Error creating ami: %s", err)
	}

	registerSnapshot := exec.Command(
		"ec2-register",
		"-a", ec2ami.AmiArchitecture,
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", amiConfig.Region,
		"-s", snapshotID,
		"-n", amiName,
		"-d", amiConfig.Description,
		"--virtualization-type", amiConfig.VirtualizationType,
	)

	secondField, err := command.SelectField(2)
	if err != nil {
		return "", err
	}

	amiID, err := command.RunPipeline([]*exec.Cmd{registerSnapshot, secondField})
	if err != nil {
		return "", fmt.Errorf("Error registering image: %s", err)
	}

	return amiID, nil
}
