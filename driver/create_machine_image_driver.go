package driver

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// The SDKCreateMachineImageDriver uploads a machine image to S3 and creates a presigned URL for GET operations
type SDKCreateMachineImageDriver struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	logger        *log.Logger
}

// NewCreateMachineImageDriver creates a MachineImageDriver for S3 uploads
func NewCreateMachineImageDriver(logDest io.Writer, creds config.Credentials) *SDKCreateMachineImageDriver {
	logger := log.New(logDest, "SDKCreateMachineImageDriver ", log.LstdFlags)

	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)
	cfg.Retryer = func() aws.Retryer {
		return NewS3RetryerWithRetries(50).AsAWSRetryer()
	}

	s3Client := s3.NewFromConfig(cfg)

	return &SDKCreateMachineImageDriver{
		s3Client:      s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		logger:        logger,
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
	defer f.Close() //nolint:errcheck

	keyName := fmt.Sprintf("bosh-machine-image-%d", time.Now().UnixNano())
	d.logger.Printf("uploading image to s3://%s/%s\n", driverConfig.BucketName, keyName)

	ctx := context.Background()
	uploadStartTime := time.Now()
	uploader := manager.NewUploader(d.s3Client) //nolint:staticcheck
	input := &s3.PutObjectInput{
		Body:   f,
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	}
	if driverConfig.ServerSideEncryption != "" {
		input.ServerSideEncryption = s3types.ServerSideEncryption(driverConfig.ServerSideEncryption)
	}
	_, err = uploader.Upload(ctx, input) //nolint:staticcheck
	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("uploading machine image to S3: %s", err)
	}

	d.logger.Printf("finished uploaded image to s3 after %f minutes\n", time.Since(uploadStartTime).Minutes())

	machineImageGetURL := fmt.Sprintf("s3://%s/%s", driverConfig.BucketName, keyName)
	d.logger.Printf("generated GET URL %s\n", machineImageGetURL)

	deleteReq, err := d.presignClient.PresignDeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	}, s3.WithPresignExpires(24*time.Hour))
	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("failed to sign DELETE request: %s", err)
	}

	machineImageDeleteURL := deleteReq.URL
	d.logger.Printf("generated presigned DELETE URL %s\n", machineImageDeleteURL)

	machineImage := resources.MachineImage{
		GetURL:     machineImageGetURL,
		DeleteURLs: []string{machineImageDeleteURL},
	}

	return machineImage, nil
}
