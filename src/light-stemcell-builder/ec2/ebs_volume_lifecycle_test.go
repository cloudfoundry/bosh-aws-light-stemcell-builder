package ec2_test

import (
	"fmt"
	"light-stemcell-builder/ec2"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateEBSVolume lifecycle", func() {
	aws := getAWSImplmentation()

	It("allows an EBS volume to be created from a machine image then deleted", func() {
		Expect(aws.GetConfig().Region).ToNot(Equal("cn-north-1"), "due to a bug in the ec2 cli, cleaning fails against v4 signing regions such as China")

		Expect(localDiskImagePath).ToNot(BeEmpty())

		taskInfo, err := ec2.ImportVolume(aws, localDiskImagePath)
		volID := taskInfo.EBSVolumeID
		Expect(err).ToNot(HaveOccurred())
		Expect(volID).ToNot(BeEmpty())

		resp, err := http.Get(taskInfo.ManifestUrl)
		Expect(err).ToNot(HaveOccurred())

		err = resp.Body.Close()
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).ToNot(Equal(404))

		err = ec2.CleanupImportVolume(aws, taskInfo.TaskID)
		Expect(err).ToNot(HaveOccurred())

		resp, err = http.Get(taskInfo.ManifestUrl)
		Expect(err).ToNot(HaveOccurred())

		err = resp.Body.Close()
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(404))

		volumeInfo, err := aws.DescribeVolume(ec2.VolumeResource{VolumeID: volID})
		Expect(err).ToNot(HaveOccurred())

		Expect(volumeInfo.Status()).To(Equal(ec2.VolumeAvailableStatus))

		err = ec2.DeleteVolume(aws, volID)
		Expect(err).ToNot(HaveOccurred())

		_, err = aws.DescribeVolume(ec2.VolumeResource{VolumeID: volID})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(fmt.Sprintf("volume with id: %s is not available due to status: unknown", volID)))
	})
})
