package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"

	"light-stemcell-builder/config"
	"light-stemcell-builder/ec2/ec2ami"
)

var regionToAKI = map[string]string{
	"ap-northeast-1": "aki-176bf516",
	"ap-northeast-2": "aki-01a66b6f",
	"ap-southeast-1": "aki-503e7402",
	"ap-southeast-2": "aki-c362fff9",
	"eu-central-1":   "aki-184c7a05",
	"eu-west-1":      "aki-52a34525",
	"sa-east-1":      "aki-5553f448",
	"us-east-1":      "aki-919dcaf8",
	"us-gov-west":    "aki-1de98d3e",
	"us-west-1":      "aki-880531cd",
	"us-west-2":      "aki-fc8f11cc",
}

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

	if amiConfig.VirtualizationType == config.Paravirtualization {
		akiID, found := regionToAKI[amiConfig.Region]
		if !found {
			return "", fmt.Errorf("No AKI known for region: %s", amiConfig.Region)
		}

		registerSnapshot.Args = append(registerSnapshot.Args, "--kernel")
		registerSnapshot.Args = append(registerSnapshot.Args, akiID)

		rootDeviceMapping := fmt.Sprintf("/dev/sda=%s", snapshotID)

		registerSnapshot.Args = append(registerSnapshot.Args, "--block-device-mapping")
		registerSnapshot.Args = append(registerSnapshot.Args, rootDeviceMapping)
	}

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
