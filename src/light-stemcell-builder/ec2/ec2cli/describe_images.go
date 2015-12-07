package ec2cli

import (
	"fmt"
	"light-stemcell-builder/command"
	"os/exec"
	"strings"
)

type ImageInfo struct {
	Accessibility      string
	Name               string
	ImageStatus        string
	VirtualizationType string
}

func (i ImageInfo) Status() string {
	return i.ImageStatus
}

func DescribeImageStatus(c Config, amiID string) (statusInfo, error) {
	describeTask := exec.Command(
		"ec2-describe-images",
		"-O", c.AccessKey,
		"-W", c.SecretKey,
		"--region", c.Region,
		amiID,
	)

	firstLine, err := command.SelectLine(1)
	if err != nil {
		return ImageInfo{}, err
	}

	filterFields, err := command.SelectFields([]int{3, 5, 6, 11})
	if err != nil {
		return ImageInfo{}, err
	}

	describeImageCommand := []*exec.Cmd{describeTask, firstLine, filterFields}

	rawInfo, err := command.RunPipeline(describeImageCommand)
	if err != nil {
		return ImageInfo{}, fmt.Errorf("fetching info for AMI %s: %s", amiID, err)
	}

	infoFields := strings.Split(rawInfo, " ")
	imageInfo := ImageInfo{
		Name:               infoFields[0],
		ImageStatus:        infoFields[1],
		Accessibility:      infoFields[2],
		VirtualizationType: infoFields[3],
	}

	return imageInfo, nil
}
