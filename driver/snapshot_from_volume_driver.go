package driver

import (
	"context"
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

var _ resources.SnapshotDriver = &SDKSnapshotFromVolumeDriver{}

// SDKSnapshotFromVolumeDriver creates an AMI from a previously created EBS volume
type SDKSnapshotFromVolumeDriver struct {
	ec2Client *ec2.Client
	logger    *log.Logger
}

// NewSnapshotFromVolumeDriver creates a NewSnapshotFromVolumeDriver for creating snapshots in EC2
func NewSnapshotFromVolumeDriver(logDest io.Writer, creds config.Credentials) *SDKSnapshotFromVolumeDriver {
	logger := log.New(logDest, "SDKSnapshotFromVolumeDriver ", log.LstdFlags)
	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)

	ec2Client := ec2.NewFromConfig(cfg)
	return &SDKSnapshotFromVolumeDriver{ec2Client: ec2Client, logger: logger}
}

// Create produces a snapshot in EC2 from a previously created EBS volume
func (d *SDKSnapshotFromVolumeDriver) Create(driverConfig resources.SnapshotDriverConfig) (resources.Snapshot, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	ctx := context.Background()

	d.logger.Printf("initiating CreateSnapshot task from volume: %s\n", driverConfig.VolumeID)
	reqOutput, err := d.ec2Client.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{
		VolumeId:    aws.String(driverConfig.VolumeID),
		Description: aws.String(fmt.Sprintf("bosh-light-stemcell-builder-%d", time.Now().UnixNano())),
	})
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("creating snapshot from EBS volume: %s: %s", driverConfig.VolumeID, err)
	}

	modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
		SnapshotId:    reqOutput.SnapshotId,
		Attribute:     ec2types.SnapshotAttributeNameCreateVolumePermission,
		OperationType: ec2types.OperationTypeAdd,
		GroupNames:    []string{"all"},
	}
	_, err = d.ec2Client.ModifySnapshotAttribute(ctx, modifySnapshotAttributeInput)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("making snapshot with id %s public: %s", *reqOutput.SnapshotId, err)
	}

	d.logger.Printf("waiting on snapshot %s to be completed\n", *reqOutput.SnapshotId)
	waitStartTime := time.Now()

	snapshotCompletedWaiter := ec2.NewSnapshotCompletedWaiter(d.ec2Client, func(o *ec2.SnapshotCompletedWaiterOptions) {
		o.MinDelay = 15 * time.Second
		o.MaxDelay = 15 * time.Second
	})
	err = snapshotCompletedWaiter.Wait(ctx, &ec2.DescribeSnapshotsInput{
		SnapshotIds: []string{*reqOutput.SnapshotId},
	}, 60*15*time.Second)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("waiting for snapshot to complete: %s", err)
	}

	d.logger.Printf("waited for snapshot %s completion for %f minutes\n", *reqOutput.SnapshotId, time.Since(waitStartTime).Minutes())
	d.logger.Printf("created snapshot %s\n", *reqOutput.SnapshotId)

	return resources.Snapshot{ID: *reqOutput.SnapshotId}, nil
}
