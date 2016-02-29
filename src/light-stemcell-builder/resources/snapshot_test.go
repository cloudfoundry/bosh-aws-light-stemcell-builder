package resources_test

import (
	"errors"
	"light-stemcell-builder/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeSnapshotDriver struct {
	callCounter  int
	passedConfig resources.SnapshotDriverConfig
}

func (d *fakeSnapshotDriver) Create(c resources.SnapshotDriverConfig) (string, error) {
	d.callCounter++
	d.passedConfig = c

	if d.callCounter > 1 {
		return "", errors.New("called multiple times")
	}

	return "a-new-snapshot-id", nil
}

var _ = Describe("Snapshot", func() {
	Describe("Wait", func() {
		It("calls create on the driver exactly once", func() {
			fakeConfig := resources.SnapshotDriverConfig{
				VolumeID: "some-existing-volume-id",
			}
			fakeDriver := &fakeSnapshotDriver{}

			snapshot := resources.NewSnapshot(fakeDriver, fakeConfig)
			snapshotID, err := snapshot.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(snapshotID).To(Equal("a-new-snapshot-id"))
			Expect(fakeDriver.callCounter).To(Equal(1))
			Expect(fakeDriver.passedConfig).To(Equal(fakeConfig))
		})

		It("is idempotent", func() {
			emptyConfig := resources.SnapshotDriverConfig{}
			fakeDriver := &fakeSnapshotDriver{}

			snapshot := resources.NewSnapshot(fakeDriver, emptyConfig)
			snapshotID, err := snapshot.WaitForCreation()

			Expect(err).ToNot(HaveOccurred())
			Expect(snapshotID).To(Equal("a-new-snapshot-id"))
			Expect(fakeDriver.callCounter).To(Equal(1))

			Consistently(snapshot.WaitForCreation).Should(Equal("a-new-snapshot-id"))
		})
	})
})
