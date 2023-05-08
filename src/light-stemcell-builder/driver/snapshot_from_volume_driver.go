package driver

import (
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
		WithCredentials(awsCreds(creds)).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	ec2Client := ec2.New(session.New(), awsConfig) //nolint:staticcheck
	return &SDKSnapshotFromVolumeDriver{ec2Client: ec2Client, logger: logger}
}

// Create produces a snapshot in EC2 from a previously created EBS volume
func (d *SDKSnapshotFromVolumeDriver) Create(driverConfig resources.SnapshotDriverConfig) (resources.Snapshot, error) {
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
		return resources.Snapshot{}, fmt.Errorf("creating snapshot from EBS volume: %s: %s", driverConfig.VolumeID, err)
	}

	modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
		SnapshotId:    reqOutput.SnapshotId,
		Attribute:     aws.String("createVolumePermission"),
		OperationType: aws.String("add"),
		GroupNames:    []*string{aws.String("all")},
	}
	_, err = d.ec2Client.ModifySnapshotAttribute(modifySnapshotAttributeInput)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("making snapshot with id %s public: %s", *reqOutput.SnapshotId, err)
	}

	d.logger.Printf("waiting on snapshot %s to be completed\n", *reqOutput.SnapshotId)
	waitStartTime := time.Now()
	err = d.waitUntilSnapshotCompleted(&ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{reqOutput.SnapshotId},
	}, d.ec2Client)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("waiting for snapshot to complete: %s", err)
	}

	d.logger.Printf("waited for snapshot %s completion for %f minutes\n", *reqOutput.SnapshotId, time.Since(waitStartTime).Minutes())
	d.logger.Printf("created snapshot %s\n", *reqOutput.SnapshotId)

	return resources.Snapshot{ID: *reqOutput.SnapshotId}, nil
}

func (d *SDKSnapshotFromVolumeDriver) waitUntilSnapshotCompleted(input *ec2.DescribeSnapshotsInput, c *ec2.EC2) error {
	ctx := aws.BackgroundContext()
	opts := []request.WaiterOption{
		request.WithWaiterDelay(request.ConstantWaiterDelay(15 * time.Second)),
		request.WithWaiterMaxAttempts(60),
	}
	return c.WaitUntilSnapshotCompletedWithContext(ctx, input, opts...)
}
