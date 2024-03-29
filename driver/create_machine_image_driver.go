package driver

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// The SDKCreateMachineImageDriver uploads a machine image to S3 and creates a presigned URL for GET operations
type SDKCreateMachineImageDriver struct {
	s3Client *s3.S3
	logger   *log.Logger
}

// NewCreateMachineImageDriver creates a MachineImageDriver for S3 uploads
func NewCreateMachineImageDriver(logDest io.Writer, creds config.Credentials) *SDKCreateMachineImageDriver {
	logger := log.New(logDest, "SDKCreateMachineImageDriver ", log.LstdFlags)

	awsConfig := creds.GetAwsConfig().
		WithLogger(newDriverLogger(logger))

	awsConfig.Retryer = NewS3RetryerWithRetries(50)

	s3Client := s3.New(session.Must(session.NewSession(awsConfig)))

	return &SDKCreateMachineImageDriver{
		s3Client: s3Client,
		logger:   logger,
	}
}

// Create uploads a machine image to S3 and returns a presigned URL
func (d *SDKCreateMachineImageDriver) Create(driverConfig resources.MachineImageDriverConfig) (resources.MachineImage, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("opening image for upload to S3: %s\n", driverConfig.MachineImagePath)

	f, err := os.Open(driverConfig.MachineImagePath)
	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("opening machine image for upload: %s", err)
	}

	keyName := fmt.Sprintf("bosh-machine-image-%d", time.Now().UnixNano())
	d.logger.Printf("uploading image to s3://%s/%s\n", driverConfig.BucketName, keyName)

	uploadStartTime := time.Now()
	uploader := s3manager.NewUploaderWithClient(d.s3Client)
	input := &s3manager.UploadInput{
		Body:   f,
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	}
	if driverConfig.ServerSideEncryption != "" {
		input.ServerSideEncryption = aws.String(driverConfig.ServerSideEncryption)
	}
	_, err = uploader.Upload(input)

	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("uploading machine image to S3: %s", err)
	}

	d.logger.Printf("finished uploaded image to s3 after %f minutes\n", time.Since(uploadStartTime).Minutes())

	machineImageGetURL := fmt.Sprintf("s3://%s/%s", driverConfig.BucketName, keyName)
	d.logger.Printf("generated GET URL %s\n", machineImageGetURL)

	deleteReq, _ := d.s3Client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	})

	machineImageDeleteURL, err := deleteReq.Presign(24 * time.Hour)
	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("failed to sign DELETE request: %s", err)
	}

	d.logger.Printf("generated presigned DELETE URL %s\n", machineImageDeleteURL)

	machineImage := resources.MachineImage{
		GetURL:     machineImageGetURL,
		DeleteURLs: []string{machineImageDeleteURL},
	}

	return machineImage, nil
}
