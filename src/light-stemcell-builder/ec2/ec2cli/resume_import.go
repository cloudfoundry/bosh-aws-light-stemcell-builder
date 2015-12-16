package ec2cli

import (
	"light-stemcell-builder/command"
	"os/exec"
	"time"
)

func (e *EC2Cli) ResumeImport(taskID string, imagePath string) error {
	importVolume := exec.Command(
		"ec2-resume-import",
		"-t", taskID,
		"-o", e.config.AccessKey,
		"-w", e.config.SecretKey,
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		imagePath,
	)

	importVolumeCommands := []*exec.Cmd{importVolume}

	_, err := command.RunPipelineWithTimeout(1*time.Minute, importVolumeCommands)
	return err
}
