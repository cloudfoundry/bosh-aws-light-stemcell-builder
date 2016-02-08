package ec2cli

import (
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2cli/table"
	"light-stemcell-builder/ec2/ec2instance"
	"os/exec"
)

func (e *EC2Cli) DescribeInstance(instance ec2.StatusResource) (ec2.StatusInfo, error) {
	runCmd := exec.Command(
		"ec2-describe-instances",
		instance.ID(),
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
	)

	output, err := runCmd.CombinedOutput()
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to run `ec2-describe-instances`. Error: %s\nOutput: %s", err.Error(), output)
	}

	updatedInstance := ec2instance.Info{}
	err = table.Marshall(string(output), &updatedInstance)
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to marshall output into ec2instance.Info: %s", err.Error())
	}

	return updatedInstance, nil
}
