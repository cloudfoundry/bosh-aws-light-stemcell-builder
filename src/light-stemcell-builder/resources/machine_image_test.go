package resources_test

import (
	"errors"
	"light-stemcell-builder/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeMachineImageDriver struct {
	callCounter  int
	passedConfig resources.MachineImageDriverConfig
}

func (d *fakeMachineImageDriver) Create(c resources.MachineImageDriverConfig) (string, error) {
	d.callCounter++
	d.passedConfig = c

	if d.callCounter > 1 {
		return "", errors.New("called multiple times")
	}

	return "presigned-url", nil
}

var _ = Describe("MachineImage", func() {
	Describe("Wait", func() {
		It("calls create on the driver exactly once", func() {
			fakeConfig := resources.MachineImageDriverConfig{
				MachineImagePath: "some-path",
				BucketName:       "some-bucket",
			}
			fakeDriver := &fakeMachineImageDriver{}

			machineImage := resources.NewMachineImage(fakeDriver, fakeConfig)
			presignedURL, err := machineImage.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(presignedURL).To(Equal("presigned-url"))
			Expect(fakeDriver.callCounter).To(Equal(1))
			Expect(fakeDriver.passedConfig).To(Equal(fakeConfig))
		})

		It("is idempotent", func() {
			emptyConfig := resources.MachineImageDriverConfig{}
			fakeDriver := &fakeMachineImageDriver{}

			machineImage := resources.NewMachineImage(fakeDriver, emptyConfig)
			presignedURL, err := machineImage.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(presignedURL).To(Equal("presigned-url"))
			Expect(fakeDriver.callCounter).To(Equal(1))

			Consistently(machineImage.WaitForCreation).Should(Equal("presigned-url"))
		})
	})
})
