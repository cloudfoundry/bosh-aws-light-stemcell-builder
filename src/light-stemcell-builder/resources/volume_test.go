package resources_test

import (
	"errors"
	"light-stemcell-builder/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeVolumeImageDriver struct {
	callCounter  int
	passedConfig resources.VolumeDriverConfig
}

func (d *fakeVolumeImageDriver) Create(c resources.VolumeDriverConfig) (string, error) {
	d.callCounter++
	d.passedConfig = c

	if d.callCounter > 1 {
		return "", errors.New("called multiple times")
	}

	return "a-new-volume-id", nil
}

var _ = Describe("Volume", func() {
	Describe("Wait", func() {
		It("calls create on the driver exactly once", func() {
			fakeConfig := resources.VolumeDriverConfig{
				MachineImageManifestURL: "some-manifest-url",
			}
			fakeDriver := &fakeVolumeImageDriver{}

			volume := resources.NewVolume(fakeDriver, fakeConfig)
			volumeID, err := volume.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(volumeID).To(Equal("a-new-volume-id"))
			Expect(fakeDriver.callCounter).To(Equal(1))
			Expect(fakeDriver.passedConfig).To(Equal(fakeConfig))
		})

		It("is idempotent", func() {
			emptyConfig := resources.VolumeDriverConfig{}
			fakeDriver := &fakeVolumeImageDriver{}

			volume := resources.NewVolume(fakeDriver, emptyConfig)
			volumeID, err := volume.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(volumeID).To(Equal("a-new-volume-id"))
			Expect(fakeDriver.callCounter).To(Equal(1))

			Consistently(volume.WaitForCreation).Should(Equal("a-new-volume-id"))
		})
	})
})
