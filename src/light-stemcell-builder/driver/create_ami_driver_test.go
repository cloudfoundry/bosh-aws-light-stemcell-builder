package driver_test

import (
	"fmt"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
)

var _ = Describe("CreateAmiDriver", func() {
	It("creates a bootable HVM AMI from an existing snapshot", func() {

		logger := log.New(GinkgoWriter, "CreateAmiDriver - Bootable HVM Test: ", log.LstdFlags)

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

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiDriver := ds.CreateAmiDriver()
		ami, err := amiDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(ami.VirtualizationType).To(Equal(resources.HvmAmiVirtualization))

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(ami.Region)})
		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(ami.ID)}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))
		Expect(*reqOutput.Images[0].Name).To(Equal(amiName))
		Expect(*reqOutput.Images[0].Architecture).To(Equal(resources.AmiArchitecture))
		Expect(*reqOutput.Images[0].VirtualizationType).To(Equal(ami.VirtualizationType))
		Expect(*reqOutput.Images[0].SriovNetSupport).To(Equal("simple"))
		Expect(*reqOutput.Images[0].Public).To(BeTrue())

		instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      aws.String(ami.ID),
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

		instanceID := instanceReservation.Instances[0].InstanceId
		logger.Printf("Created VM with instance ID: %s", *instanceID)

		Eventually(func() error {
			// there is a bug in the Instance Waiters where the status InvalidInstanceID.NotFound is not properly handled
			// retry waiting in an Eventually block to work around this problem
			return ec2Client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceID}})
		}, 15*time.Minute, 10*time.Second).Should(BeNil())

		err = ec2Client.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{InstanceIds: []*string{instanceID}})
		if err != nil {
			logger.Printf("Encountered error waiting for VM to boot, retrying once: %s", err)
			err = ec2Client.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{InstanceIds: []*string{instanceID}})
			Expect(err).ToNot(HaveOccurred())
		}

		_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{instanceID}}) // Ignore TerminateInstancesOutput
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceID}})
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: &ami.ID}) // Ignore DeregisterImageOutput
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates a bootable PV AMI from an existing snapshot", func() {

		logger := log.New(GinkgoWriter, "CreateAmiDriver - Bootable PV Test: ", log.LstdFlags)

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

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiDriver := ds.CreateAmiDriver()
		ami, err := amiDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(ami.ID)}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))
		Expect(*reqOutput.Images[0].Architecture).To(Equal(resources.AmiArchitecture))
		Expect(reqOutput.Images[0].SriovNetSupport).To(BeNil())
		Expect(*reqOutput.Images[0].VirtualizationType).To(Equal(resources.PvAmiVirtualization))
		Expect(*reqOutput.Images[0].Public).To(BeTrue())

		instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      aws.String(ami.ID),
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

		instanceID := instanceReservation.Instances[0].InstanceId
		logger.Printf("Created VM with instance ID: %s", *instanceID)

		Eventually(func() error {
			// there is a bug in the Instance Waiters where the status InvalidInstanceID.NotFound is not properly handled
			// retry waiting in an Eventually block to work around this problem
			return ec2Client.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceID}})
		}, 15*time.Minute, 10*time.Second).Should(BeNil())

		err = ec2Client.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{InstanceIds: []*string{instanceID}})
		if err != nil {
			logger.Printf("Encountered error waiting for VM to boot, retrying once: %s", err)
			err = ec2Client.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{InstanceIds: []*string{instanceID}})
			Expect(err).ToNot(HaveOccurred())
		}

		_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{instanceID}}) // Ignore TerminateInstancesOutput
		Expect(err).ToNot(HaveOccurred())

		err = ec2Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{InstanceIds: []*string{instanceID}})
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: &ami.ID}) // Ignore DeregisterImageOutput
		Expect(err).ToNot(HaveOccurred())
	})
})
