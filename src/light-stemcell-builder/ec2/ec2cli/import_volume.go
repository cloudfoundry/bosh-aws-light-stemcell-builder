package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"
)

func ImportVolume(c Config, imagePath string) (string, error) {
	zone := fmt.Sprintf("%sa", c.Region)

	createTask := exec.Command(
		"ec2-import-volume",
		"-f", "RAW",
		"-b", c.BucketName,
		"-o", c.AccessKey,
		"-w", c.SecretKey,
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"-z", zone,
		"--region", c.Region,
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

	fourhField, err := command.SelectField(4)
	if err != nil {
		return "", err
	}

	createTaskPipeline := []*exec.Cmd{createTask, secondLine, fourhField}

	return command.RunPipeline(createTaskPipeline)
}
