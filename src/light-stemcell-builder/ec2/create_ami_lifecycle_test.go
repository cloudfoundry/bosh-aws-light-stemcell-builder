package ec2_test

import (
	"os"

	ourEC2 "light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws/aws-sdk-go/aws/session"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateAmi lifecycle", func() {
	Describe("creating and deleting an ami", func() {
		aws := getAWSImplmentation()
		var volumeID string

		BeforeEach(func() {
			Expect(localDiskImagePath).ToNot(BeEmpty(), "Expected LOCAL_DISK_IMAGE_PATH to be set")

			taskInfo, err := ourEC2.ImportVolume(aws, localDiskImagePath)
			Expect(err).ToNot(HaveOccurred())
			volumeID = taskInfo.EBSVolumeID
			Expect(volumeID).ToNot(BeEmpty())

			err = ourEC2.CleanupImportVolume(aws, taskInfo.TaskID)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := ourEC2.DeleteVolume(aws, volumeID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows an AMI to be created from an EBS volume then deleted", func() {
			amiConfig := ec2ami.Config{
				Region:             aws.GetConfig().Region,
				VirtualizationType: "hvm",
				Description:        "BOSH CI test AMI",
			}

			amiInfo, err := ourEC2.CreateAmi(aws, volumeID, amiConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(amiInfo.AmiID).ToNot(BeEmpty())

			Expect(amiInfo.Status()).To(Equal(ourEC2.VolumeAvailableStatus))
			Expect(amiInfo.Architecture).To(Equal(ec2ami.AmiArchitecture))
			Expect(amiInfo.VirtualizationType).To(Equal(amiConfig.VirtualizationType))
			Expect(amiInfo.Accessibility).To(Equal(ec2ami.AmiPrivateAccessibility))

			err = ourEC2.DeleteAmi(aws, amiInfo)
			Expect(err).ToNot(HaveOccurred())
		})

		It("makes the AMI public if desired", func() {
			amiConfig := ec2ami.Config{
				Region:             aws.GetConfig().Region,
				Public:             true,
				VirtualizationType: "hvm",
				Description:        "BOSH CI test AMI",
			}

			amiInfo, err := ourEC2.CreateAmi(aws, volumeID, amiConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(amiInfo.AmiID).ToNot(BeEmpty())

			statusInfo, err := aws.DescribeImage(&amiInfo.InputConfig)
			Expect(statusInfo).To(BeAssignableToTypeOf(amiInfo))
			newAmiInfo := statusInfo.(ec2ami.Info)
			Expect(newAmiInfo.Status()).To(Equal(ourEC2.VolumeAvailableStatus))
			Expect(newAmiInfo.Architecture).To(Equal(ec2ami.AmiArchitecture))
			Expect(newAmiInfo.VirtualizationType).To(Equal(amiConfig.VirtualizationType))
			Expect(newAmiInfo.Accessibility).To(Equal(ec2ami.AmiPublicAccessibility))

			err = ourEC2.DeleteAmi(aws, newAmiInfo)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("a published HVM AMI", func() {
			var numInstances int64 = 1
			var associatePublicIP bool = true
			var instanceType string = ec2.InstanceTypeM3Medium
			var networkDeviceIndex int64 = 0

			It("is bootable", func() {
				Expect(os.Getenv("AWS_ACCESS_KEY_ID")).ToNot(BeEmpty(), "Expected AWS_ACCESS_KEY_ID to be set")
				Expect(os.Getenv("AWS_SECRET_ACCESS_KEY")).ToNot(BeEmpty(), "Expected AWS_SECRET_ACCESS_KEY to be set")

				amiConfig := ec2ami.Config{
					Region:             aws.GetConfig().Region,
					VirtualizationType: "hvm",
					Description:        "BOSH CI test AMI",
				}

				amiInfo, err := ourEC2.CreateAmi(aws, volumeID, amiConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(amiInfo.AmiID).ToNot(BeEmpty())

				amiID := amiInfo.AmiID

				ec2Client := ec2.New(session.New(), &awssdk.Config{Region: awssdk.String(aws.GetConfig().Region)})

				instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
					ImageId:      &amiID,
					InstanceType: &instanceType,
					MinCount:     &numInstances,
					MaxCount:     &numInstances,
					NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
						&ec2.InstanceNetworkInterfaceSpecification{
							DeviceIndex:              &networkDeviceIndex,
							AssociatePublicIpAddress: &associatePublicIP,
						},
					},
				})
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

		Describe("a published Paravirtual AMI", func() {
			var numInstances int64 = 1
			var associatePublicIP bool = true
			var instanceType string = ec2.InstanceTypeM3Medium
			var networkDeviceIndex int64 = 0

			It("is bootable", func() {
				Expect(os.Getenv("AWS_ACCESS_KEY_ID")).ToNot(BeEmpty(), "Expected AWS_ACCESS_KEY_ID to be set")
				Expect(os.Getenv("AWS_SECRET_ACCESS_KEY")).ToNot(BeEmpty(), "Expected AWS_SECRET_ACCESS_KEY to be set")

				amiConfig := ec2ami.Config{
					Region:             aws.GetConfig().Region,
					VirtualizationType: "paravirtual",
					Description:        "BOSH CI test AMI",
				}

				amiInfo, err := ourEC2.CreateAmi(aws, volumeID, amiConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(amiInfo.AmiID).ToNot(BeEmpty())

				amiID := amiInfo.AmiID

				ec2Client := ec2.New(session.New(), &awssdk.Config{Region: awssdk.String(aws.GetConfig().Region)})

				instanceReservation, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
					ImageId:      &amiID,
					InstanceType: &instanceType,
					MinCount:     &numInstances,
					MaxCount:     &numInstances,
					NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
						&ec2.InstanceNetworkInterfaceSpecification{
							DeviceIndex:              &networkDeviceIndex,
							AssociatePublicIpAddress: &associatePublicIP,
						},
					},
				})
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
	})
})
