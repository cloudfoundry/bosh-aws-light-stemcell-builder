package ec2cli

import (
	"bufio"
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2"
	"os/exec"
	"strings"
)

func (e *EC2Cli) DescribeVolume(volumeResource ec2.StatusResource) (ec2.StatusInfo, error) {
	describeVolume := exec.Command(
		"ec2-describe-volumes",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		"--filter", fmt.Sprintf("volume-id=%s", volumeResource.ID()),
	)

	stderr := &bytes.Buffer{}
	describeVolume.Stderr = stderr

	stdout, err := describeVolume.Output()
	if err != nil {
		return ec2.VolumeInfo{}, fmt.Errorf("getting volume status for volume: %s: %s, stderr: %s", volumeResource.ID(), err, stderr.String())
	}

	outputLines := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
	}

	if len(outputLines) == 0 {
		return ec2.VolumeInfo{}, ec2.NonAvailableVolumeError{VolumeID: volumeResource.ID(), VolumeStatus: ec2.VolumeUnknownStatus}
	}

	firstLineFields := strings.Fields(outputLines[0])
	volumeInfo := ec2.VolumeInfo{
		VolumeStatus: firstLineFields[4],
	}

	return volumeInfo, nil
}
