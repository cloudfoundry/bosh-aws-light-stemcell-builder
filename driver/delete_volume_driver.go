package driver

import (
	"context"
	"io"
	"log"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// SDKDeleteVolumeDriver handles deletion of a volume from a machine image on AWS
type SDKDeleteVolumeDriver struct {
	ec2Client *ec2.Client
	logger    *log.Logger
}

// NewDeleteVolumeDriver deletes a previously created Volume
func NewDeleteVolumeDriver(logDest io.Writer, creds config.Credentials) *SDKDeleteVolumeDriver {
	logger := log.New(logDest, "SDKDeleteVolumeDriver ", log.LstdFlags)
	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)

	ec2Client := ec2.NewFromConfig(cfg)
	return &SDKDeleteVolumeDriver{ec2Client: ec2Client, logger: logger}
}

// Delete makes a request to delete the Volume
func (d *SDKDeleteVolumeDriver) Delete(volume resources.Volume) error {
	deleteStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Delete() in %f minutes\n", time.Since(startTime).Minutes())
	}(deleteStartTime)

	_, err := d.ec2Client.DeleteVolume(context.Background(), &ec2.DeleteVolumeInput{VolumeId: aws.String(volume.ID)})
	if err != nil {
		return err
	}
	return nil
}
