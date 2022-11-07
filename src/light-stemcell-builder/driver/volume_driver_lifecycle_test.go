package driver_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/resources"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Volume Driver Lifecycle", func() {
	It("creates and deletes an EBS Volume from a previously uploaded machine image", func() {
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

		machineImagePath := os.Getenv("MACHINE_IMAGE_PATH")
		Expect(machineImagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		machineImageFormat := os.Getenv("MACHINE_IMAGE_FORMAT")
		Expect(machineImageFormat).ToNot(BeEmpty(), "MACHINE_IMAGE_FORMAT must be set")

		bucketName := os.Getenv("AWS_BUCKET_NAME")
		Expect(bucketName).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")

		createMachineImageDriver := driver.NewCreateMachineImageManifestDriver(GinkgoWriter, creds)
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

		createVolumeDriver := driver.NewCreateVolumeDriver(GinkgoWriter, creds)

		volume, err := createVolumeDriver.Create(volumeDriverConfig)
		Expect(err).ToNot(HaveOccurred())

		ec2Client := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
		reqOutput, err := ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{VolumeIds: []*string{aws.String(volume.ID)}})
		Expect(err).ToNot(HaveOccurred())

		Expect(reqOutput.Volumes).To(HaveLen(1))

		deleteVolumeDriver := driver.NewDeleteVolumeDriver(GinkgoWriter, creds)

		err = deleteVolumeDriver.Delete(volume)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err = ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{VolumeIds: []*string{aws.String(volume.ID)}})
			return err
		}, 10*time.Minute, 10*time.Second).Should(MatchError(ContainSubstring("InvalidVolume.NotFound")))

		deleteMachineImageDriver := driver.NewDeleteMachineImageDriver(GinkgoWriter, creds)
		_ = deleteMachineImageDriver.Delete(machineImage) // ignore error on cleanup
	})
})
