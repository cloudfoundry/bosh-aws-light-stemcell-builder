package ec2cli

import (
	"light-stemcell-builder/command"
	"os/exec"
	"time"
)

func ResumeImport(c Config, taskID string, imagePath string) error {
	importVolume := exec.Command(
		"ec2-resume-import",
		"-t", taskID,
		"-o", c.AccessKey,
		"-w", c.SecretKey,
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
		imagePath,
	)

	importVolumeCommands := []*exec.Cmd{importVolume}

	_, err := command.RunPipelineWithTimeout(1*time.Minute, importVolumeCommands)
	return err
}
