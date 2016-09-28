package driver

import (
	"errors"
	"fmt"
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver/reqinputs"
	"light-stemcell-builder/resources"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
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
	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, "")).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	ec2Client := ec2.New(session.New(), awsConfig)
	return &SDKCreateAmiDriver{ec2Client: ec2Client, region: creds.Region, logger: logger}
}

// Create registers an AMI from an existing snapshot and optionally makes the AMI publically available
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
	case resources.PvAmiVirtualization:
		kernelID, err := d.findLatestKernelImage()
		if err != nil {
			return resources.Ami{}, fmt.Errorf("generating register image request for PV AMI: %s", err)
		}

		reqInput = reqinputs.NewPVAmiRequest(amiName, driverConfig.Description, driverConfig.SnapshotID, kernelID)
	case resources.HvmAmiVirtualization:
		reqInput = reqinputs.NewHVMAmiRequestInput(amiName, driverConfig.Description, driverConfig.SnapshotID)
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

	d.logger.Printf("waiting for AMI: %s to be available\n", *amiIDptr)
	err = d.ec2Client.WaitUntilImageAvailable(&ec2.DescribeImagesInput{
		ImageIds: []*string{amiIDptr},
	})
	if err != nil {
		return resources.Ami{}, fmt.Errorf("waiting for AMI %s to be available: %s", *amiIDptr, err)
	}

	if driverConfig.Accessibility == resources.PublicAmiAccessibility {
		d.logger.Printf("making AMI: %s public", *amiIDptr)
		d.ec2Client.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
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

	ami := resources.Ami{
		ID:                 *amiIDptr,
		Region:             d.region,
		VirtualizationType: driverConfig.VirtualizationType,
	}

	return ami, nil
}

func (d *SDKCreateAmiDriver) findLatestKernelImage() (string, error) {
	describeImagesOutput, err := d.ec2Client.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{aws.String(amazonOwner)},
		Filters: []*ec2.Filter{
			&ec2.Filter{
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

type imageList []*ec2.Image

func (l imageList) Len() int {
	return len(l)
}

func (l imageList) Less(i, j int) bool {
	iCreationTime, _ := time.Parse(time.RFC3339Nano, *l[i].CreationDate) // swallow error as not supported by sortable interface
	jCreationTime, _ := time.Parse(time.RFC3339Nano, *l[j].CreationDate) // swallow error as not supported by sortable interface
	return iCreationTime.After(jCreationTime)                            // ensure oldest time is first

}

func (l imageList) Swap(i, j int) {
	temp := l[i]
	l[i] = l[j]
	l[j] = temp
}
