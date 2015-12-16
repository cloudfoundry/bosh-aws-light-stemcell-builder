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

	err := makePublicCommand.Run()
	if err != nil {
		return fmt.Errorf("making AMI: %s public: %s", amiConfig.AmiID, err)
	}

	return nil
}
