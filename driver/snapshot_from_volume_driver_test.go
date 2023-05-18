package driver_test

import (
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SnapshotFromVolumeDriver", func() {
	It("creates a public snapshot from an existing EBS volume", func() {
		driverConfig := resources.SnapshotDriverConfig{VolumeID: ebsVolumeID}

		ds := driverset.NewIsolatedRegionDriverSet(GinkgoWriter, awsSession, creds)
		driver := ds.CreateSnapshotDriver()

		snapshot, err := driver.Create(driverConfig)
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
})
