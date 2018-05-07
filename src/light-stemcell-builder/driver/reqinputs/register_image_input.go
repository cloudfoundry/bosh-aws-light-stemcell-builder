package reqinputs

import (
	"fmt"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	firstDeviceNameHVMAmi = "/dev/xvda"
	firstDeviceNamePVAmi  = "/dev/sda"
)

// NewHVMAmiRequestInput builds the required input to create an HVM AMI
func NewHVMAmiRequestInput(amiName string, amiDescription string, snapshotID string) *ec2.RegisterImageInput {
	return &ec2.RegisterImageInput{
		SriovNetSupport:    aws.String("simple"),
		Architecture:       aws.String(resources.AmiArchitecture),
		Description:        aws.String(amiDescription),
		VirtualizationType: aws.String(resources.HvmAmiVirtualization),
		Name:               aws.String(amiName),
		RootDeviceName:     aws.String(firstDeviceNameHVMAmi),
		EnaSupport:         aws.Bool(true),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			&ec2.BlockDeviceMapping{
				DeviceName: aws.String(firstDeviceNameHVMAmi),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          aws.String(snapshotID),
				},
			},
		},
	}
}

// NewPVAmiRequest builds the required input to create an PV AMI
func NewPVAmiRequest(amiName string, amiDescription string, snapshotID string, kernelID string) *ec2.RegisterImageInput {
	return &ec2.RegisterImageInput{
		Architecture:       aws.String(resources.AmiArchitecture),
		Description:        aws.String(amiDescription),
		VirtualizationType: aws.String(resources.PvAmiVirtualization),
		Name:               aws.String(amiName),
		RootDeviceName:     aws.String(fmt.Sprintf("%s1", firstDeviceNamePVAmi)),
		KernelId:           aws.String(kernelID),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			&ec2.BlockDeviceMapping{
				DeviceName: aws.String(firstDeviceNamePVAmi),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          aws.String(snapshotID),
				},
			},
		},
	}
}
