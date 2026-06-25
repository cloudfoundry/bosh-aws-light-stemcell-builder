package reqinputs

import (
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	firstDeviceNameHVMAmi = "/dev/xvda"
	firstDeviceNamePVAmi  = "/dev/sda"
)

// NewHVMAmiRequestInput builds the required input to create an HVM AMI
func NewHVMAmiRequestInput(amiName string, amiDescription string, snapshotID string, efi bool) *ec2.RegisterImageInput {
	bootMode := ec2types.BootModeValuesLegacyBios
	if efi {
		bootMode = ec2types.BootModeValuesUefiPreferred
	}
	return &ec2.RegisterImageInput{
		SriovNetSupport:    aws.String("simple"),
		Architecture:       ec2types.ArchitectureValuesX8664,
		Description:        aws.String(amiDescription),
		VirtualizationType: aws.String(resources.HvmAmiVirtualization),
		Name:               aws.String(amiName),
		RootDeviceName:     aws.String(firstDeviceNameHVMAmi),
		EnaSupport:         aws.Bool(true),
		BootMode:           bootMode,
		BlockDeviceMappings: []ec2types.BlockDeviceMapping{
			{
				DeviceName: aws.String(firstDeviceNameHVMAmi),
				Ebs: &ec2types.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          aws.String(snapshotID),
				},
			},
		},
	}
}
