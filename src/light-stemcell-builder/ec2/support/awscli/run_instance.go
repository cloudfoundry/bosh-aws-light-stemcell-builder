package awscli

import (
	"encoding/json"
	"fmt"
	"light-stemcell-builder/ec2/ec2instance"
	"os/exec"
)

type instanceState struct {
	State string `key:"Name"`
}

type instanceDescription struct {
	InstanceID      string        `key:"InstanceId"`
	PublicIPAddress string        `key:"PublicIPAddress"`
	State           instanceState `key:"State"`
}

type runInstanceOutput struct {
	Instances []instanceDescription `key:"Instances"`
}

// RunInstance shells out to the python AWS CLI to create a new instance from an AMI
func RunInstance(instanceConfig ec2instance.Config) (ec2instance.Info, error) {
	runArgs := []string{
		"--region", instanceConfig.Region,
		"--output", "json",
		"ec2", "run-instances",
		"--image-id", instanceConfig.AmiID,
		"--instance-type", instanceConfig.InstanceType,
	}
	if instanceConfig.AssociatePublicIP {
		runArgs = append(runArgs, "--associate-public-ip-address")
	}
	runCmd := exec.Command("aws", runArgs...)

	output, err := runCmd.CombinedOutput()
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to run `aws ec2 run-instances`. Error: %s\nOutput: %s", err.Error(), output)
	}

	structOutput := &runInstanceOutput{}
	err = json.Unmarshal(output, structOutput)
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Error unmarshaling json output: %s\nOutput: %s", err.Error(), output)
	}

	waitCmd := exec.Command(
		"aws",
		"--region", instanceConfig.Region,
		"--output", "json",
		"ec2", "wait", "instance-running",
		"--instance-ids", structOutput.Instances[0].InstanceID,
	)
	output, err = waitCmd.CombinedOutput()
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to run `aws ec2 wait instance-running`. Error: %s\nOutput: %s", err.Error(), output)
	}

	describeCmd := exec.Command(
		"aws",
		"--region", instanceConfig.Region,
		"--output", "json",
		"ec2", "describe-instances",
		"--instance-ids", structOutput.Instances[0].InstanceID,
	)
	output, err = describeCmd.CombinedOutput()
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Failed to run `aws ec2 describe-instances`. Error: %s\nOutput: %s", err.Error(), output)
	}

	err = json.Unmarshal(output, structOutput)
	if err != nil {
		return ec2instance.Info{}, fmt.Errorf("Error unmarshaling json output: %s\nOutput: %s", err.Error(), output)
	}

	instance := ec2instance.Info{
		InstanceID: structOutput.Instances[0].InstanceID,
		State:      structOutput.Instances[0].State.State,
		PublicIP:   structOutput.Instances[0].PublicIPAddress,
	}

	return instance, nil
}
