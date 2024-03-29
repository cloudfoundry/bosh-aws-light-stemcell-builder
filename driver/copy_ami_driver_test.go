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

type AmiCopyConfig struct {
	amiId              string
	encrypted          bool
	kmsKeyId           string
	sharedWithAccounts []string
}

var _ = Describe("CopyAmiDriver", func() {
	BeforeEach(func() {
		if os.Getenv("SKIP_REPLICATION_TESTS") != "" {
			Skip("Skipping test, found 'SKIP_REPLICATION_TESTS'")
		}
	})

	It("copies an existing AMI to a new region while preserving its properties", func() {
		copyAmi(
			AmiCopyConfig{
				amiId:     amiFixtureID,
				encrypted: false,
				kmsKeyId:  ""},
			func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
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
			copyAmi(
				AmiCopyConfig{
					amiId:     amiFixtureID,
					encrypted: true,
					kmsKeyId:  "",
				},
				func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
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
			copyAmi(
				AmiCopyConfig{
					amiId:     amiFixtureID,
					encrypted: true,
					kmsKeyId:  "",
				},
				func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
					respSnapshots, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId}})
					Expect(err).ToNot(HaveOccurred())

					Expect(*respSnapshots.Snapshots[0].Encrypted).To(BeTrue())
				})
		})

		Context("when kms_key_id is provided", func() {
			It("encrypts destination AMI using the kms key in the destination region", func() {
				destinationRegionKmsKeyId := strings.ReplaceAll(multiRegionKey, creds.Region, destinationRegion)
				copyAmi(
					AmiCopyConfig{
						amiId:     privateAmiFixtureID,
						encrypted: true,
						kmsKeyId:  destinationRegionKmsKeyId,
					},
					func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
						respSnapshots, err := ec2Client.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId}})
						Expect(err).ToNot(HaveOccurred())

						Expect(*respSnapshots.Snapshots[0].Encrypted).To(BeTrue())
						Expect(*respSnapshots.Snapshots[0].KmsKeyId).To(Equal(destinationRegionKmsKeyId))
					})
			})
		})
	})

	Context("when shared_with_accounts is provided", func() {
		It("shares the AMI with other accounts", func() {
			destinationRegionKmsKeyId := strings.ReplaceAll(multiRegionKey, creds.Region, destinationRegion)
			copyAmi(AmiCopyConfig{
				amiId:              privateAmiFixtureID,
				encrypted:          true,
				kmsKeyId:           destinationRegionKmsKeyId,
				sharedWithAccounts: []string{awsAccount},
			},
				func(ec2Client *ec2.EC2, reqOutput *ec2.DescribeImagesOutput) {
					attribute := "launchPermission"
					output, err := ec2Client.DescribeImageAttribute(&ec2.DescribeImageAttributeInput{
						ImageId:   reqOutput.Images[0].ImageId,
						Attribute: &attribute,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(*output.LaunchPermissions[0].UserId).To(Equal(awsAccount))
				})
		})
	})
	Context("when making an AMI public", func() {
		It("will return an error if it can't make the AMI public", func() {
			amiProperties := resources.AmiProperties{
				Name:               fmt.Sprintf("BOSH-%s", strings.ToUpper(uuid.NewV4().String())),
				VirtualizationType: resources.HvmAmiVirtualization,
				Description:        "bosh cpi test ami",
				Accessibility:      resources.PublicAmiAccessibility,
				Encrypted:          true,
				KmsKeyId:           multiRegionKey,
			}
			destinationRegionKmsKeyId := strings.ReplaceAll(multiRegionKey, creds.Region, destinationRegion)
			amiCopyConfig := AmiCopyConfig{
				amiId:     privateAmiFixtureID,
				encrypted: true,
				kmsKeyId:  destinationRegionKmsKeyId,
			}
			amiDriverConfig := resources.AmiDriverConfig{
				ExistingAmiID:     amiCopyConfig.amiId,
				DestinationRegion: destinationRegion,
				AmiProperties:     amiProperties,
				KmsKey:            resources.KmsKey{ARN: amiCopyConfig.kmsKeyId},
			}
			amiCopyDriver := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds).CopyAmiDriver()
			_, err := amiCopyDriver.Create(amiDriverConfig)
			Expect(err).To(HaveOccurred())
		})
	})
})

func copyAmi(amiCopyConfig AmiCopyConfig, cb ...func(*ec2.EC2, *ec2.DescribeImagesOutput)) {
	accessibility := resources.PublicAmiAccessibility
	if amiCopyConfig.encrypted {
		accessibility = resources.PrivateAmiAccessibility
	}

	amiProperties := resources.AmiProperties{
		Name:               fmt.Sprintf("BOSH-%s", strings.ToUpper(uuid.NewV4().String())),
		VirtualizationType: resources.HvmAmiVirtualization,
		Description:        "bosh cpi test ami",
		Accessibility:      accessibility,
		Encrypted:          amiCopyConfig.encrypted,
		KmsKeyId:           amiCopyConfig.kmsKeyId,
	}
	if len(amiCopyConfig.sharedWithAccounts) > 0 {
		amiProperties.SharedWithAccounts = amiCopyConfig.sharedWithAccounts
	}

	amiDriverConfig := resources.AmiDriverConfig{
		ExistingAmiID:     amiCopyConfig.amiId,
		DestinationRegion: destinationRegion,
		AmiProperties:     amiProperties,
		KmsKey:            resources.KmsKey{ARN: amiCopyConfig.kmsKeyId},
	}

	amiCopyDriver := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds).CopyAmiDriver()
	copiedAmi, err := amiCopyDriver.Create(amiDriverConfig)
	Expect(err).ToNot(HaveOccurred())

	destinationCreds := config.Credentials{
		AccessKey: creds.AccessKey,
		SecretKey: creds.SecretKey,
		RoleArn:   creds.RoleArn,
		Region:    destinationRegion,
	}
	awsSession, err := session.NewSession(destinationCreds.GetAwsConfig())
	Expect(err).ToNot(HaveOccurred())
	ec2Client := ec2.New(awsSession)
	reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(copiedAmi.ID)}})
	Expect(err).ToNot(HaveOccurred())

	Expect(len(reqOutput.Images)).To(Equal(1))

	firstImage := reqOutput.Images[0]
	Expect(*firstImage.Name).To(Equal(amiDriverConfig.Name))
	Expect(*firstImage.Architecture).To(Equal(resources.AmiArchitecture))
	Expect(*firstImage.VirtualizationType).To(Equal(amiDriverConfig.VirtualizationType))
	if !amiCopyConfig.encrypted {
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
