package driver_test

import (
	"fmt"
	"log"
	"strings"
	"time"

	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("CreateAmiDriver", func() {
	It("creates a bootable HVM AMI from an existing snapshot", func() {
		logger := log.New(GinkgoWriter, "CreateAmiDriver - Bootable HVM Test: ", log.LstdFlags)

		amiDriverConfig := resources.AmiDriverConfig{
			SnapshotID: ebsSnapshotID,
			AmiProperties: resources.AmiProperties{
				Name:               fmt.Sprintf("BOSH-%s", strings.ToUpper(uuid.NewV4().String())),
				VirtualizationType: resources.HvmAmiVirtualization,
				Accessibility:      resources.PublicAmiAccessibility,
				Description:        "bosh cpi test ami",
			},
		}

		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)

		amiDriver := ds.CreateAmiDriver()
		ami, err := amiDriver.Create(amiDriverConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(ami.VirtualizationType).To(Equal(resources.HvmAmiVirtualization))

		awsSession, err := session.NewSession(creds.GetAwsConfig())
		Expect(err).ToNot(HaveOccurred())
		ec2Client := ec2.New(awsSession)

		reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(ami.ID)}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))

		firstImage := reqOutput.Images[0]
		Expect(*firstImage.Name).To(Equal(amiDriverConfig.Name))
		Expect(*firstImage.Architecture).To(Equal(resources.AmiArchitecture))
		Expect(*firstImage.VirtualizationType).To(Equal(ami.VirtualizationType))
		Expect(*firstImage.EnaSupport).To(BeTrue())
		Expect(*firstImage.SriovNetSupport).To(Equal("simple"))
		Expect(*firstImage.Public).To(BeTrue())

		instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      aws.String(ami.ID),
			InstanceType: aws.String(ec2.InstanceTypeM3Medium),
			MinCount:     aws.Int64(1),
			MaxCount:     aws.Int64(1),
			NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
				{
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

	Context("when shared_with_accounts is provided", func() {
		It("shares the AMI with other accounts", func() {
			amiDriverConfig := resources.AmiDriverConfig{
				SnapshotID: ebsSnapshotID,
				AmiProperties: resources.AmiProperties{
					Name:               fmt.Sprintf("BOSH-%s", strings.ToUpper(uuid.NewV4().String())),
					VirtualizationType: resources.HvmAmiVirtualization,
					Accessibility:      resources.PublicAmiAccessibility,
					SharedWithAccounts: []string{awsAccount},
				},
			}

			ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)

			amiDriver := ds.CreateAmiDriver()
			ami, err := amiDriver.Create(amiDriverConfig)
			Expect(err).ToNot(HaveOccurred())

			awsSession, err := session.NewSession(creds.GetAwsConfig())
			Expect(err).ToNot(HaveOccurred())
			ec2Client := ec2.New(awsSession)

			attribute := "launchPermission"
			output, err := ec2Client.DescribeImageAttribute(&ec2.DescribeImageAttributeInput{
				ImageId:   &ami.ID,
				Attribute: &attribute,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(*output.LaunchPermissions[0].UserId).To(Equal(awsAccount))

			_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: &ami.ID})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
