package driver_test

import (
	"fmt"
	"os"
	"strings"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("CopyAmiDriver", func() {
	cpiAmi := func(encrypted bool, kmsKey string, cb ...func(*ec2.EC2, *ec2.DescribeImagesOutput)) {
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

		dstRegion := os.Getenv("AWS_DESTINATION_REGION")
		Expect(dstRegion).ToNot(BeEmpty(), "AWS_DESTINATION_REGION must be set")
		Expect(dstRegion).ToNot(Equal(region), "AWS_REGION and AWS_DESTINATION_REGION should be different")

		existingAmiID := os.Getenv("AMI_FIXTURE_ID")
		Expect(existingAmiID).ToNot(BeEmpty(), "AMI_FIXTURE_ID must be set")

		amiDriverConfig := resources.AmiDriverConfig{}
		amiUniqueID := strings.ToUpper(uuid.NewV4().String())
		amiName := fmt.Sprintf("BOSH-%s", amiUniqueID)

		accessibility := resources.PublicAmiAccessibility
		if encrypted {
			accessibility = resources.PrivateAmiAccessibility
		}

		amiDriverConfig.Name = amiName
		amiDriverConfig.VirtualizationType = resources.HvmAmiVirtualization
		amiDriverConfig.Accessibility = accessibility
		amiDriverConfig.Description = "bosh cpi test ami"
		amiDriverConfig.ExistingAmiID = existingAmiID
		amiDriverConfig.DestinationRegion = dstRegion
		amiDriverConfig.Encrypted = encrypted
		amiDriverConfig.KmsKeyId = kmsKey

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiCopyDriver := ds.CopyAmiDriver()
		copiedAmi, err := amiCopyDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		newSession, err := session.NewSession()
		Expect(err).ToNot(HaveOccurred())
		ec2Client := ec2.New(newSession, &aws.Config{Region: aws.String(dstRegion)}) //nolint:staticcheck
		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(copiedAmi.ID)}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))
		Expect(*reqOutput.Images[0].Name).To(Equal(amiDriverConfig.Name))
		Expect(*reqOutput.Images[0].Architecture).To(Equal(resources.AmiArchitecture))
		Expect(*reqOutput.Images[0].VirtualizationType).To(Equal(amiDriverConfig.VirtualizationType))
		if !encrypted {
			Expect(*reqOutput.Images[0].Public).To(BeTrue())
		}

		if len(cb) > 0 {
			cb[0](ec2Client, reqOutput)
		}

		_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: aws.String(copiedAmi.ID)}) // Ignore DeregisterImageOutput
		Expect(err).ToNot(HaveOccurred())
	}

	getSnapshotID := func(describeImagesOutput *ec2.DescribeImagesOutput) *string {
		var snapshotIDptr *string
		image := describeImagesOutput.Images[0]
		for _, deviceMapping := range image.BlockDeviceMappings {
			if *deviceMapping.DeviceName == *image.RootDeviceName {
				snapshotIDptr = deviceMapping.Ebs.SnapshotId
			}
		}
		return snapshotIDptr
	}

	It("copies an existing AMI to a new region while preserving its properties", func() {
		cpiAmi(false, "", func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
			snapshotIDptr := getSnapshotID(reqOutput)

			snapshotAttributes, err := ec2Client.DescribeSnapshotAttribute(&ec2.DescribeSnapshotAttributeInput{
				SnapshotId: snapshotIDptr,
				Attribute:  aws.String("createVolumePermission"),
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(len(snapshotAttributes.CreateVolumePermissions)).To(Equal(1))
			Expect(*snapshotAttributes.CreateVolumePermissions[0].Group).To(Equal("all"))
		})
	})

	Context("when encrypted flag is set to true", func() {
		It("does NOT make snapshot public", func() {
			cpiAmi(true, "", func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
				snapshotIDptr := getSnapshotID(reqOutput)

				snapshotAttributes, err := ec2Client.DescribeSnapshotAttribute(&ec2.DescribeSnapshotAttributeInput{
					SnapshotId: snapshotIDptr,
					Attribute:  aws.String("createVolumePermission"),
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(snapshotAttributes.CreateVolumePermissions)).To(Equal(0))
			})
		})

		It("encrypts destination AMI using default AWS KMS key", func() {
			cpiAmi(true, "", func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
				respSnapshots, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId}})
				Expect(err).ToNot(HaveOccurred())

				Expect(*respSnapshots.Snapshots[0].Encrypted).To(BeTrue())
			})
		})

		Context("when kms_key_id is provided", func() {
			It("encrypts destination AMI using provided kms key", func() {
				kmsKeyId := os.Getenv("AWS_KMS_KEY_ID")
				Expect(kmsKeyId).ToNot(BeEmpty(), "AWS_KMS_KEY_ID must be set")

				cpiAmi(true, kmsKeyId, func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
					respSnapshots, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId}})
					Expect(err).ToNot(HaveOccurred())

					Expect(*respSnapshots.Snapshots[0].Encrypted).To(BeTrue())
					Expect(*respSnapshots.Snapshots[0].KmsKeyId).To(Equal(kmsKeyId))
				})
			})
		})
	})
})
