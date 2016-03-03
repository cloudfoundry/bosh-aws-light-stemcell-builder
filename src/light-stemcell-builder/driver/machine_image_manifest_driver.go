package driver

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver/manifests"
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

// The SDKMachineImageManifestDriver uploads a machine image to S3 and creates an import volume manifest
type SDKMachineImageManifestDriver struct {
	s3Client    *s3.S3
	logger      *log.Logger
	genManifest bool
}

// NewMachineImageManifestDriver creates a MachineImageDriver machine image manifest generation
func NewMachineImageManifestDriver(logDest io.Writer, creds config.Credentials) *SDKMachineImageManifestDriver {
	logger := log.New(logDest, "SDKMachineImageManifestDriver ", log.LstdFlags)

	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(creds.AccessKey, creds.SecretKey, "")).
		WithRegion(creds.Region).
		WithLogger(newDriverLogger(logger))

	s3Session := session.New(awsConfig)
	s3Client := s3.New(s3Session)

	return &SDKMachineImageManifestDriver{
		s3Client: s3Client,
		logger:   logger,
	}
}

// Create uploads a machine image to S3 and returns a presigned URL to an import volume manifest
func (d *SDKMachineImageManifestDriver) Create(driverConfig resources.MachineImageDriverConfig) (resources.MachineImage, error) {
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
	_, err = uploader.Upload(&s3manager.UploadInput{
		Body:   f,
		Bucket: aws.String(driverConfig.BucketName),
		Key:    aws.String(keyName),
	})

	if err != nil {
		return resources.MachineImage{}, fmt.Errorf("uploading machine image to S3: %s", err)
	}

	d.logger.Printf("finished uploaded image to s3 after %f minutes\n", time.Since(uploadStartTime).Minutes())

	headReqOutput, err := d.s3Client.HeadObject(&s3.HeadObjectInput{
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

	manifestURL, err := d.generateManifest(driverConfig.BucketName, keyName, *sizeInBytesPtr)

	return resources.MachineImage{GetURL: manifestURL}, nil
}

func (d *SDKMachineImageManifestDriver) generateManifest(bucketName string, keyName string, sizeInBytes int64) (string, error) {
	// Generate presigned GET request
	req, _ := d.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	})

	presignedGetURL, err := req.Presign(2 * time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to sign request: %s", err)
	}

	d.logger.Printf("generated presigned GET URL %s\n", presignedGetURL)

	// Generate presigned HEAD request for the machine image
	req, _ = d.s3Client.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	})

	presignedHeadURL, err := req.Presign(1 * time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to sign request: %s", err)
	}

	d.logger.Printf("generated presigned HEAD URL %s\n", presignedHeadURL)

	// Generate presigned DELETE request for the machine image
	req, _ = d.s3Client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	})

	presignedDeleteURL, err := req.Presign(1 * time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to sign request: %s", err)
	}

	d.logger.Printf("generated presigned DELETE URL %s\n", presignedDeleteURL)

	manifestKey := fmt.Sprintf("bosh-machine-image-manifest-%d", time.Now().UnixNano())

	imageProps := manifests.MachineImageProperties{
		KeyName:   keyName,
		HeadURL:   presignedHeadURL,
		GetURL:    presignedGetURL,
		DeleteURL: presignedDeleteURL,
		SizeBytes: sizeInBytes,
	}

	// create presigned DELETE request for the manifest
	deleteReq, _ := d.s3Client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(manifestKey),
	})

	manifestDeleteURL, err := deleteReq.Presign(2 * time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to sign manifest delete request: %s", err)
	}

	d.logger.Printf("generated presigned manifest DELETE URL %s\n", manifestDeleteURL)

	m := manifests.New(imageProps, manifestDeleteURL)

	manifestBytes, err := xml.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("serializing machine image manifest: %s", err)
	}

	manifestReader := bytes.NewReader(manifestBytes)

	uploadStartTime := time.Now()
	uploader := s3manager.NewUploaderWithClient(d.s3Client)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Body:   manifestReader,
		Bucket: aws.String(bucketName),
		Key:    aws.String(manifestKey),
	})

	if err != nil {
		return "", fmt.Errorf("uploading machine image manifest to S3: %s", err)
	}

	d.logger.Printf("finished uploaded machine image manifest to s3 after %f seconds\n", time.Since(uploadStartTime).Seconds())

	// create presigned GET request for the manifest
	getReq, _ := d.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(manifestKey),
	})

	manifestGetURL, err := getReq.Presign(1 * time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to sign manifest GET request: %s", err)
	}

	d.logger.Printf("generated presigned manifest GET URL %s\n", manifestGetURL)

	return manifestGetURL, nil
}
