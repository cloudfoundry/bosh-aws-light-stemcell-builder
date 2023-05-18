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

var _ resources.SnapshotDriver = &SDKSnapshotFromImageDriver{}

// SDKSnapshotFromImageDriver creates an AMI directly from a machine image
type SDKSnapshotFromImageDriver struct {
	ec2Client *ec2.EC2
	logger    *log.Logger
}

// NewSnapshotFromImageDriver creates a SDKSnapshotFromImageDriver for creating snapshots in EC2
func NewSnapshotFromImageDriver(logDest io.Writer, awsRegionSession *session.Session, creds config.Credentials) *SDKSnapshotFromImageDriver {
	logger := log.New(logDest, "SDKSnapshotFromImageDriver ", log.LstdFlags)

	ec2Client := ec2.New(awsRegionSession)

	return &SDKSnapshotFromImageDriver{
		ec2Client: ec2Client,
		logger:    logger,
	}
}

// Create produces a snapshot in EC2 from a machine image previously uploaded to S3
func (d *SDKSnapshotFromImageDriver) Create(driverConfig resources.SnapshotDriverConfig) (resources.Snapshot, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("initiating ImportSnapshot task from image: %s\n", driverConfig.MachineImageURL)
	reqOutput, err := d.ec2Client.ImportSnapshot(&ec2.ImportSnapshotInput{
		DiskContainer: &ec2.SnapshotDiskContainer{
			Url:    &driverConfig.MachineImageURL,
			Format: aws.String(driverConfig.FileFormat),
		},
	})
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("creating import snapshot task: %s", err)
	}

	d.logger.Printf("waiting on ImportSnapshot task %s\n", *reqOutput.ImportTaskId)

	taskFilter := &ec2.DescribeImportSnapshotTasksInput{
		ImportTaskIds: []*string{reqOutput.ImportTaskId},
	}

	waitStartTime := time.Now()
	err = d.waitUntilImportSnapshotTaskCompleted(taskFilter, d.ec2Client)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("waiting for snapshot to become available: %s", err)
	}

	d.logger.Printf("waited on import task %s for %f minutes\n", *reqOutput.ImportTaskId, time.Since(waitStartTime).Minutes())

	describeOutput, err := d.ec2Client.DescribeImportSnapshotTasks(taskFilter)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("describing snapshot from import snapshot task %s: %s", *reqOutput.ImportTaskId, err)
	}

	snapshotIDptr := describeOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId
	if snapshotIDptr == nil {
		return resources.Snapshot{}, fmt.Errorf("snapshot ID empty for import task: %s", *reqOutput.ImportTaskId)
	}

	d.logger.Printf("created snapshot %s\n", *snapshotIDptr)

	modifySnapshotAttributeInput := &ec2.ModifySnapshotAttributeInput{
		SnapshotId:    snapshotIDptr,
		Attribute:     aws.String("createVolumePermission"),
		OperationType: aws.String("add"),
		GroupNames:    []*string{aws.String("all")},
	}
	_, err = d.ec2Client.ModifySnapshotAttribute(modifySnapshotAttributeInput)
	if err != nil {
		return resources.Snapshot{}, fmt.Errorf("making snapshot with id %s public: %s", *snapshotIDptr, err)
	}

	d.logger.Printf("snapshot %s is public\n", *snapshotIDptr)

	return resources.Snapshot{ID: *snapshotIDptr}, nil
}

func (d *SDKSnapshotFromImageDriver) waitUntilImportSnapshotTaskCompleted(input *ec2.DescribeImportSnapshotTasksInput, c *ec2.EC2) error {
	ctx := aws.BackgroundContext()
	w := request.Waiter{
		Name:        "WaitUntilImportSnapshotTasksCompleted",
		MaxAttempts: 60,
		Delay:       request.ConstantWaiterDelay(60 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:    request.SuccessWaiterState,
				Matcher:  request.PathAllWaiterMatch,
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "completed",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.PathAnyWaiterMatch,
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleted",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.PathAnyWaiterMatch,
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleting",
			},
		},
		Logger: c.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *ec2.DescribeImportSnapshotTasksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := c.DescribeImportSnapshotTasksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}

	return w.WaitWithContext(ctx)
}
