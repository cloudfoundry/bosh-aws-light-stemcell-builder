package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver/reqinputs"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SDKCreateAmiDriver uses the AWS SDK to register an AMI from an existing snapshot in EC2
type SDKCreateAmiDriver struct {
	ec2Client *ec2.Client
	region    string
	logger    *log.Logger
}

// NewCreateAmiDriver creates a SDKCreateAmiDriver for an AMI from a snapshot in EC2
func NewCreateAmiDriver(logDest io.Writer, creds config.Credentials) *SDKCreateAmiDriver {
	logger := log.New(logDest, "SDKCreateAmiDriver ", log.LstdFlags)
	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)

	ec2Client := ec2.NewFromConfig(cfg)
	return &SDKCreateAmiDriver{ec2Client: ec2Client, region: creds.Region, logger: logger}
}

// Create registers an AMI from an existing snapshot and optionally makes the AMI publicly available
func (d *SDKCreateAmiDriver) Create(driverConfig resources.AmiDriverConfig) (resources.Ami, error) {
	var err error

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	ctx := context.Background()

	d.logger.Printf("creating AMI from snapshot: %s\n", driverConfig.SnapshotID)
	amiName := driverConfig.Name

	var reqInput *ec2.RegisterImageInput
	switch driverConfig.VirtualizationType {
	case resources.HvmAmiVirtualization:
		reqInput = reqinputs.NewHVMAmiRequestInput(amiName, driverConfig.Description, driverConfig.SnapshotID, driverConfig.Efi)
	}

	reqOutput, err := d.ec2Client.RegisterImage(ctx, reqInput)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("registering AMI: %s", err)
	}

	amiIDptr := reqOutput.ImageId
	if amiIDptr == nil {
		return resources.Ami{}, errors.New("AMI id nil")
	}

	d.logger.Printf("waiting for AMI: %s to exist\n", *amiIDptr)
	imageExistsWaiter := ec2.NewImageExistsWaiter(d.ec2Client)
	err = imageExistsWaiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{*amiIDptr},
	}, 10*time.Minute)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to exist: %s", *amiIDptr, err)
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
	_, err = d.ec2Client.CreateTags(ctx, tags)
	if err != nil {
		d.logger.Printf("Error tagging AMI: %s, Error: %s ", *amiIDptr, err.Error())
	}

	snapshotTags := &ec2.CreateTagsInput{
		Resources: []string{driverConfig.SnapshotID},
		Tags: []ec2types.Tag{
			{Key: aws.String("Name"), Value: name},
			{Key: aws.String("distro"), Value: distro},
			{Key: aws.String("version"), Value: version},
			{Key: aws.String("ami_id"), Value: amiIDptr},
		},
	}
	d.logger.Printf("tagging Snapshot: %s, with %v", driverConfig.SnapshotID, snapshotTags)
	_, err = d.ec2Client.CreateTags(ctx, snapshotTags)
	if err != nil {
		d.logger.Printf("Error tagging Snapshot: %s, Error: %s ", driverConfig.SnapshotID, err.Error())
	}

	for i := range driverConfig.SharedWithAccounts {
		account := driverConfig.SharedWithAccounts[i]

		_, err := d.ec2Client.ModifyImageAttribute(ctx, &ec2.ModifyImageAttributeInput{
			ImageId: amiIDptr,
			LaunchPermission: &ec2types.LaunchPermissionModifications{
				Add: []ec2types.LaunchPermission{
					{UserId: &account},
				},
			},
		})
		if err != nil {
			return resources.Ami{}, fmt.Errorf("failed to share AMI '%s' with account '%s': %w", *amiIDptr, account, err)
		}

		modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
			SnapshotId:    aws.String(driverConfig.SnapshotID),
			Attribute:     ec2types.SnapshotAttributeNameCreateVolumePermission,
			OperationType: ec2types.OperationTypeAdd,
			UserIds:       []string{account},
		}
		_, err = d.ec2Client.ModifySnapshotAttribute(ctx, modifySnapshotAttributeInput)
		if err != nil {
			return resources.Ami{}, fmt.Errorf("sharing snapshot with id %s with account %s: %v", driverConfig.SnapshotID, account, err)
		}
	}

	d.logger.Printf("waiting for AMI: %s to be available\n", *amiIDptr)
	imageAvailableWaiter := ec2.NewImageAvailableWaiter(d.ec2Client)
	err = imageAvailableWaiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{*amiIDptr},
	}, 30*time.Minute)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to be available: %s", *amiIDptr, err)
	}

	if driverConfig.Accessibility == resources.PublicAmiAccessibility {
		d.logger.Printf("making AMI: %s public", *amiIDptr)
		d.ec2Client.ModifyImageAttribute(ctx, &ec2.ModifyImageAttributeInput{ //nolint:errcheck
			ImageId: amiIDptr,
			LaunchPermission: &ec2types.LaunchPermissionModifications{
				Add: []ec2types.LaunchPermission{
					{Group: ec2types.PermissionGroupAll},
				},
			},
		})
	}

	ami := resources.Ami{
		ID:                 *amiIDptr,
		Region:             d.region,
		VirtualizationType: driverConfig.VirtualizationType,
	}

	return ami, nil
}

func (d *SDKCreateAmiDriver) findLatestKernelImage() (string, error) { //nolint:unused
	describeImagesOutput, err := d.ec2Client.DescribeImages(context.Background(), &ec2.DescribeImagesInput{
		Owners: []string{"amazon"},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"pv-grub-hd0_*-x86_64.gz"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("finding AKI for PV AMI: %s", err)
	}

	if len(describeImagesOutput.Images) == 0 {
		return "", errors.New("empty AKI list")
	}

	var images imageList = describeImagesOutput.Images
	sort.Sort(images)

	return *images[0].ImageId, nil
}

type imageList []ec2types.Image //nolint:unused

func (l imageList) Len() int { //nolint:unused
	return len(l)
}

func (l imageList) Less(i, j int) bool { //nolint:unused
	iCreationTime, _ := time.Parse(time.RFC3339Nano, *l[i].CreationDate) //nolint:errcheck
	jCreationTime, _ := time.Parse(time.RFC3339Nano, *l[j].CreationDate) //nolint:errcheck
	return iCreationTime.After(jCreationTime)
}

func (l imageList) Swap(i, j int) { //nolint:unused
	temp := l[i]
	l[i] = l[j]
	l[j] = temp
}
