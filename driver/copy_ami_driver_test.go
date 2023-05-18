package driver_test

import (
	"fmt"
	"strings"

	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"light-stemcell-builder/test_helpers"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("CopyAmiDriver", func() {
	It("copies an existing AMI to a new region while preserving its properties", func() {
		cpiAmi(false, "", awsSession, func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
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
			cpiAmi(true, "", awsSession, func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
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
			cpiAmi(true, "", awsSession, func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
				respSnapshots, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId}})
				Expect(err).ToNot(HaveOccurred())

				Expect(*respSnapshots.Snapshots[0].Encrypted).To(BeTrue())
			})
		})

		Context("when kms_key_id is provided", func() {
			It("encrypts destination AMI using provided kms key", func() {
				cpiAmi(true, kmsKeyId, awsSession, func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
					respSnapshots, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId}})
					Expect(err).ToNot(HaveOccurred())

					Expect(*respSnapshots.Snapshots[0].Encrypted).To(BeTrue())
					Expect(*respSnapshots.Snapshots[0].KmsKeyId).To(Equal(kmsKeyId))
				})
			})
		})
	})
})

func cpiAmi(encrypted bool, kmsKey string, awsSession *session.Session, cb ...func(*ec2.EC2, *ec2.DescribeImagesOutput)) {
	accessibility := resources.PublicAmiAccessibility
	if encrypted {
		accessibility = resources.PrivateAmiAccessibility
	}

	amiDriverConfig := resources.AmiDriverConfig{
		ExistingAmiID:     amiFixtureID,
		DestinationRegion: destinationRegion,
		AmiProperties: resources.AmiProperties{
			Name:               fmt.Sprintf("BOSH-%s", strings.ToUpper(uuid.NewV4().String())),
			VirtualizationType: resources.HvmAmiVirtualization,
			Description:        "bosh cpi test ami",
			Accessibility:      accessibility,
			Encrypted:          encrypted,
			KmsKeyId:           kmsKey,
		},
	}

	amiCopyDriver := driverset.NewStandardRegionDriverSet(GinkgoWriter, awsSession, creds).CopyAmiDriver()
	copiedAmi, err := amiCopyDriver.Create(amiDriverConfig)
	Expect(err).ToNot(HaveOccurred())

	destinationCreds := config.Credentials{
		AccessKey: creds.AccessKey,
		SecretKey: creds.SecretKey,
		RoleArn:   creds.RoleArn,
		Region:    destinationRegion,
	}
	destinationAwsSession, err := session.NewSession(test_helpers.AwsConfigFrom(destinationCreds))
	Expect(err).ToNot(HaveOccurred())
	ec2Client := ec2.New(destinationAwsSession)
	reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(copiedAmi.ID)}})
	Expect(err).ToNot(HaveOccurred())

	Expect(len(reqOutput.Images)).To(Equal(1))

	firstImage := reqOutput.Images[0]
	Expect(*firstImage.Name).To(Equal(amiDriverConfig.Name))
	Expect(*firstImage.Architecture).To(Equal(resources.AmiArchitecture))
	Expect(*firstImage.VirtualizationType).To(Equal(amiDriverConfig.VirtualizationType))
	if !encrypted {
		Expect(*firstImage.Public).To(BeTrue())
	}

	if len(cb) > 0 {
		cb[0](ec2Client, reqOutput)
	}

	_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: aws.String(copiedAmi.ID)}) // Ignore DeregisterImageOutput
	Expect(err).ToNot(HaveOccurred())
}

func getSnapshotID(describeImagesOutput *ec2.DescribeImagesOutput) *string {
	var snapshotIDptr *string
	image := describeImagesOutput.Images[0]
	for _, deviceMapping := range image.BlockDeviceMappings {
		if *deviceMapping.DeviceName == *image.RootDeviceName {
			snapshotIDptr = deviceMapping.Ebs.SnapshotId
		}
	}
	return snapshotIDptr
}
