package reqinputs_test

import (
	"light-stemcell-builder/driver/reqinputs"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("building inputs for register image", func() {
	Describe("NewHVMAmiRequestInput", func() {
		It("builds valid request input for building an HVM AMI", func() {
			input := reqinputs.NewHVMAmiRequestInput("some-ami-name", "some-ami-description", "some-snapshot-id", false)
			Expect(input).To(BeAssignableToTypeOf(&ec2.RegisterImageInput{}))
			Expect(*input.SriovNetSupport).To(Equal("simple"))
			Expect(*input.Architecture).To(Equal(resources.AmiArchitecture))
			Expect(*input.Description).To(Equal("some-ami-description"))
			Expect(*input.VirtualizationType).To(Equal(resources.HvmAmiVirtualization))
			Expect(*input.Name).To(Equal("some-ami-name"))
			Expect(*input.BootMode).To(Equal("legacy-bios"))
			Expect(*input.RootDeviceName).To(Equal("/dev/xvda"))
			Expect(input.BlockDeviceMappings).To(HaveLen(1))
			Expect(*input.BlockDeviceMappings[0].DeviceName).To(Equal("/dev/xvda"))
			Expect(*input.BlockDeviceMappings[0].Ebs.SnapshotId).To(Equal("some-snapshot-id"))
			Expect(*input.BlockDeviceMappings[0].Ebs.DeleteOnTermination).To(BeTrue())
		})

		It("sets bootmode correctly when efi is true", func() {
			input := reqinputs.NewHVMAmiRequestInput("some-ami-name", "some-ami-description", "some-snapshot-id", true)
			Expect(*input.BootMode).To(Equal("uefi"))
		})
	})
})
