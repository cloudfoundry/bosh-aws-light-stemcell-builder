package ec2cli

import (
	"fmt"
	"os/exec"
)

func makeImagePublic(ec2Config Config, amiID string) error {
	makePublicCommand := exec.Command(
		"ec2-modify-image-attribute",
		"-O", ec2Config.AccessKey,
		"-W", ec2Config.SecretKey,
		"--region", ec2Config.Region,
		"-l",
		"--add", "all",
		amiID,
	)

	err := makePublicCommand.Run()
	if err != nil {
		return fmt.Errorf("making AMI: %s public: %s", amiID, err)
	}

	return nil
}
