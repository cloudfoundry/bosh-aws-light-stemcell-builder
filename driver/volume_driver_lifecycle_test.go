package driver_test

import (
	"time"

	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Volume Driver Lifecycle", func() {
	It("creates and deletes an EBS Volume from a previously uploaded machine image", func() {
		createMachineImageDriver := driver.NewCreateMachineImageManifestDriver(GinkgoWriter, awsSession, creds)
		machineImageDriverConfig := resources.MachineImageDriverConfig{
			MachineImagePath: machineImagePath,
			FileFormat:       machineImageFormat,
			BucketName:       bucketName,
			VolumeSizeGB:     3,
		}

		machineImage, err := createMachineImageDriver.Create(machineImageDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		volumeDriverConfig := resources.VolumeDriverConfig{
			MachineImageManifestURL: machineImage.GetURL,
		}

		createVolumeDriver := driver.NewCreateVolumeDriver(GinkgoWriter, awsSession, creds)

		volume, err := createVolumeDriver.Create(volumeDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(awsSession)

		reqOutput, err := ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{VolumeIds: []*string{aws.String(volume.ID)}})
		Expect(err).ToNot(HaveOccurred())

		Expect(reqOutput.Volumes).To(HaveLen(1))

		deleteVolumeDriver := driver.NewDeleteVolumeDriver(GinkgoWriter, awsSession, creds)

		err = deleteVolumeDriver.Delete(volume)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err = ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{VolumeIds: []*string{aws.String(volume.ID)}})
			return err
		}, 10*time.Minute, 10*time.Second).Should(MatchError(ContainSubstring("InvalidVolume.NotFound")))

		deleteMachineImageDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, awsSession, creds)
		_ = deleteMachineImageDriver.Delete(machineImage) // ignore error on cleanup
	})
})
