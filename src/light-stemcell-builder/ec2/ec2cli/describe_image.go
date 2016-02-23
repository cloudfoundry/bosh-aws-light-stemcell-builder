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

	firstLineFields := strings.Split(outputLines[0], "\t")
	/*
		Row 1 (IMAGE)
			Column | Description
			1 		 | The ID of the image
			2 		 | The source of the image
			3 		 | The date and time the image was created
			4 		 | The status of the image
			5 		 | The visibility of the image (public or private)
			6 		 | The product codes, if any, that are attached to the instance
			7 		 | The architecture of the image (i386 or x86_64)
			8 		 | The image type (machine, kernel, or ramdisk)
			9 		 | The ID of the kernel associated with the image (machine images only)
			10		 | The ID of the RAM disk associated with the image (machine images only)
			11		 | The platform of the image
			12		 | The type of root device (ebs or instance-store)
			13		 | The root device name
			14		 | The virtualization type (paravirtual or hvm)
			15		 | The Hypervisor type (xen or ovm)
		Row 2 (BLOCKDEVICE)
			1 		 | N/A
			2 		 | The device name
			3 		 | N/A
			4 		 | The ID of the snapshot
			5 		 | The volume size
			6 		 | Indicates whether the volume is deleted on instance termination (true or false)
			7 		 | The volume type
			8 		 | N/A
			9 		 | The encryption status of the volume
	*/
	imageInfo := ec2ami.Info{
		AmiID:              amiConfig.AmiID,
		Region:             amiConfig.Region,
		InputConfig:        *amiConfig,
		Name:               firstLineFields[1],
		ImageStatus:        firstLineFields[4],
		Accessibility:      firstLineFields[5],
		Architecture:       firstLineFields[7],
		KernelId:           firstLineFields[9],
		VirtualizationType: firstLineFields[14],
	}

	// If the block device mapping isn't set yet, then just return the first line fields
	if len(outputLines) == 2 {
		secondLineFields := strings.Split(outputLines[1], "\t")
		imageInfo.SnapshotID = secondLineFields[4]
		imageInfo.StorageType = secondLineFields[7]
	}

	return imageInfo, nil
}
