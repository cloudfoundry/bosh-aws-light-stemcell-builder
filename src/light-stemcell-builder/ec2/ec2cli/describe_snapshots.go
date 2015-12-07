package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"
)

type SnapshotInfo struct {
	SnapshotStatus string
}

func (i SnapshotInfo) Status() string {
	return i.SnapshotStatus
}

func DescribeSnapshot(c Config, snapshotID string) (statusInfo, error) {
	describeSnapshot := exec.Command(
		"ec2-describe-snapshots",
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
		snapshotID,
	)

	fourthField, err := command.SelectField(4)
	if err != nil {
		return SnapshotInfo{}, err
	}

	describeSnapshotCommands := []*exec.Cmd{describeSnapshot, fourthField}
	status, err := command.RunPipeline(describeSnapshotCommands)
	if err != nil {
		return SnapshotInfo{}, fmt.Errorf("fetching snapshot information for snapshot %s: %s", snapshotID, err)
	}

	snapshotInfo := SnapshotInfo{
		SnapshotStatus: status,
	}
	return snapshotInfo, nil
}
