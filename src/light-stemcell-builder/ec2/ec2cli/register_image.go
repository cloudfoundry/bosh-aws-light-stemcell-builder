package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"

	"light-stemcell-builder/ec2/ec2ami"
)

func RegisterImage(c Config, amiConfig ec2ami.Config, snapshotID string) (string, error) {
	amiName, err := amiConfig.Name()
	if err != nil {
		return "", fmt.Errorf("creating ami: %s", err)
	}

	registerSnapshot := exec.Command(
		"ec2-register",
		"-a", ec2ami.AmiArchitecture,
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
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
		return "", fmt.Errorf("registering image: %s", err)
	}

	waiterConfig := WaiterConfig{
		ResourceID:    amiID,
		DesiredStatus: imageAvailableStatus,
		FetcherConfig: c,
	}

	err = WaitForStatus(DescribeImageStatus, waiterConfig)
	if err != nil {
		return "", fmt.Errorf("waiting for ami %s to be available %s", amiID, err)
	}

	if amiConfig.Public {
		err = makeImagePublic(c, amiID)
		if err != nil {
			return "", fmt.Errorf("making image %s public", amiID)
		}
	}

	return amiID, nil
}
