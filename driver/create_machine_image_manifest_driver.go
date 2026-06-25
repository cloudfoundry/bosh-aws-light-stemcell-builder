package driver

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driver/manifests"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const gbInBytes = 1 << 30

// The SDKCreateMachineImageManifestDriver uploads a machine image to S3 and creates an import volume manifest
type SDKCreateMachineImageManifestDriver struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	logger        *log.Logger
	genManifest   bool //nolint:unused
}

// NewCreateMachineImageManifestDriver creates a MachineImageDriver machine image manifest generation
func NewCreateMachineImageManifestDriver(logDest io.Writer, creds config.Credentials) *SDKCreateMachineImageManifestDriver {
	logger := log.New(logDest, "SDKCreateMachineImageManifestDriver ", log.LstdFlags)

	cfg := creds.GetAwsConfig()
	cfg.Logger = newDriverLogger(logger)
	cfg.Retryer = func() aws.Retryer {
		return NewS3RetryerWithRetries(50).AsAWSRetryer()
	}

	s3Client := s3.NewFromConfig(cfg)

	return &SDKCreateMachineImageManifestDriver{
		s3Client:      s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		logger:        logger,
	}
}

// Create uploads a machine image to S3 and returns a presigned URL to an import volume manifest
func (d *SDKCreateMachineImageManifestDriver) Create(driverConfig resources.MachineImageDriverConfig) (resources.MachineImage, error) {
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

	headReqOutput, err := d.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	})
	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("fetching properties for uploaded machine image: %s in bucket: %s: %s", keyName, driverConfig.BucketName, err)
	}

	sizeInBytesPtr := headReqOutput.ContentLength
	if sizeInBytesPtr == nil {
		return resources.MachineImage{}, errors.New("size in bytes nil")
	}

	volumeSizeGB := driverConfig.VolumeSizeGB
	if volumeSizeGB == 0 {
		volumeSizeGB = int64(math.Ceil(float64(*sizeInBytesPtr) / gbInBytes))
	}

	m, err := d.generateManifest(ctx, driverConfig.BucketName, keyName, *sizeInBytesPtr, volumeSizeGB, driverConfig.FileFormat)
	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("Failed to generate machine image manifest: %s", err) //nolint:staticcheck
	}

	manifestURL, err := d.uploadManifest(ctx, driverConfig.BucketName, driverConfig.ServerSideEncryption, m) //nolint:ineffassign,staticcheck

	machineImage := resources.MachineImage{
		GetURL:     manifestURL,
		DeleteURLs: []string{m.SelfDestructURL, m.Parts.Part.DeleteURL},
	}

	return machineImage, nil
}

func (d *SDKCreateMachineImageManifestDriver) generateManifest(ctx context.Context, bucketName string, keyName string, sizeInBytes int64, volumeSizeGB int64, fileFormat string) (*manifests.ImportVolumeManifest, error) {
	getReq, err := d.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	}, s3.WithPresignExpires(2*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to sign GET request: %s", err)
	}
	presignedGetURL := getReq.URL

	d.logger.Printf("generated presigned GET URL %s\n", presignedGetURL)

	headReq, err := d.presignClient.PresignHeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	}, s3.WithPresignExpires(1*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to sign HEAD request: %s", err)
	}
	presignedHeadURL := headReq.URL

	d.logger.Printf("generated presigned HEAD URL %s\n", presignedHeadURL)

	deleteReq, err := d.presignClient.PresignDeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	}, s3.WithPresignExpires(1*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to sign DELETE request: %s", err)
	}
	presignedDeleteURL := deleteReq.URL

	d.logger.Printf("generated presigned DELETE URL %s\n", presignedDeleteURL)

	imageProps := manifests.MachineImageProperties{
		KeyName:      keyName,
		HeadURL:      presignedHeadURL,
		GetURL:       presignedGetURL,
		DeleteURL:    presignedDeleteURL,
		SizeBytes:    sizeInBytes,
		VolumeSizeGB: volumeSizeGB,
		FileFormat:   fileFormat,
	}

	return manifests.New(imageProps), nil
}

func (d *SDKCreateMachineImageManifestDriver) uploadManifest(ctx context.Context, bucketName, serverSideEncryption string, m *manifests.ImportVolumeManifest) (string, error) {
	manifestKey := fmt.Sprintf("bosh-machine-image-manifest-%d", time.Now().UnixNano())

	getReq, err := d.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(manifestKey),
	}, s3.WithPresignExpires(1*time.Hour))
	if err != nil {
		return "", fmt.Errorf("failed to sign manifest GET request: %s", err)
	}
	manifestGetURL := getReq.URL

	d.logger.Printf("generated presigned manifest GET URL %s\n", manifestGetURL)

	deleteReq, err := d.presignClient.PresignDeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(manifestKey),
	}, s3.WithPresignExpires(2*time.Hour))
	if err != nil {
		return "", fmt.Errorf("failed to sign manifest delete request: %s", err)
	}
	manifestDeleteURL := deleteReq.URL

	d.logger.Printf("generated presigned manifest DELETE URL %s\n", manifestDeleteURL)

	m.SelfDestructURL = manifestDeleteURL

	manifestBytes, err := xml.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("serializing machine image manifest: %s", err)
	}

	manifestReader := bytes.NewReader(manifestBytes)

	uploadStartTime := time.Now()
	uploader := manager.NewUploader(d.s3Client) //nolint:staticcheck
	uploadInput := &s3.PutObjectInput{
		Body:   manifestReader,
		Bucket: aws.String(bucketName),
		Key:    aws.String(manifestKey),
	}
	if serverSideEncryption != "" {
		uploadInput.ServerSideEncryption = s3types.ServerSideEncryption(serverSideEncryption)
	}
	_, err = uploader.Upload(ctx, uploadInput) //nolint:staticcheck
	if err != nil {
		return "", fmt.Errorf("uploading machine image manifest to S3: %s", err)
	}

	d.logger.Printf("finished uploaded machine image manifest to s3 after %f seconds\n", time.Since(uploadStartTime).Seconds())

	return manifestGetURL, nil
}
