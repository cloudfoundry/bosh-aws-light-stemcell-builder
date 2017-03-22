package driver

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
	"github.com/aws/aws-sdk-go/private/waiter"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var _ resources.SnapshotDriver = &SDKSnapshotFromImageDriver{}

// SDKSnapshotFromImageDriver creates an AMI directly from a machine image
type SDKSnapshotFromImageDriver struct {
	ec2Client *ec2.EC2
	logger    *log.Logger
}

// NewSnapshotFromImageDriver creates a SDKSnapshotFromImageDriver for creating snapshots in EC2
func NewSnapshotFromImageDriver(logDest io.Writer, creds config.Credentials) *SDKSnapshotFromImageDriver {
	logger := log.New(logDest, "SDKSnapshotFromImageDriver ", log.LstdFlags)
	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, "")).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	ec2Client := ec2.New(session.New(), awsConfig)
	return &SDKSnapshotFromImageDriver{ec2Client: ec2Client, logger: logger}
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
	err = d.waitUntilImportSnapshotTaskCompleted(taskFilter)
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

	return resources.Snapshot{ID: *snapshotIDptr}, nil
}

func (d *SDKSnapshotFromImageDriver) waitUntilImportSnapshotTaskCompleted(input *ec2.DescribeImportSnapshotTasksInput) error {
	waiterCfg := waiter.Config{
		Operation:   "DescribeImportSnapshotTasks",
		Delay:       30,
		MaxAttempts: 60,
		Acceptors: []waiter.WaitAcceptor{
			{
				State:    "success",
				Matcher:  "pathAll",
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "completed",
			},
			{
				State:    "failure",
				Matcher:  "pathAny",
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleted",
			},
			{
				State:    "failure",
				Matcher:  "pathAny",
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleting",
			},
		},
	}

	w := waiter.Waiter{
		Client: d.ec2Client,
		Input:  input,
		Config: waiterCfg,
	}
	return w.Wait()
}
