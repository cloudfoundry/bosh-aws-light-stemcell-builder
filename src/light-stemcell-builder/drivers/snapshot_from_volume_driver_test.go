package drivers_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/driversets"
	"light-stemcell-builder/resources"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SnapshotFromVolumeDriver", func() {
	It("creates an snapshot from an existing EBS volume", func() {
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

		volumeID := os.Getenv("EBS_VOLUME_ID")
		Expect(volumeID).ToNot(BeEmpty(), "EBS_VOLUME_ID must be set")

		driverConfig := resources.SnapshotDriverConfig{
			VolumeID: volumeID,
		}

		ds := driversets.NewIsolatedRegionDriverSet(GinkgoWriter, creds)
		driver := ds.CreateSnapshotDriver()

		snapshotID, err := driver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
		reqOutput, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{&snapshotID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Snapshots)).To(Equal(1))
	})
})
