package driver

import (
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

// SDKDeleteVolumeDriver handles deletion of a volume from a machine image on AWS
type SDKDeleteVolumeDriver struct {
	ec2Client *ec2.EC2
	logger    *log.Logger
}

// NewDeleteVolumeDriver deletes a previously created Volume
func NewDeleteVolumeDriver(logDest io.Writer, creds config.Credentials) *SDKDeleteVolumeDriver {
	logger := log.New(logDest, "SDKDeleteVolumeDriver ", log.LstdFlags)
	awsConfig := aws.NewConfig().
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	if d.creds.AccessKey != "" && d.creds.SecretKey != "" {
		awsConfig = awsConfig.WithCredentials(credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, ""))
	}

	ec2Client := ec2.New(session.New(), awsConfig)
	return &SDKDeleteVolumeDriver{ec2Client: ec2Client, logger: logger}
}

// Delete makes a request to delete the Volume
func (d *SDKDeleteVolumeDriver) Delete(volume resources.Volume) error {
	deleteStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Delete() in %f minutes\n", time.Since(startTime).Minutes())
	}(deleteStartTime)

	_, err := d.ec2Client.DeleteVolume(&ec2.DeleteVolumeInput{VolumeId: aws.String(volume.ID)})
	if err != nil {
		return err
	}
	return nil
}
