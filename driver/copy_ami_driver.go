package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

// Create creates an AMI, copied from a source AMI, and optionally makes the AMI publicly available
func (d *SDKCopyAmiDriver) Create(driverConfig resources.AmiDriverConfig) (resources.Ami, error) {
	srcRegion := d.creds.Region
	dstRegion := driverConfig.DestinationRegion

	destinationCreds := config.Credentials{
		AccessKey: d.creds.AccessKey,
		SecretKey: d.creds.SecretKey,
		RoleArn:   d.creds.RoleArn,
		Region:    dstRegion,
	}
	cfg := destinationCreds.GetAwsConfig()
	cfg.Logger = newDriverLogger(d.logger)

	ec2Client := ec2.NewFromConfig(cfg)
	ctx := context.Background()

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
		input.KmsKeyId = &driverConfig.KmsKey.ARN //nolint:staticcheck
	}
	output, err := ec2Client.CopyImage(ctx, input)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("copying AMI: %s", err)
	}

	amiIDptr := output.ImageId
	if amiIDptr == nil {
		return resources.Ami{}, errors.New("AMI id nil")
	}

	d.logger.Printf("waiting for AMI %s to be available in region %s\n", *amiIDptr, dstRegion)
	imageAvailableWaiter := ec2.NewImageAvailableWaiter(ec2Client, func(o *ec2.ImageAvailableWaiterOptions) {
		o.MinDelay = 15 * time.Second
		o.MaxDelay = 15 * time.Second
	})
	err = imageAvailableWaiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{*amiIDptr},
	}, 240*15*time.Second)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to be available: %s", *amiIDptr, err)
	}

	name := aws.String(driverConfig.Tags["distro"] + "-" + driverConfig.Tags["version"])
	distro := aws.String(driverConfig.Tags["distro"])
	version := aws.String(driverConfig.Tags["version"])
	tags := &ec2.CreateTagsInput{
		Resources: []string{*amiIDptr},
		Tags: []ec2types.Tag{
			{Key: aws.String("Name"), Value: name},
			{Key: aws.String("distro"), Value: distro},
			{Key: aws.String("version"), Value: version},
			{Key: aws.String("published"), Value: aws.String("false")},
		},
	}
	d.logger.Printf("tagging AMI: %s, with %v", *amiIDptr, tags)
	_, err = ec2Client.CreateTags(ctx, tags)
	if err != nil {
		d.logger.Printf("Error tagging AMI: %s, Error: %s ", *amiIDptr, err.Error())
	}

	for _, account := range driverConfig.SharedWithAccounts {
		accountCopy := account
		_, err := ec2Client.ModifyImageAttribute(ctx, &ec2.ModifyImageAttributeInput{
			ImageId: amiIDptr,
			LaunchPermission: &ec2types.LaunchPermissionModifications{
				Add: []ec2types.LaunchPermission{
					{UserId: &accountCopy},
				},
			},
		})
		if err != nil {
			return resources.Ami{}, fmt.Errorf("failed to share AMI '%s' with account '%s': %w", *amiIDptr, account, err)
		}
	}

	if driverConfig.Accessibility == resources.PublicAmiAccessibility {
		d.logger.Printf("making AMI: %s public", *amiIDptr)
		_, err = ec2Client.ModifyImageAttribute(ctx, &ec2.ModifyImageAttributeInput{
			ImageId: amiIDptr,
			LaunchPermission: &ec2types.LaunchPermissionModifications{
				Add: []ec2types.LaunchPermission{
					{Group: ec2types.PermissionGroupAll},
				},
			},
		})
		if err != nil {
			return resources.Ami{}, fmt.Errorf("failed to make AMI '%s' public: %w", *amiIDptr, err)
		}
	}

	var snapshotIDptr *string
	var snapshotErr error

	for i := 0; i < 100; i++ {
		describeImagesOutput, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("image-id"),
					Values: []string{*amiIDptr},
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
	snapshotTags := &ec2.CreateTagsInput{
		Resources: []string{*snapshotIDptr},
		Tags: []ec2types.Tag{
			{Key: aws.String("Name"), Value: amiIDptr},
			{Key: aws.String("ami_id"), Value: amiIDptr},
			{Key: aws.String("distro"), Value: distro},
			{Key: aws.String("version"), Value: version},
		},
	}
	d.logger.Printf("tagging Snapshot: %s, with %v", *snapshotIDptr, snapshotTags)
	_, err = ec2Client.CreateTags(ctx, snapshotTags)
	if err != nil {
		d.logger.Printf("Error tagging Snapshot: %s, Error: %s ", *snapshotIDptr, err.Error())
	}

	for _, account := range driverConfig.SharedWithAccounts {
		accountCopy := account
		modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
			SnapshotId:    snapshotIDptr,
			Attribute:     ec2types.SnapshotAttributeNameCreateVolumePermission,
			OperationType: ec2types.OperationTypeAdd,
			UserIds:       []string{accountCopy},
		}
		_, err = ec2Client.ModifySnapshotAttribute(ctx, modifySnapshotAttributeInput)
		if err != nil {
			return resources.Ami{}, fmt.Errorf("sharing snapshot with id %s with account %s: %v", *snapshotIDptr, account, err)
		}
	}

	if driverConfig.Encrypted {
		return resources.Ami{ID: *amiIDptr, Region: dstRegion}, nil
	}

	modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
		SnapshotId:    snapshotIDptr,
		Attribute:     ec2types.SnapshotAttributeNameCreateVolumePermission,
		OperationType: ec2types.OperationTypeAdd,
		GroupNames:    []string{"all"},
	}
	_, err = ec2Client.ModifySnapshotAttribute(ctx, modifySnapshotAttributeInput)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("making snapshot with id %s public: %v", *snapshotIDptr, err)
	}

	d.logger.Printf("snapshot %s is public\n", *snapshotIDptr)

	return resources.Ami{ID: *amiIDptr, Region: dstRegion}, nil
}
