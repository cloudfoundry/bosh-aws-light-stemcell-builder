package resources_test

import (
	"errors"
	"light-stemcell-builder/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeCreateAmiDriver struct {
	callCounter  int
	passedConfig resources.AmiDriverConfig
}

func (d *fakeCreateAmiDriver) Create(c resources.AmiDriverConfig) (string, error) {
	d.callCounter++
	d.passedConfig = c

	if d.callCounter > 1 {
		return "", errors.New("called multiple times")
	}

	return "a-new-ami-id", nil
}

var _ = Describe("Ami", func() {
	Describe("Wait", func() {
		It("calls create on the driver exactly once", func() {
			fakeConfig := resources.AmiDriverConfig{
				SnapshotID: "some-snapshot-id",
			}
			fakeDriver := &fakeCreateAmiDriver{}

			ami := resources.NewAmi(fakeDriver, fakeConfig)
			amiID, err := ami.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(amiID).To(Equal("a-new-ami-id"))
			Expect(fakeDriver.callCounter).To(Equal(1))
			Expect(fakeDriver.passedConfig).To(Equal(fakeConfig))
		})

		It("is idempotent", func() {
			emptyConfig := resources.AmiDriverConfig{}
			fakeDriver := &fakeCreateAmiDriver{}

			ami := resources.NewAmi(fakeDriver, emptyConfig)
			amiID, err := ami.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(amiID).To(Equal("a-new-ami-id"))
			Expect(fakeDriver.callCounter).To(Equal(1))

			Consistently(ami.WaitForCreation).Should(Equal("a-new-ami-id"))
		})
	})
})
