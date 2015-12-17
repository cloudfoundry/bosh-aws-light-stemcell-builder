package ec2cli

import (
	"bufio"
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"os/exec"
	"strings"
)

func (e *EC2Cli) DescribeImage(amiResource ec2.StatusResource) (ec2.StatusInfo, error) {
	amiConfig := amiResource.(*ec2ami.Config)

	describeImage := exec.Command(
		"ec2-describe-images",
		"-O", e.config.AccessKey,
		"-W", e.config.SecretKey,
		"--region", amiConfig.Region,
		amiConfig.AmiID,
	)

	stderr := &bytes.Buffer{}
	describeImage.Stderr = stderr

	stdout, err := describeImage.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "Client.InvalidAMIID.NotFound") {
			return ec2ami.Info{}, ec2ami.NonAvailableAmiError{AmiID: amiConfig.AmiID, AmiStatus: ec2ami.AmiUnknownStatus}
		}
		return ec2ami.Info{}, fmt.Errorf("Error getting image status for image: %s: %s, stderr: %s", amiConfig.AmiID, err, stderr.String())
	}

	outputLines := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
	}

	if len(outputLines) == 0 {
		return ec2ami.Info{}, ec2ami.NonAvailableAmiError{AmiID: amiConfig.AmiID, AmiStatus: ec2ami.AmiUnknownStatus}
	}

	firstLineFields := strings.Fields(outputLines[0])

	imageInfo := ec2ami.Info{
		AmiID:              amiConfig.AmiID,
		Region:             amiConfig.Region,
		InputConfig:        *amiConfig,
		Name:               firstLineFields[1],
		ImageStatus:        firstLineFields[4],
		Accessibility:      firstLineFields[5],
		Architecture:       firstLineFields[6],
		KernelId:           firstLineFields[8],
		VirtualizationType: firstLineFields[10],
	}

	// If the block device mapping isn't set yet, then just return the first line fields
	if len(outputLines) == 2 {
		secondLineFields := strings.Fields(outputLines[1])
		imageInfo.SnapshotID = secondLineFields[3]
		imageInfo.StorageType = secondLineFields[6]
	}

	return imageInfo, nil
}
