package driver

import (
	"context"
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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SDKCreateVolumeDriver is an implementation of the resources VolumeDriver that
// handles creation of a volume from a machine image on AWS
type SDKCreateVolumeDriver struct {
	ec2Client *ec2.Client
	region    string
	logger    *log.Logger
}

// NewCreateVolumeDriver creates a SDKCreateVolumeDriver for importing a volume from a machine image url
func NewCreateVolumeDriver(logDest io.Writer, creds config.Credentials) *SDKCreateVolumeDriver {
	logger := log.New(logDest, "SDKCreateVolumeDriver ", log.LstdFlags)
	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)

	ec2Client := ec2.NewFromConfig(cfg)
	return &SDKCreateVolumeDriver{ec2Client: ec2Client, region: creds.Region, logger: logger}
}

// Create makes an EBS volume from a machine image URL in the first availability zone returned from DescribeAvailabilityZones
func (d *SDKCreateVolumeDriver) Create(driverConfig resources.VolumeDriverConfig) (resources.Volume, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	ctx := context.Background()

	availabilityZoneOutput, err := d.ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("state"), Values: []string{"available"}},
		},
	})
	if err != nil {
		return resources.Volume{}, fmt.Errorf("listing availability zones: %s", err)
	}

	if len(availabilityZoneOutput.AvailabilityZones) == 0 {
		return resources.Volume{}, fmt.Errorf("finding any available availability zones in region %s", d.region)
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
		return resources.Volume{}, fmt.Errorf("Received invalid response code '%d' fetching resource '%s': %s", //nolint:staticcheck
			fetchManifestResp.StatusCode,
			driverConfig.MachineImageManifestURL,
			manifestBytes)
	}

	m := manifests.ImportVolumeManifest{}

	err = xml.Unmarshal(manifestBytes, &m)
	if err != nil {
		return resources.Volume{}, fmt.Errorf("deserializing import volume manifest. Bytes:\n%s\nError: %s", manifestBytes, err)
	}

	reqOutput, err := d.ec2Client.ImportVolume(ctx, &ec2.ImportVolumeInput{
		AvailabilityZone: availabilityZone,
		Image: &ec2types.DiskImageDetail{
			ImportManifestUrl: aws.String(driverConfig.MachineImageManifestURL),
			Format:            ec2types.DiskImageFormat(strings.ToUpper(m.FileFormat)),
			Bytes:             aws.Int64(m.VolumeSizeGB),
		},
		Volume: &ec2types.VolumeDetail{
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
		ConversionTaskIds: []string{*conversionTaskIDptr},
	}

	waitStartTime := time.Now()
	err = d.waitUntilConversionTaskCompleted(ctx, taskFilter)
	d.logger.Printf("waited on import task %s for %f minutes\n", *conversionTaskIDptr, time.Since(waitStartTime).Minutes())

	if err != nil {
		return resources.Volume{}, fmt.Errorf("waiting for volume to be imported: %s", err)
	}

	taskOutput, err := d.ec2Client.DescribeConversionTasks(ctx, taskFilter)
	if err != nil {
		return resources.Volume{}, fmt.Errorf("fetching volume ID from conversion task %s", *conversionTaskIDptr)
	}

	volumeIDptr := taskOutput.ConversionTasks[0].ImportVolume.Volume.Id
	if volumeIDptr == nil {
		return resources.Volume{}, fmt.Errorf("volume ID nil")
	}

	d.logger.Printf("waiting for volume to be available: %s\n", *volumeIDptr)
	waitStartTime = time.Now()
	volumeAvailableWaiter := ec2.NewVolumeAvailableWaiter(d.ec2Client)
	err = volumeAvailableWaiter.Wait(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{*volumeIDptr}}, 30*time.Minute) //nolint:ineffassign,staticcheck
	d.logger.Printf("waited on volume %s for %f seconds\n", *volumeIDptr, time.Since(waitStartTime).Seconds())

	return resources.Volume{ID: *volumeIDptr}, nil
}

// waitUntilConversionTaskCompleted polls until the conversion task is complete.
func (d *SDKCreateVolumeDriver) waitUntilConversionTaskCompleted(ctx context.Context, input *ec2.DescribeConversionTasksInput) error {
	const (
		maxAttempts  = 120
		pollInterval = 15 * time.Second
	)

	for i := 0; i < maxAttempts; i++ {
		output, err := d.ec2Client.DescribeConversionTasks(ctx, input)
		if err != nil {
			return fmt.Errorf("describing conversion tasks: %s", err)
		}

		if len(output.ConversionTasks) == 0 {
			return fmt.Errorf("no conversion tasks found")
		}

		task := output.ConversionTasks[0]
		state := task.State
		switch state {
		case ec2types.ConversionTaskStateCompleted:
			return nil
		case ec2types.ConversionTaskStateCancelled, ec2types.ConversionTaskStateCancelling:
			return fmt.Errorf("conversion task %s is in state %s", *task.ConversionTaskId, state)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timed out waiting for conversion task to complete after %d attempts", maxAttempts)
}
