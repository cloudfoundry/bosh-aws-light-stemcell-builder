package driver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver/reqinputs"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	firstDeviceNameHVMAmi = "/dev/xvda"
	firstDeviceNamePVAmi  = "/dev/sda"
	publicGroup           = "all"
	amazonOwner           = "amazon"
)

// SDKCreateAmiDriver uses the AWS SDK to register an AMI from an existing snapshot in EC2
type SDKCreateAmiDriver struct {
	ec2Client *ec2.EC2
	region    string
	logger    *log.Logger
}

// NewCreateAmiDriver creates a SDKCreateAmiDriver for an AMI from a snapshot in EC2
func NewCreateAmiDriver(logDest io.Writer, creds config.Credentials) *SDKCreateAmiDriver {
	logger := log.New(logDest, "SDKCreateAmiDriver ", log.LstdFlags)
	awsConfig := creds.GetAwsConfig().
		WithLogger(newDriverLogger(logger))

	ec2Client := ec2.New(session.Must(session.NewSession(awsConfig)))
	return &SDKCreateAmiDriver{ec2Client: ec2Client, region: creds.Region, logger: logger}
}

// Create registers an AMI from an existing snapshot and optionally makes the AMI publicly available
func (d *SDKCreateAmiDriver) Create(driverConfig resources.AmiDriverConfig) (resources.Ami, error) {
	var err error

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("creating AMI from snapshot: %s\n", driverConfig.SnapshotID)
	amiName := driverConfig.Name

	var reqInput *ec2.RegisterImageInput
	switch driverConfig.VirtualizationType {
	case resources.HvmAmiVirtualization:
		reqInput = reqinputs.NewHVMAmiRequestInput(amiName, driverConfig.Description, driverConfig.SnapshotID, driverConfig.AmiProperties.Efi) //nolint:staticcheck
	}

	reqOutput, err := d.ec2Client.RegisterImage(reqInput)
	if err != nil {
		return resources.Ami{}, fmt.Errorf("registering AMI: %s", err)
	}

	amiIDptr := reqOutput.ImageId
	if amiIDptr == nil {
		return resources.Ami{}, errors.New("AMI id nil")
	}

	d.logger.Printf("waiting for AMI: %s to exist\n", *amiIDptr)
	err = d.ec2Client.WaitUntilImageExists(&ec2.DescribeImagesInput{
		ImageIds: []*string{amiIDptr},
	})
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to exist: %s", *amiIDptr, err)
	}
	name := aws.String(driverConfig.AmiProperties.Tags["distro"] + "-" + driverConfig.AmiProperties.Tags["version"]) //nolint:staticcheck
	distro := aws.String(driverConfig.AmiProperties.Tags["distro"])                                                  //nolint:staticcheck
	version := aws.String(driverConfig.AmiProperties.Tags["version"])                                                //nolint:staticcheck
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
	_, err = d.ec2Client.CreateTags(tags)
	if err != nil {
		d.logger.Printf("Error tagging AMI: %s, Error: %s ", *amiIDptr, err.Error())
	}

	tags = &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(driverConfig.SnapshotID),
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
				Key:   aws.String("ami_id"),
				Value: amiIDptr,
			},
		},
	}
	d.logger.Printf("tagging Snapshot: %s, with %s", driverConfig.SnapshotID, tags)
	_, err = d.ec2Client.CreateTags(tags)
	if err != nil {
		d.logger.Printf("Error tagging Snapshot: %s, Error: %s ", driverConfig.SnapshotID, err.Error())
	}

	for i := range driverConfig.SharedWithAccounts {
		account := driverConfig.SharedWithAccounts[i]

		_, err := d.ec2Client.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
			ImageId: amiIDptr,
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: []*ec2.LaunchPermission{
					{
						UserId: &account,
					},
				},
			},
		})
		if err != nil {
			return resources.Ami{}, fmt.Errorf("failed to share AMI '%s' with account '%s': %w", *amiIDptr, account, err)
		}

		modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
			SnapshotId:    aws.String(driverConfig.SnapshotID),
			Attribute:     aws.String("createVolumePermission"),
			OperationType: aws.String("add"),
			UserIds: []*string{
				aws.String(account),
			},
		}
		_, err = d.ec2Client.ModifySnapshotAttribute(modifySnapshotAttributeInput)
		if err != nil {
			return resources.Ami{}, fmt.Errorf("sharing snapshot with id %s with account %s: %v", driverConfig.SnapshotID, account, err)
		}
	}

	d.logger.Printf("waiting for AMI: %s to be available\n", *amiIDptr)
	err = d.ec2Client.WaitUntilImageAvailable(&ec2.DescribeImagesInput{
		ImageIds: []*string{amiIDptr},
	})
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to be available: %s", *amiIDptr, err)
	}

	if driverConfig.Accessibility == resources.PublicAmiAccessibility {
		d.logger.Printf("making AMI: %s public", *amiIDptr)
		d.ec2Client.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{ //nolint:errcheck
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

	ami := resources.Ami{
		ID:                 *amiIDptr,
		Region:             d.region,
		VirtualizationType: driverConfig.VirtualizationType,
	}

	return ami, nil
}

func (d *SDKCreateAmiDriver) findLatestKernelImage() (string, error) { //nolint:unused
	describeImagesOutput, err := d.ec2Client.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{aws.String(amazonOwner)},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String("pv-grub-hd0_*-x86_64.gz")},
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

type imageList []*ec2.Image //nolint:unused

func (l imageList) Len() int { //nolint:unused
	return len(l)
}

func (l imageList) Less(i, j int) bool { //nolint:unused
	// swallow error as not supported by sortable interface
	iCreationTime, _ := time.Parse(time.RFC3339Nano, *l[i].CreationDate) //nolint:errcheck
	jCreationTime, _ := time.Parse(time.RFC3339Nano, *l[j].CreationDate) //nolint:errcheck
	return iCreationTime.After(jCreationTime)                            // ensure the oldest time is first

}

func (l imageList) Swap(i, j int) { //nolint:unused
	temp := l[i]
	l[i] = l[j]
	l[j] = temp
}
