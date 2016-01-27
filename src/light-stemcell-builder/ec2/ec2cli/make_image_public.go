package ec2cli

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"os/exec"
)

func (e *EC2Cli) MakeImagePublic(amiConfig ec2ami.Config) error {
	makePublicCommand := exec.Command(
		"ec2-modify-image-attribute",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", amiConfig.Region,
		"-l",
		"--add", "all",
		amiConfig.AmiID,
	)

	output, err := makePublicCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error making AMI: %s public: %s with output: %s", amiConfig.AmiID, err, output)
	}

	return nil
}
