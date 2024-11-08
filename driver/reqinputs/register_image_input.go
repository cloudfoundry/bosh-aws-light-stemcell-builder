package reqinputs

import (
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	firstDeviceNameHVMAmi = "/dev/xvda"
	firstDeviceNamePVAmi  = "/dev/sda"
)

// NewHVMAmiRequestInput builds the required input to create an HVM AMI
func NewHVMAmiRequestInput(amiName string, amiDescription string, snapshotID string, efi bool) *ec2.RegisterImageInput {
	bootMode := "legacy-bios"
	if efi {
		bootMode = "uefi-preferred"
	}
	return &ec2.RegisterImageInput{
		SriovNetSupport:    aws.String("simple"),
		Architecture:       aws.String(resources.AmiArchitecture),
		Description:        aws.String(amiDescription),
		VirtualizationType: aws.String(resources.HvmAmiVirtualization),
		Name:               aws.String(amiName),
		RootDeviceName:     aws.String(firstDeviceNameHVMAmi),
		EnaSupport:         aws.Bool(true),
		BootMode:           &bootMode,
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
