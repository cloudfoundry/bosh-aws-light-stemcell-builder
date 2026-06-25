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

var _ resources.SnapshotDriver = &SDKSnapshotFromImageDriver{}

// SDKSnapshotFromImageDriver creates an AMI directly from a machine image
type SDKSnapshotFromImageDriver struct {
	ec2Client *ec2.Client
	logger    *log.Logger
}

// NewSnapshotFromImageDriver creates a SDKSnapshotFromImageDriver for creating snapshots in EC2
func NewSnapshotFromImageDriver(logDest io.Writer, creds config.Credentials) *SDKSnapshotFromImageDriver {
	logger := log.New(logDest, "SDKSnapshotFromImageDriver ", log.LstdFlags)
	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)

	ec2Client := ec2.NewFromConfig(cfg)
	return &SDKSnapshotFromImageDriver{ec2Client: ec2Client, logger: logger}
}

// Create produces a snapshot in EC2 from a machine image previously uploaded to S3
func (d *SDKSnapshotFromImageDriver) Create(driverConfig resources.SnapshotDriverConfig) (resources.Snapshot, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	ctx := context.Background()

	d.logger.Printf("initiating ImportSnapshot task from image: %s\n", driverConfig.MachineImageURL)

	input := &ec2.ImportSnapshotInput{
		DiskContainer: &ec2types.SnapshotDiskContainer{
			Url:    &driverConfig.MachineImageURL,
			Format: aws.String(driverConfig.FileFormat),
		},
		Encrypted: &driverConfig.AmiProperties.Encrypted, //nolint:staticcheck
	}

	if driverConfig.KmsAlias.ARN != "" { //nolint:staticcheck
		input.KmsKeyId = &driverConfig.KmsAlias.ARN //nolint:staticcheck
	}

	reqOutput, err := d.ec2Client.ImportSnapshot(ctx, input)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("creating import snapshot task: %s", err)
	}

	d.logger.Printf("waiting on ImportSnapshot task %s\n", *reqOutput.ImportTaskId)

	taskFilter := &ec2.DescribeImportSnapshotTasksInput{
		ImportTaskIds: []string{*reqOutput.ImportTaskId},
	}

	waitStartTime := time.Now()
	err = d.waitUntilImportSnapshotTaskCompleted(ctx, taskFilter)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("waiting for snapshot to become available: %s", err)
	}

	d.logger.Printf("waited on import task %s for %f minutes\n", *reqOutput.ImportTaskId, time.Since(waitStartTime).Minutes())

	describeOutput, err := d.ec2Client.DescribeImportSnapshotTasks(ctx, taskFilter)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("describing snapshot from import snapshot task %s: %s", *reqOutput.ImportTaskId, err)
	}

	snapshotIDptr := describeOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId
	if snapshotIDptr == nil {
		return resources.Snapshot{}, fmt.Errorf("snapshot ID empty for import task: %s", *reqOutput.ImportTaskId)
	}

	d.logger.Printf("created snapshot %s\n", *snapshotIDptr)

	if driverConfig.Accessibility != resources.PrivateAmiAccessibility {
		modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
			SnapshotId:    snapshotIDptr,
			Attribute:     ec2types.SnapshotAttributeNameCreateVolumePermission,
			OperationType: ec2types.OperationTypeAdd,
			GroupNames:    []string{"all"},
		}
		_, err = d.ec2Client.ModifySnapshotAttribute(ctx, modifySnapshotAttributeInput)
		if err != nil {
			return resources.Snapshot{}, fmt.Errorf("making snapshot with id %s public: %s", *snapshotIDptr, err)
		}

		d.logger.Printf("snapshot %s is public\n", *snapshotIDptr)
	}

	return resources.Snapshot{ID: *snapshotIDptr}, nil
}

// waitUntilImportSnapshotTaskCompleted polls until all import snapshot tasks are completed.
func (d *SDKSnapshotFromImageDriver) waitUntilImportSnapshotTaskCompleted(ctx context.Context, input *ec2.DescribeImportSnapshotTasksInput) error {
	const (
		maxAttempts  = 60
		pollInterval = 60 * time.Second
	)

	for i := 0; i < maxAttempts; i++ {
		output, err := d.ec2Client.DescribeImportSnapshotTasks(ctx, input)
		if err != nil {
			return fmt.Errorf("describing import snapshot tasks: %s", err)
		}

		allCompleted := true
		for _, task := range output.ImportSnapshotTasks {
			status := ""
			if task.SnapshotTaskDetail != nil && task.SnapshotTaskDetail.Status != nil {
				status = *task.SnapshotTaskDetail.Status
			}
			switch status {
			case "completed":
				// this task is done
			case "deleted", "deleting":
				return fmt.Errorf("import snapshot task %s is in state %s", aws.ToString(task.ImportTaskId), status)
			default:
				allCompleted = false
			}
		}

		if allCompleted {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timed out waiting for import snapshot tasks to complete after %d attempts", maxAttempts)
}
