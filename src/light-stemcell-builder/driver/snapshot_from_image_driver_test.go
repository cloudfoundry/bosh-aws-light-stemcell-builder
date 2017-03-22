package driver_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SnapshotFromImageDriver", func() {
	It("creates a public snapshot from a machine image located at some S3 URL", func() {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		Expect(accessKey).ToNot(BeEmpty(), "AWS_ACCESS_KEY_ID must be set")

		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		Expect(secretKey).ToNot(BeEmpty(), "AWS_SECRET_ACCESS_KEY must be set")

		region := os.Getenv("AWS_REGION")
		Expect(region).ToNot(BeEmpty(), "AWS_REGION must be set")

		creds := config.Credentials{
			AccessKey: accessKey,
			SecretKey: secretKey,
			Region:    region,
		}

		imagePath := os.Getenv("S3_MACHINE_IMAGE_URL")
		Expect(imagePath).ToNot(BeEmpty(), "S3_MACHINE_IMAGE_URL must be set")

		imageFormat := os.Getenv("S3_MACHINE_IMAGE_FORMAT")
		Expect(imageFormat).ToNot(BeEmpty(), "S3_MACHINE_IMAGE_FORMAT must be set")

		driverConfig := resources.SnapshotDriverConfig{
			MachineImageURL: imagePath,
			FileFormat:      imageFormat,
		}

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)
		driver := ds.CreateSnapshotDriver()

		snapshot, err := driver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
		reqOutput, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{&snapshot.ID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Snapshots)).To(Equal(1))

		snapshotAttributes, err := ec2Client.DescribeSnapshotAttribute(&ec2.DescribeSnapshotAttributeInput{
			SnapshotId: aws.String(snapshot.ID),
			Attribute:  aws.String("createVolumePermission"),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(snapshotAttributes.CreateVolumePermissions)).To(Equal(1))
		Expect(*snapshotAttributes.CreateVolumePermissions[0].Group).To(Equal("all"))
	})
})
