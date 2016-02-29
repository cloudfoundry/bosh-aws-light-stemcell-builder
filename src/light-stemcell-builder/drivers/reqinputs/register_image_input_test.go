package reqinputs_test

import (
	"light-stemcell-builder/drivers/reqinputs"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("building inputs for register image", func() {
	Describe("NewHVMAmiRequestInput", func() {
		It("builds valid request input for building an HVM AMI", func() {
			input := reqinputs.NewHVMAmiRequestInput("some-ami-name", "some-ami-description", "some-snapshot-id")
			Expect(input).To(BeAssignableToTypeOf(&ec2.RegisterImageInput{}))
			Expect(*input.Architecture).To(Equal(resources.AmiArchitecture))
			Expect(*input.Description).To(Equal("some-ami-description"))
			Expect(*input.VirtualizationType).To(Equal(resources.HvmAmiVirtualization))
			Expect(*input.Name).To(Equal("some-ami-name"))
			Expect(*input.RootDeviceName).To(Equal("/dev/xvda"))
			Expect(input.BlockDeviceMappings).To(HaveLen(1))
			Expect(*input.BlockDeviceMappings[0].DeviceName).To(Equal("/dev/xvda"))
			Expect(*input.BlockDeviceMappings[0].Ebs.SnapshotId).To(Equal("some-snapshot-id"))
			Expect(*input.BlockDeviceMappings[0].Ebs.DeleteOnTermination).To(BeTrue())
		})
	})

	Describe("NewPVAmiRequest", func() {
		It("builds valid request input for building an PV AMI", func() {
			input := reqinputs.NewPVAmiRequest("some-ami-name", "some-ami-description", "some-snapshot-id", "some-kernel-id")
			Expect(input).To(BeAssignableToTypeOf(&ec2.RegisterImageInput{}))
			Expect(*input.Architecture).To(Equal(resources.AmiArchitecture))
			Expect(*input.Description).To(Equal("some-ami-description"))
			Expect(*input.VirtualizationType).To(Equal(resources.PvAmiVirtualization))
			Expect(*input.Name).To(Equal("some-ami-name"))
			Expect(*input.RootDeviceName).To(Equal("/dev/sda1"))
			Expect(*input.KernelId).To(Equal("some-kernel-id"))
			Expect(input.BlockDeviceMappings).To(HaveLen(1))
			Expect(*input.BlockDeviceMappings[0].DeviceName).To(Equal("/dev/sda"))
			Expect(*input.BlockDeviceMappings[0].Ebs.SnapshotId).To(Equal("some-snapshot-id"))
			Expect(*input.BlockDeviceMappings[0].Ebs.DeleteOnTermination).To(BeTrue())
		})
	})

})
