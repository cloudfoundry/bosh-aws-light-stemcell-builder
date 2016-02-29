package drivers

import (
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

var _ resources.SnapshotDriver = &SDKSnapshotFromVolumeDriver{}

// SDKSnapshotFromVolumeDriver creates an AMI from a previously created EBS volume
type SDKSnapshotFromVolumeDriver struct {
	ec2Client *ec2.EC2
	logger    *log.Logger
}

// NewSnapshotFromVolumeDriver creates a NewSnapshotFromVolumeDriver for creating snapshots in EC2
func NewSnapshotFromVolumeDriver(logDest io.Writer, creds config.Credentials) *SDKSnapshotFromVolumeDriver {
	logger := log.New(logDest, "SDKSnapshotFromVolumeDriver ", log.LstdFlags)
	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, "")).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	ec2Client := ec2.New(session.New(), awsConfig)
	return &SDKSnapshotFromVolumeDriver{ec2Client: ec2Client, logger: logger}
}

// Create produces a snapshot in EC2 from a previoulsy created EBS volume
func (d *SDKSnapshotFromVolumeDriver) Create(driverConfig resources.SnapshotDriverConfig) (string, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("initiating CreateSnapshot task from volume: %s\n", driverConfig.VolumeID)
	reqOutput, err := d.ec2Client.CreateSnapshot(&ec2.CreateSnapshotInput{
		VolumeId:    aws.String(driverConfig.VolumeID),
		Description: aws.String(fmt.Sprintf("bosh-light-stemcell-builder-%d", time.Now().UnixNano())),
	})
	if err != nil {
		return "", fmt.Errorf("creating snapshot from EBS volume: %s: %s", driverConfig.VolumeID, err)
	}

	d.logger.Printf("waiting on snapshot %s to be completed\n", *reqOutput.SnapshotId)
	waitStartTime := time.Now()
	err = d.ec2Client.WaitUntilSnapshotCompleted(&ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{reqOutput.SnapshotId},
	})
	if err != nil {
		return "", fmt.Errorf("waiting for snapshot to complete: %s", err)
	}

	d.logger.Printf("waited for snapshot %s completion for %f minutes\n", *reqOutput.SnapshotId, time.Since(waitStartTime).Minutes())
	d.logger.Printf("created snapshot %s\n", *reqOutput.SnapshotId)

	return *reqOutput.SnapshotId, nil
}
