package ec2cli

import (
	"light-stemcell-builder/command"
	"os/exec"
)

type ConversionTaskInfo struct {
	conversionStatus string
}

func (i ConversionTaskInfo) Status() string {
	return i.conversionStatus
}

func DescribeConverionTaskStatus(c Config, taskID string) (string, error) {
	describeTask := exec.Command(
		"ec2-describe-conversion-tasks",
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
		taskID,
	)

	firstLine, err := command.SelectLine(1)
	if err != nil {
		return "", err
	}

	eighthField, err := command.SelectField(8)
	if err != nil {
		return "", err
	}

	describeTaskCommands := []*exec.Cmd{describeTask, firstLine, eighthField}

	return command.RunPipeline(describeTaskCommands)
}

func DescribeEbsVolumeID(c Config, taskID string) (string, error) {
	describeTask := exec.Command(
		"ec2-describe-conversion-tasks",
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
		taskID,
	)

	secondLine, err := command.SelectLine(2)
	if err != nil {
		return "", err
	}

	seventhField, err := command.SelectField(7)
	if err != nil {
		return "", err
	}

	volumeIDCommands := []*exec.Cmd{describeTask, secondLine, seventhField}
	return command.RunPipeline(volumeIDCommands)
}
