package driver_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

		ec2Client := ec2.NewFromConfig(creds.GetAwsConfig())

		reqOutput, err := ec2Client.DescribeImages(context.Background(), &ec2.DescribeImagesInput{ImageIds: []string{ami.ID}})
		Expect(err).ToNot(HaveOccurred())

		Expect(len(reqOutput.Images)).To(Equal(1))

		firstImage := reqOutput.Images[0]
		Expect(*firstImage.Name).To(Equal(amiDriverConfig.Name))
		Expect(string(firstImage.Architecture)).To(Equal(resources.AmiArchitecture))
		Expect(string(firstImage.VirtualizationType)).To(Equal(ami.VirtualizationType))
		Expect(*firstImage.EnaSupport).To(BeTrue())
		Expect(*firstImage.SriovNetSupport).To(Equal("simple"))
		Expect(*firstImage.Public).To(BeTrue())

		instanceReservation, err := ec2Client.RunInstances(context.Background(), &ec2.RunInstancesInput{
			ImageId:      aws.String(ami.ID),
			InstanceType: ec2types.InstanceTypeM3Medium,
			MinCount:     aws.Int32(1),
			MaxCount:     aws.Int32(1),
			NetworkInterfaces: []ec2types.InstanceNetworkInterfaceSpecification{
				{
					DeviceIndex:              aws.Int32(0),
					AssociatePublicIpAddress: aws.Bool(true),
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		instanceID := instanceReservation.Instances[0].InstanceId
		logger.Printf("Created VM with instance ID: %s", *instanceID)

		instanceRunningWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
		Eventually(func() error {
			return instanceRunningWaiter.Wait(context.Background(), &ec2.DescribeInstancesInput{InstanceIds: []string{*instanceID}}, 15*time.Minute)
		}, 15*time.Minute, 10*time.Second).Should(BeNil())

		instanceStatusOkWaiter := ec2.NewInstanceStatusOkWaiter(ec2Client)
		err = instanceStatusOkWaiter.Wait(context.Background(), &ec2.DescribeInstanceStatusInput{InstanceIds: []string{*instanceID}}, 30*time.Minute)
		if err != nil {
			logger.Printf("Encountered error waiting for VM to boot, retrying once: %s", err)
			err = instanceStatusOkWaiter.Wait(context.Background(), &ec2.DescribeInstanceStatusInput{InstanceIds: []string{*instanceID}}, 30*time.Minute)
			Expect(err).ToNot(HaveOccurred())
		}

		_, err = ec2Client.TerminateInstances(context.Background(), &ec2.TerminateInstancesInput{InstanceIds: []string{*instanceID}})
		Expect(err).ToNot(HaveOccurred())

		instanceTerminatedWaiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
		err = instanceTerminatedWaiter.Wait(context.Background(), &ec2.DescribeInstancesInput{InstanceIds: []string{*instanceID}}, 15*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		_, err = ec2Client.DeregisterImage(context.Background(), &ec2.DeregisterImageInput{ImageId: &ami.ID})
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

			ec2Client := ec2.NewFromConfig(creds.GetAwsConfig())

			output, err := ec2Client.DescribeImageAttribute(context.Background(), &ec2.DescribeImageAttributeInput{
				ImageId:   &ami.ID,
				Attribute: ec2types.ImageAttributeNameLaunchPermission,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(*output.LaunchPermissions[0].UserId).To(Equal(awsAccount))

			_, err = ec2Client.DeregisterImage(context.Background(), &ec2.DeregisterImageInput{ImageId: &ami.ID})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
