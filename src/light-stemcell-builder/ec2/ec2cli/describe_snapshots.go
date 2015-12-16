package ec2cli

import (
	"bufio"
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2"
	"os/exec"
	"strings"
)

func (e *EC2Cli) DescribeSnapshot(snapshotResource ec2.StatusResource) (ec2.StatusInfo, error) {
	describeSnapshot := exec.Command(
		"ec2-describe-snapshots",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		snapshotResource.ID(),
	)

	stderr := &bytes.Buffer{}
	describeSnapshot.Stderr = stderr

	stdout, err := describeSnapshot.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "Client.InvalidSnapshot.NotFound") {
			return ec2.SnapshotInfo{}, ec2.NonCompletedSnapshotError{SnapshotID: snapshotResource.ID(), SnapshotStatus: ec2.SnapshotUnknownStatus}
		}
		return ec2.SnapshotInfo{}, fmt.Errorf("getting snapshot status for snapshot: %s: %s, stderr: %s", snapshotResource.ID(), err, stderr.String())
	}

	outputLines := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
	}

	if len(outputLines) == 0 {
		return ec2.SnapshotInfo{}, ec2.NonCompletedSnapshotError{SnapshotID: snapshotResource.ID(), SnapshotStatus: ec2.SnapshotUnknownStatus}
	}

	firstLineFields := strings.Fields(outputLines[0])
	snapshotInfo := ec2.SnapshotInfo{
		SnapshotStatus: firstLineFields[3],
	}
	return snapshotInfo, nil
}
