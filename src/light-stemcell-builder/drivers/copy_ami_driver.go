package drivers

import (
	"errors"
	"fmt"
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// SDKCopyAmiDriver uses the AWS SDK to register an AMI from an existing snapshot in EC2
type SDKCopyAmiDriver struct {
	creds  config.Credentials
	logger *log.Logger
}

// NewCopyAmiDriver creates a SDKCopyAmiDriver for copying AMIs in EC2
func NewCopyAmiDriver(logDest io.Writer, creds config.Credentials) *SDKCopyAmiDriver {
	logger := log.New(logDest, "SDKCopyAmiDriver ", log.LstdFlags)
	return &SDKCopyAmiDriver{creds: creds, logger: logger}
}

// Create creates an AMI, copied from a source AMI, and optionally makes the AMI publically available
func (d *SDKCopyAmiDriver) Create(driverConfig resources.AmiDriverConfig) (resources.Ami, error) {
	srcRegion := d.creds.Region
	dstRegion := driverConfig.DestinationRegion

	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(d.creds.AccessKey, d.creds.SecretKey, "")).
		WithRegion(dstRegion).
		WithLogger(newDriverLogger(d.logger))

	ec2Client := ec2.New(session.New(), awsConfig)

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("copying AMI from source AMI: %s\n", driverConfig.ExistingAmiID)
	output, err := ec2Client.CopyImage(&ec2.CopyImageInput{
		Description:   &driverConfig.Description,
		Name:          &driverConfig.Name,
		SourceImageId: &driverConfig.ExistingAmiID,
		SourceRegion:  &srcRegion,
	})
	if err != nil {
		return resources.Ami{}, fmt.Errorf("copying AMI: %s", err)
	}

	amiIDptr := output.ImageId
	if amiIDptr == nil {
		return resources.Ami{}, errors.New("AMI id nil")
	}

	d.logger.Printf("waiting for AMI: %s to be available\n", *amiIDptr)
	err = ec2Client.WaitUntilImageAvailable(&ec2.DescribeImagesInput{
		ImageIds: []*string{amiIDptr},
	})
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI: %s to be available", *amiIDptr)
	}

	if driverConfig.Accessibility == resources.PublicAmiAccessibility {
		d.logger.Printf("making AMI: %s public", *amiIDptr)
		ec2Client.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
			ImageId: amiIDptr,
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: []*ec2.LaunchPermission{
					&ec2.LaunchPermission{
						Group: aws.String(publicGroup),
					},
				},
			},
		})
	}

	return resources.Ami{ID: *amiIDptr, Region: dstRegion}, nil
}
