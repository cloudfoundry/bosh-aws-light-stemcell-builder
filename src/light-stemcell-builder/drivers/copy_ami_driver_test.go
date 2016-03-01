package drivers_test

import (
	"fmt"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driversets"
	"light-stemcell-builder/resources"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
)

var _ = Describe("CopyAmiDriver", func() {
	It("copies an existing AMI to a new region while preserving its properties", func() {
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

		existingAmiID := os.Getenv("AWS_AMI_FIXTURE_ID")
		Expect(existingAmiID).ToNot(BeEmpty(), "AWS_AMI_FIXTURE_ID must be set")

		amiDriverConfig := resources.AmiDriverConfig{}
		amiUniqueID := strings.ToUpper(uuid.NewV4().String())
		amiName := fmt.Sprintf("BOSH-%s", amiUniqueID)

		amiDriverConfig.Name = amiName
		amiDriverConfig.VirtualizationType = resources.HvmAmiVirtualization
		amiDriverConfig.Accessibility = resources.PublicAmiAccessibility
		amiDriverConfig.Description = "bosh cpi test ami"
		amiDriverConfig.ExistingAmiID = existingAmiID
		amiDriverConfig.DestinationRegion = dstRegion

		ds := driversets.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiCopyDriver := ds.CopyAmiDriver()
		copiedAmiID, err := amiCopyDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(dstRegion)})
		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{&copiedAmiID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))
		Expect(*reqOutput.Images[0].Name).To(Equal(amiDriverConfig.Name))
		Expect(*reqOutput.Images[0].Architecture).To(Equal(resources.AmiArchitecture))
		Expect(*reqOutput.Images[0].VirtualizationType).To(Equal(amiDriverConfig.VirtualizationType))
		Expect(*reqOutput.Images[0].Public).To(BeTrue())

		_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: &copiedAmiID}) // Ignore DeregisterImageOutput
		Expect(err).ToNot(HaveOccurred())
	})
})
