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

var _ = Describe("CreateAmiDriver", func() {
	It("creates a bootable HVM AMI from an existing snapshot", func() {
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

		snapshotID := os.Getenv("EBS_SNAPSHOT_ID")
		Expect(snapshotID).ToNot(BeEmpty(), "EBS_SNAPSHOT_ID must be set")

		amiDriverConfig := resources.AmiDriverConfig{SnapshotID: snapshotID}
		amiUniqueID := strings.ToUpper(uuid.NewV4().String())
		amiName := fmt.Sprintf("BOSH-%s", amiUniqueID)

		amiDriverConfig.Name = amiName
		amiDriverConfig.VirtualizationType = resources.HvmAmiVirtualization
		amiDriverConfig.Accessibility = resources.PublicAmiAccessibility
		amiDriverConfig.Description = "bosh cpi test ami"

		ds := driversets.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiDriver := ds.CreateAmiDriver()
		amiID, err := amiDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{&amiID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))
		Expect(*reqOutput.Images[0].Name).To(Equal(amiName))
		Expect(*reqOutput.Images[0].Architecture).To(Equal(resources.AmiArchitecture))
		Expect(*reqOutput.Images[0].VirtualizationType).To(Equal(resources.HvmAmiVirtualization))
		Expect(*reqOutput.Images[0].Public).To(BeTrue())

		instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      &amiID,
			InstanceType: aws.String(ec2.InstanceTypeM3Medium),
			MinCount:     aws.Int64(1),
			MaxCount:     aws.Int64(1),
			NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
				&ec2.InstanceNetworkInterfaceSpecification{
					DeviceIndex:              aws.Int64(0),
					AssociatePublicIpAddress: aws.Bool(true), // Associate a public address to avoid explicitly defining subnet information
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}})
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}})
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}}) // Ignore TerminateInstancesOutput
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}})
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: &amiID}) // Ignore DeregisterImageOutput
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates a bootable PV AMI from an existing snapshot", func() {
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

		snapshotID := os.Getenv("EBS_SNAPSHOT_ID")
		Expect(snapshotID).ToNot(BeEmpty(), "EBS_SNAPSHOT_ID must be set")

		amiUniqueID := strings.ToUpper(uuid.NewV4().String())
		amiName := fmt.Sprintf("BOSH-%s", amiUniqueID)

		amiDriverConfig := resources.AmiDriverConfig{SnapshotID: snapshotID}
		amiDriverConfig.VirtualizationType = resources.PvAmiVirtualization
		amiDriverConfig.Accessibility = resources.PublicAmiAccessibility
		amiDriverConfig.Name = amiName
		amiDriverConfig.Description = "bosh cpi test ami"

		ds := driversets.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiDriver := ds.CreateAmiDriver()
		amiID, err := amiDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{&amiID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))
		Expect(*reqOutput.Images[0].Architecture).To(Equal(resources.AmiArchitecture))
		Expect(*reqOutput.Images[0].VirtualizationType).To(Equal(resources.PvAmiVirtualization))
		Expect(*reqOutput.Images[0].Public).To(BeTrue())

		instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      &amiID,
			InstanceType: aws.String(ec2.InstanceTypeM3Medium),
			MinCount:     aws.Int64(1),
			MaxCount:     aws.Int64(1),
			NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
				&ec2.InstanceNetworkInterfaceSpecification{
					DeviceIndex:              aws.Int64(0),
					AssociatePublicIpAddress: aws.Bool(true), // Associate a public address to avoid explicitly defining subnet information
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}})
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}})
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}}) // Ignore TerminateInstancesOutput
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceReservation.Instances[0].InstanceId}})
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: &amiID}) // Ignore DeregisterImageOutput
		Expect(err).ToNot(HaveOccurred())
	})
})
