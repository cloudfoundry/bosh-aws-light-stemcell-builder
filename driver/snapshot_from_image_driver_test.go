package driver_test

import (
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SnapshotFromImageDriver", func() {
	It("creates a public snapshot from a machine image located at some S3 URL", func() {
		driverConfig := resources.SnapshotDriverConfig{
			MachineImageURL: s3MachineImageUrl,
			FileFormat:      s3MachineImageFormat,
		}

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)
		driver := ds.CreateSnapshotDriver()

		snapshot, err := driver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		awsSession, err := session.NewSession(creds.GetAwsConfig())
		Expect(err).ToNot(HaveOccurred())
		ec2Client := ec2.New(awsSession)

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

		//cleanup
		_, err = ec2Client.DeleteSnapshot(&ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(snapshot.ID),
		})
		Expect(err).To(BeNil())
	})

	It("skips modifying the snapshot privacy for a private machine image", func() {
		driverConfig := resources.SnapshotDriverConfig{
			MachineImageURL: s3MachineImageUrl,
			FileFormat:      s3MachineImageFormat,
		}

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)
		driver := ds.CreateSnapshotDriver()

		snapshot, err := driver.Create(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		awsSession, err := session.NewSession(creds.GetAwsConfig())
		Expect(err).ToNot(HaveOccurred())
		ec2Client := ec2.New(awsSession)

		reqOutput, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{&snapshot.ID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Snapshots)).To(Equal(1))

		snapshotAttributes, err := ec2Client.DescribeSnapshotAttribute(&ec2.DescribeSnapshotAttributeInput{
			SnapshotId: aws.String(snapshot.ID),
			Attribute:  aws.String("createVolumePermission"),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(snapshotAttributes.CreateVolumePermissions)).To(Equal(0))

		//cleanup
		_, err = ec2Client.DeleteSnapshot(&ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(snapshot.ID),
		})
		Expect(err).To(BeNil())
	})
})
