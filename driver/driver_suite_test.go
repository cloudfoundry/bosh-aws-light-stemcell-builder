package driver_test

import (
	"os"
	"testing"

	"light-stemcell-builder/config"
	"light-stemcell-builder/test_helpers"

	"github.com/aws/aws-sdk-go/aws/session"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var creds config.Credentials
var awsSession *session.Session

var destinationRegion string

var bucketName string

var ebsVolumeID, ebsSnapshotID string
var machineImagePath, machineImageFormat string
var s3MachineImageUrl, s3MachineImageFormat string

var kmsKeyId string

var amiFixtureID string

func TestDrivers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Drivers Suite")
}

var _ = SynchronizedBeforeSuite(
	func() []byte { return []byte{} },
	func([]byte) {
		creds = constructCredentials()

		// Destination Region
		destinationRegion = os.Getenv("AWS_DESTINATION_REGION")
		Expect(destinationRegion).ToNot(BeEmpty(), "AWS_DESTINATION_REGION must be set")
		Expect(destinationRegion).ToNot(Equal(creds.Region), "AWS_REGION and AWS_DESTINATION_REGION should be different")

		// AWS Bucket
		bucketName = os.Getenv("AWS_BUCKET_NAME")
		Expect(bucketName).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")

		// EBS info
		ebsVolumeID = os.Getenv("EBS_VOLUME_ID")
		Expect(ebsVolumeID).ToNot(BeEmpty(), "EBS_VOLUME_ID must be set")

		ebsSnapshotID = os.Getenv("EBS_SNAPSHOT_ID")
		Expect(ebsSnapshotID).ToNot(BeEmpty(), "EBS_SNAPSHOT_ID must be set")

		// Machine Image info
		machineImagePath = os.Getenv("MACHINE_IMAGE_PATH")
		Expect(machineImagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		machineImageFormat = os.Getenv("MACHINE_IMAGE_FORMAT")
		Expect(machineImagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_FORMAT must be set")

		// S3 Machine Image info
		s3MachineImageUrl = os.Getenv("S3_MACHINE_IMAGE_URL")
		Expect(s3MachineImageUrl).ToNot(BeEmpty(), "S3_MACHINE_IMAGE_URL must be set")

		s3MachineImageFormat = os.Getenv("S3_MACHINE_IMAGE_FORMAT")
		Expect(s3MachineImageFormat).ToNot(BeEmpty(), "S3_MACHINE_IMAGE_FORMAT must be set")

		// AMI fixture
		amiFixtureID = os.Getenv("AMI_FIXTURE_ID")
		Expect(amiFixtureID).ToNot(BeEmpty(), "AMI_FIXTURE_ID must be set")

		// KMS Key info
		kmsKeyId = os.Getenv("AWS_KMS_KEY_ID")
		Expect(kmsKeyId).ToNot(BeEmpty(), "AWS_KMS_KEY_ID must be set")

		var err error
		awsSession, err = session.NewSession(test_helpers.AwsConfigFrom(creds))
		Expect(err).ToNot(HaveOccurred())
	},
)

func constructCredentials() config.Credentials {
	// Credentials
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	Expect(accessKey).ToNot(BeEmpty(), "AWS_ACCESS_KEY_ID must be set")

	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	Expect(secretKey).ToNot(BeEmpty(), "AWS_SECRET_ACCESS_KEY must be set")

	region := os.Getenv("AWS_REGION")
	Expect(region).ToNot(BeEmpty(), "AWS_REGION must be set")

	roleArn := os.Getenv("AWS_ROLE_ARN")

	return config.Credentials{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
		RoleArn:   roleArn,
	}
}
