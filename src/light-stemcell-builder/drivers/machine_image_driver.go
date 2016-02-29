package drivers

import (
	"fmt"
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var _ resources.MachineImageDriver = &SDKMachineImageDriver{}

// The SDKMachineImageDriver uploads a machine image to S3 and creates a presigned URL for GET operations
type SDKMachineImageDriver struct {
	s3Client *s3.S3
	logger   *log.Logger
}

// NewMachineImageDriver creates a MachineImageDriver for S3 uploads
func NewMachineImageDriver(logDest io.Writer, creds config.Credentials) *SDKMachineImageDriver {
	logger := log.New(logDest, "SDKMachineImageDriver ", log.LstdFlags)

	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, "")).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	s3Session := session.New(awsConfig)
	s3Client := s3.New(s3Session)

	return &SDKMachineImageDriver{
		s3Client: s3Client,
		logger:   logger,
	}
}

// Create uploads a machine image to S3 and returns a presigned URL
func (d *SDKMachineImageDriver) Create(driverConfig resources.MachineImageDriverConfig) (string, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("completed Create() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("opening image for upload to S3: %s\n", driverConfig.MachineImagePath)

	f, err := os.Open(driverConfig.MachineImagePath)
	if err != nil {
		return "", fmt.Errorf("opening machine image for upload: %s", err)
	}

	keyName := fmt.Sprintf("bosh-machine-image-%d", time.Now().UnixNano())
	d.logger.Printf("uploading image to s3://%s/%s\n", driverConfig.BucketName, keyName)

	uploadStartTime := time.Now()
	uploader := s3manager.NewUploaderWithClient(d.s3Client)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Body:   f,
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	})

	if err != nil {
		return "", fmt.Errorf("uploading machine image to S3: %s", err)
	}

	d.logger.Printf("finished uploaded image to s3 after %f minutes\n", time.Since(uploadStartTime).Minutes())

	req, _ := d.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	})

	machineImageURL, err := req.Presign(2 * time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to sign GET request: %s", err)
	}

	d.logger.Printf("generated presigned GET URL %s\n", machineImageURL)

	return machineImageURL, nil
}
