package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"
)

func (e *EC2Cli) ImportVolume(imagePath string) (string, error) {
	zone := fmt.Sprintf("%sa", e.config.Region)

	createTask := exec.Command(
		"ec2-import-volume",
		"-f", "RAW",
		"-b", e.config.BucketName,
		"-o", e.config.AccessKey,
		"-w", e.config.SecretKey,
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"-z", zone,
		"--region", e.config.Region,
		"--no-upload",
		imagePath,
	)

	// We expect to parse output of the form:
	//
	// Requesting volume size: 3 GB
	// TaskType  IMPORTVOLUME  TaskId  import-vol-fggu8ihs ExpirationTime  2015-12-01T21:51:13Z  Status  active  StatusMessage Pending
	// DISKIMAGE DiskImageFormat RAW DiskImageSize 3221225472  VolumeSize  3 AvailabilityZone  cn-north-1b ApproximateBytesConverted 0
	secondLine, err := command.SelectLine(2)
	if err != nil {
		return "", err
	}

	fourthField, err := command.SelectField(4)
	if err != nil {
		return "", err
	}

	createTaskPipeline := []*exec.Cmd{createTask, secondLine, fourthField}

	return command.RunPipeline(createTaskPipeline)
}
