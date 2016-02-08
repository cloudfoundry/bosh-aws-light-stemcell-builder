package ec2cli

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2cli/table"
	"light-stemcell-builder/ec2/ec2instance"
	"os/exec"
	"strconv"
)

func (e *EC2Cli) RunInstance(config ec2instance.Config) (ec2instance.Info, error) {
	runCmd := exec.Command(
		"ec2-run-instances",
		config.AmiID,
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		"--instance-type", config.InstanceType,
		"--associate-public-ip-address", strconv.FormatBool(config.AssociatePublicIP),
	)

	output, err := runCmd.CombinedOutput()
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to run `ec2-run-instances`. Error: %s\nOutput: %s", err.Error(), output)
	}

	instance := ec2instance.Info{}
	err = table.Marshall(string(output), &instance)
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to marshall output into ec2instance.Info: %s", err.Error())
	}

	return instance, nil
}
