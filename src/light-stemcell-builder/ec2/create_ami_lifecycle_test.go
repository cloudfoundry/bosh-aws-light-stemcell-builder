package ec2_test

import (
	"fmt"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2instance"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateAmi lifecycle", func() {
	Describe("creating and deleting an ami", func() {
		aws := getAWSImplmentation()
		var volumeID string

		BeforeEach(func() {
			Expect(localDiskImagePath).ToNot(BeEmpty(), "Expected LOCAL_DISK_IMAGE_PATH to be set")

			taskInfo, err := ec2.ImportVolume(aws, localDiskImagePath)
			Expect(err).ToNot(HaveOccurred())
			volumeID = taskInfo.EBSVolumeID
			Expect(volumeID).ToNot(BeEmpty())

			err = ec2.CleanupImportVolume(aws, taskInfo.TaskID)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := ec2.DeleteVolume(aws, volumeID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows an AMI to be created from an EBS volume then deleted", func() {
			amiConfig := ec2ami.Config{
				Region:             aws.GetConfig().Region,
				VirtualizationType: "hvm",
				Description:        "BOSH CI test AMI",
			}

			amiInfo, err := ec2.CreateAmi(aws, volumeID, amiConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(amiInfo.AmiID).ToNot(BeEmpty())

			Expect(amiInfo.Status()).To(Equal(ec2.VolumeAvailableStatus))
			Expect(amiInfo.Architecture).To(Equal(ec2ami.AmiArchitecture))
			Expect(amiInfo.VirtualizationType).To(Equal(amiConfig.VirtualizationType))
			Expect(amiInfo.Accessibility).To(Equal(ec2ami.AmiPrivateAccessibility))

			err = ec2.DeleteAmi(aws, amiInfo)
			Expect(err).ToNot(HaveOccurred())
		})

		It("makes the AMI public if desired", func() {
			amiConfig := ec2ami.Config{
				Region:             aws.GetConfig().Region,
				Public:             true,
				VirtualizationType: "hvm",
				Description:        "BOSH CI test AMI",
			}

			amiInfo, err := ec2.CreateAmi(aws, volumeID, amiConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(amiInfo.AmiID).ToNot(BeEmpty())

			statusInfo, err := aws.DescribeImage(&amiInfo.InputConfig)
			Expect(statusInfo).To(BeAssignableToTypeOf(amiInfo))
			newAmiInfo := statusInfo.(ec2ami.Info)
			Expect(newAmiInfo.Status()).To(Equal(ec2.VolumeAvailableStatus))
			Expect(newAmiInfo.Architecture).To(Equal(ec2ami.AmiArchitecture))
			Expect(newAmiInfo.VirtualizationType).To(Equal(amiConfig.VirtualizationType))
			Expect(newAmiInfo.Accessibility).To(Equal(ec2ami.AmiPublicAccessibility))

			err = ec2.DeleteAmi(aws, newAmiInfo)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("a published HVM AMI", func() {
			It("is bootable", func() {
				amiConfig := ec2ami.Config{
					Region:             aws.GetConfig().Region,
					VirtualizationType: "hvm",
					Description:        "BOSH CI test AMI",
				}

				amiInfo, err := ec2.CreateAmi(aws, volumeID, amiConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(amiInfo.AmiID).ToNot(BeEmpty())

				amiID := amiInfo.AmiID
				instanceConfig := ec2instance.Config{
					AmiID:             amiID,
					InstanceType:      "t2.micro",
					AssociatePublicIP: true,
					Region:            aws.GetConfig().Region,
				}
				instance, err := ec2.RunInstance(aws, instanceConfig)
				Expect(err).ToNot(HaveOccurred())

				conn, err := net.DialTimeout(
					"tcp",
					fmt.Sprintf("%s:22", instance.PublicIP),
					10*time.Second,
				)
				Expect(err).ToNot(HaveOccurred())
				err = conn.Close()
				Expect(err).ToNot(HaveOccurred())

				err = ec2.TerminateInstance(aws, instance)
				Expect(err).ToNot(HaveOccurred())

				err = ec2.DeleteAmi(aws, amiInfo)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("a published Paravirtual AMI", func() {
			It("is bootable", func() {
				amiConfig := ec2ami.Config{
					Region:             aws.GetConfig().Region,
					VirtualizationType: "paravirtual",
					Description:        "BOSH CI test AMI",
				}

				amiInfo, err := ec2.CreateAmi(aws, volumeID, amiConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(amiInfo.AmiID).ToNot(BeEmpty())

				amiID := amiInfo.AmiID
				instanceConfig := ec2instance.Config{
					AmiID:             amiID,
					InstanceType:      "t1.micro", // pv stemcells do not support t2.*
					AssociatePublicIP: true,
					Region:            aws.GetConfig().Region,
				}
				instance, err := ec2.RunInstance(aws, instanceConfig)
				Expect(err).ToNot(HaveOccurred())

				conn, err := net.DialTimeout(
					"tcp",
					fmt.Sprintf("%s:22", instance.PublicIP),
					10*time.Second,
				)
				Expect(err).ToNot(HaveOccurred())
				err = conn.Close()
				Expect(err).ToNot(HaveOccurred())

				err = ec2.TerminateInstance(aws, instance)
				Expect(err).ToNot(HaveOccurred())

				err = ec2.DeleteAmi(aws, amiInfo)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
