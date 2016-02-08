package ec2cli

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2instance"
	"os/exec"
)

func (e *EC2Cli) TerminateInstance(info ec2instance.Info) error {
	runCmd := exec.Command(
		"ec2-terminate-instances",
		info.ID(),
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
	)

	output, err := runCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to run `ec2-terminate-instances`. Error: %s\nOutput: %s", err.Error(), output)
	}

	return nil
}
