package ec2_test

import (
	"bytes"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"strings"
	"time"

	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateAmi lifecycle", func() {
	Describe("creating and deleting an ami", func() {
		aws := getAWSImplmentation()

		createEBSVolume := func(c ec2.Config) (string, error) {
			createVolCmd := exec.Command(
				"ec2-create-volume",
				"-O", c.Credentials.AccessKey,
				"-W", c.Credentials.SecretKey,
				"--region", c.Region,
				"-s", "1",
				"-z", "us-east-1a",
			)

			stderr := &bytes.Buffer{}
			createVolCmd.Stderr = stderr

			rawOut, err := createVolCmd.Output()
			if err != nil {
				return "", fmt.Errorf("Error creating test volume: %s, stderr %s", err, stderr)
			}

			out := string(rawOut)
			fields := strings.Fields(out)
			volumeID := fields[1]

			waiterConfig := ec2.WaiterConfig{
				Resource:      ec2.VolumeResource{VolumeID: volumeID},
				DesiredStatus: ec2.VolumeAvailableStatus,
				PollTimeout:   1 * time.Minute,
			}

			fmt.Printf("waiting for volume %s to be created\n", volumeID)
			_, err = ec2.WaitForStatus(aws.DescribeVolume, waiterConfig)

			return volumeID, nil
		}

		It("allows an AMI to be created from an EBS volume then deleted", func() {
			fmt.Println(fmt.Sprintf("%s", aws))
			amiConfig := ec2ami.Config{
				Region:             aws.GetConfig().Region,
				VirtualizationType: "hvm",
				Description:        "BOSH CI test AMI",
			}

			volumeID, err := createEBSVolume(aws.GetConfig())
			Expect(err).ToNot(HaveOccurred())
			amiInfo, err := ec2.CreateAmi(aws, volumeID, amiConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(amiInfo.AmiID).ToNot(BeEmpty())

			Expect(amiInfo.Status()).To(Equal(ec2.VolumeAvailableStatus))
			Expect(amiInfo.Architecture).To(Equal(ec2ami.AmiArchitecture))
			Expect(amiInfo.VirtualizationType).To(Equal(amiConfig.VirtualizationType))
			Expect(amiInfo.Accessibility).To(Equal(ec2ami.AmiPrivateAccessibility))

			err = ec2.DeleteAmi(aws, amiInfo)
			Expect(err).ToNot(HaveOccurred())
			err = ec2.DeleteVolume(aws, volumeID)
		})

		It("makes the AMI public if desired", func() {
			amiConfig := ec2ami.Config{
				Region:             aws.GetConfig().Region,
				Public:             true,
				VirtualizationType: "hvm",
				Description:        "BOSH CI test AMI",
			}

			volumeID, err := createEBSVolume(aws.GetConfig())
			Expect(err).ToNot(HaveOccurred())
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
			err = ec2.DeleteVolume(aws, volumeID)
		})
	})

})
