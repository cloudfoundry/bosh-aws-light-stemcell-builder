package ec2cli

import (
	"bufio"
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2"
	"os/exec"
	"strings"
)

func (e *EC2Cli) DescribeConversionTask(taskResource ec2.StatusResource) (ec2.StatusInfo, error) {
	describeTask := exec.Command(
		"ec2-describe-conversion-tasks",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", e.config.Region,
		"--show-transfer-details",
		taskResource.ID(),
	)

	stderr := &bytes.Buffer{}
	describeTask.Stderr = stderr

	stdout, err := describeTask.Output()
	if err != nil {
		return ec2.ConversionTaskInfo{}, fmt.Errorf("Error getting import volume status for task: %s: %s, stderr: %s", taskResource.ID(), err, stderr.String())
	}

	outputLines := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
	}

	firstLineFields := strings.Fields(outputLines[0])
	secondLineFields := strings.Fields(outputLines[1])

	info := ec2.ConversionTaskInfo{
		TaskID:           taskResource.ID(),
		ConversionStatus: firstLineFields[7],
	}

	// the ec2 api cli changes the output format if the task is completed :(
	if info.ConversionStatus == ec2.ConversionTaskCompletedStatus {
		info.EBSVolumeID = secondLineFields[6]
		info.ManifestUrl = secondLineFields[12] // ManifestUrl is used for cleaning later
	}

	return info, nil
}
