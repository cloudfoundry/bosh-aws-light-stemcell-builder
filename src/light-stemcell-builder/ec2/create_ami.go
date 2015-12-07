package ec2

import (
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2cli"
)

// CreateAmi creates a single AMI by creating a snapshot of a provided EBS volume
func CreateAmi(volumeID string, ec2Config ec2cli.Config, amiConfig ec2ami.Config) (string, error) {
	if validationError := amiConfig.Validate(); validationError != nil {
		return "", validationError
	}

	snapshotID, err := ec2cli.CreateSnapshot(ec2Config, volumeID)
	if err != nil {
		return "", fmt.Errorf("creating snapshot: %s", err)
	}

	return ec2cli.RegisterImage(ec2Config, amiConfig, snapshotID)
}
