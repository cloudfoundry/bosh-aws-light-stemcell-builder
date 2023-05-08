package driver

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver/manifests"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// SDKCreateVolumeDriver is an implementation of the resources VolumeDriver that
// handles creation of a volume from a machine image on AWS
type SDKCreateVolumeDriver struct {
	ec2Client *ec2.EC2
	logger    *log.Logger
}

// NewCreateVolumeDriver creates a SDKCreateVolumeDriver for importing a volume from a machine image url
func NewCreateVolumeDriver(logDest io.Writer, creds config.Credentials) *SDKCreateVolumeDriver {
	logger := log.New(logDest, "SDKCreateVolumeDriver ", log.LstdFlags)
	awsConfig := aws.NewConfig().
		WithCredentials(awsCreds(creds)).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	newSession, _ := session.NewSession() //nolint:errcheck
	ec2Client := ec2.New(newSession, awsConfig)
	return &SDKCreateVolumeDriver{ec2Client: ec2Client, logger: logger}
}

// Create makes an EBS volume from a machine image URL in the first availability zone returned from DescribeAvailabilityZones
func (d *SDKCreateVolumeDriver) Create(driverConfig resources.VolumeDriverConfig) (resources.Volume, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	availabilityZoneOutput, err := d.ec2Client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("state"), Values: []*string{aws.String("available")}},
		},
	})
	if err != nil {
		return resources.Volume{}, fmt.Errorf("listing availability zones: %s", err)
	}

	if len(availabilityZoneOutput.AvailabilityZones) == 0 {
		return resources.Volume{}, fmt.Errorf("finding any available availability zones in region %s", *d.ec2Client.Config.Region)
	}

	availabilityZone := availabilityZoneOutput.AvailabilityZones[0].ZoneName
	fetchManifestResp, err := http.Get(driverConfig.MachineImageManifestURL)
	if err != nil {
		return resources.Volume{}, fmt.Errorf("fetching import volume manifest: %s", err)
	}

	defer fetchManifestResp.Body.Close() //nolint:errcheck
	manifestBytes, err := io.ReadAll(fetchManifestResp.Body)
	if err != nil {
		return resources.Volume{}, fmt.Errorf("reading import volume manifest from response: %s", err)
	}
	if fetchManifestResp.StatusCode < 200 || fetchManifestResp.StatusCode >= 300 {
		return resources.Volume{}, fmt.Errorf("Received invalid response code '%d' fetching resource '%s': %s",
			fetchManifestResp.StatusCode,
			driverConfig.MachineImageManifestURL,
			manifestBytes)
	}

	m := manifests.ImportVolumeManifest{}

	err = xml.Unmarshal(manifestBytes, &m)
	if err != nil {
		return resources.Volume{}, fmt.Errorf("deserializing import volume manifest. Bytes:\n%s\nError: %s", manifestBytes, err)
	}

	reqOutput, err := d.ec2Client.ImportVolume(&ec2.ImportVolumeInput{
		AvailabilityZone: availabilityZone,
		Image: &ec2.DiskImageDetail{
			ImportManifestUrl: aws.String(driverConfig.MachineImageManifestURL),
			Format:            aws.String(strings.ToUpper(m.FileFormat)),
			Bytes:             aws.Int64(m.VolumeSizeGB),
		},
		Volume: &ec2.VolumeDetail{
			Size: aws.Int64(m.VolumeSizeGB),
		},
	})

	if err != nil {
		return resources.Volume{}, fmt.Errorf("creating import volume task: %s", err)
	}

	conversionTaskIDptr := reqOutput.ConversionTask.ConversionTaskId
	if conversionTaskIDptr == nil {
		return resources.Volume{}, fmt.Errorf("conversion task ID nil")
	}

	d.logger.Printf("waiting on ImportVolume task %s\n", *conversionTaskIDptr)

	taskFilter := &ec2.DescribeConversionTasksInput{
		ConversionTaskIds: []*string{conversionTaskIDptr},
	}

	waitStartTime := time.Now()
	err = d.waitUntilImageConversionTaskCompleted(taskFilter, d.ec2Client)
	d.logger.Printf("waited on import task %s for %f minutes\n", *conversionTaskIDptr, time.Since(waitStartTime).Minutes())

	if err != nil {
		return resources.Volume{}, fmt.Errorf("waiting for volume to be imported: %s", err)
	}

	taskOutput, err := d.ec2Client.DescribeConversionTasks(taskFilter)
	if err != nil {
		return resources.Volume{}, fmt.Errorf("fetching volume ID from conversion task %s", *conversionTaskIDptr)
	}

	volumeIDptr := taskOutput.ConversionTasks[0].ImportVolume.Volume.Id
	if volumeIDptr == nil {
		return resources.Volume{}, fmt.Errorf("volume ID nil")
	}

	d.logger.Printf("waiting for volume to be available: %s\n", *volumeIDptr)
	waitStartTime = time.Now()
	err = d.ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{volumeIDptr}}) //nolint:ineffassign,staticcheck
	d.logger.Printf("waited on volume %s for %f seconds\n", *volumeIDptr, time.Since(waitStartTime).Seconds())

	return resources.Volume{ID: *volumeIDptr}, nil
}

func (d *SDKCreateVolumeDriver) waitUntilImageConversionTaskCompleted(input *ec2.DescribeConversionTasksInput, c *ec2.EC2) error {
	ctx := aws.BackgroundContext()
	opts := []request.WaiterOption{
		request.WithWaiterDelay(request.ConstantWaiterDelay(15 * time.Second)),
		request.WithWaiterMaxAttempts(120),
	}
	return c.WaitUntilConversionTaskCompletedWithContext(ctx, input, opts...)
}
