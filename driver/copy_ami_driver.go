package driver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
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
		WithCredentials(awsCreds(d.creds)).
		WithRegion(dstRegion).
		WithLogger(newDriverLogger(d.logger))

	ec2Client := ec2.New(session.Must(session.NewSession(awsConfig)))

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("copying AMI from source AMI: %s\n", driverConfig.ExistingAmiID)
	input := &ec2.CopyImageInput{
		Description:   &driverConfig.Description,
		Name:          &driverConfig.Name,
		SourceImageId: &driverConfig.ExistingAmiID,
		SourceRegion:  &srcRegion,
		Encrypted:     &driverConfig.Encrypted,
	}
	if driverConfig.KmsKeyId != "" {
		input.KmsKeyId = &driverConfig.KmsKeyId
	}
	output, err := ec2Client.CopyImage(input)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("copying AMI: %s", err)
	}

	amiIDptr := output.ImageId
	if amiIDptr == nil {
		return resources.Ami{}, errors.New("AMI id nil")
	}

	d.logger.Printf("waiting for AMI: %s to be available\n", *amiIDptr)
	err = d.waitUntilImageAvailable(&ec2.DescribeImagesInput{
		ImageIds: []*string{amiIDptr},
	}, ec2Client)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to be available: %s", *amiIDptr, err)
	}

	name := aws.String(driverConfig.AmiProperties.Tags["distro"] + "-" + driverConfig.AmiProperties.Tags["version"])
	distro := aws.String(driverConfig.AmiProperties.Tags["distro"])
	version := aws.String(driverConfig.AmiProperties.Tags["version"])
	tags := &ec2.CreateTagsInput{
		Resources: []*string{
			amiIDptr,
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: name,
			},
			{
				Key:   aws.String("distro"),
				Value: distro,
			},
			{
				Key:   aws.String("version"),
				Value: version,
			},
			{
				Key:   aws.String("published"),
				Value: aws.String("false"),
			},
		},
	}
	d.logger.Printf("tagging AMI: %s, with %s", *amiIDptr, tags)
	_, err = ec2Client.CreateTags(tags)
	if err != nil {
		d.logger.Printf("Error tagging AMI: %s, Error: %s ", *amiIDptr, err.Error())
	}
	if driverConfig.Accessibility == resources.PublicAmiAccessibility {
		d.logger.Printf("making AMI: %s public", *amiIDptr)
		ec2Client.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{ //nolint:errcheck
			ImageId: amiIDptr,
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: []*ec2.LaunchPermission{
					{
						Group: aws.String(publicGroup),
					},
				},
			},
		})
	}

	if driverConfig.Encrypted {
		return resources.Ami{ID: *amiIDptr, Region: dstRegion}, nil
	}

	var snapshotIDptr *string
	var snapshotErr error

	for i := 0; i < 100; i++ {
		describeImagesOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("image-id"),
					Values: []*string{aws.String(*amiIDptr)},
				},
			},
		})
		if err != nil {
			return resources.Ami{}, fmt.Errorf("failed to retrieve image %s: %v", *amiIDptr, err)
		}

		image := describeImagesOutput.Images[0]
		deviceMappings := make([]string, 0, len(image.BlockDeviceMappings))
		for _, deviceMapping := range image.BlockDeviceMappings {
			deviceMappings = append(deviceMappings, *deviceMapping.DeviceName)
			if *deviceMapping.DeviceName == *image.RootDeviceName {
				snapshotIDptr = deviceMapping.Ebs.SnapshotId
			}
		}

		if snapshotIDptr != nil {
			break
		}

		snapshotErr = fmt.Errorf(
			"snapshot for image %s not found: root device %s not found in device mappings %v",
			*amiIDptr,
			*image.RootDeviceName,
			deviceMappings,
		)

		time.Sleep(5 * time.Second)
		d.logger.Printf("waiting for snapshot to be available for AMI ID: %s...\n", *amiIDptr)
	}

	if snapshotIDptr == nil {
		return resources.Ami{}, snapshotErr
	}

	d.logger.Printf("snapshot %s for image %s found\n", *snapshotIDptr, *amiIDptr)
	tags = &ec2.CreateTagsInput{
		Resources: []*string{
			snapshotIDptr,
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: amiIDptr,
			},
			{
				Key:   aws.String("ami_id"),
				Value: amiIDptr,
			},
			{
				Key:   aws.String("distro"),
				Value: distro,
			},
			{
				Key:   aws.String("version"),
				Value: version,
			},
		},
	}
	d.logger.Printf("tagging Snapshot: %s, with %s", *snapshotIDptr, tags)
	_, err = ec2Client.CreateTags(tags)
	if err != nil {
		d.logger.Printf("Error tagging Snapshot: %s, Error: %s ", *snapshotIDptr, err.Error())
	}

	modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
		SnapshotId:    snapshotIDptr,
		Attribute:     aws.String("createVolumePermission"),
		OperationType: aws.String("add"),
		GroupNames:    []*string{aws.String("all")},
	}
	_, err = ec2Client.ModifySnapshotAttribute(modifySnapshotAttributeInput)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("making snapshot with id %s public: %v", *snapshotIDptr, err)
	}

	d.logger.Printf("snapshot %s is public\n", *snapshotIDptr)

	return resources.Ami{ID: *amiIDptr, Region: dstRegion}, nil
}

func (d *SDKCopyAmiDriver) waitUntilImageAvailable(input *ec2.DescribeImagesInput, c *ec2.EC2) error {
	ctx := aws.BackgroundContext()
	opts := []request.WaiterOption{
		request.WithWaiterDelay(request.ConstantWaiterDelay(15 * time.Second)),
		request.WithWaiterMaxAttempts(240),
	}
	return c.WaitUntilImageAvailableWithContext(ctx, input, opts...)
}
