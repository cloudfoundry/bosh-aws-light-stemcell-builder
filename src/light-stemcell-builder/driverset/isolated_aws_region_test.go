package driverset_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/driverset"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsolatedAwsRegion", func() {
	It("returns drivers of the correct type", func() {

		creds := config.Credentials{}
		ds := driverset.NewIsolatedRegionDriverSet(GinkgoWriter, creds)

		Expect(ds.MachineImageDriver()).To(BeAssignableToTypeOf(struct {
			*driver.SDKCreateMachineImageManifestDriver
			*driver.SDKDeleteMachineImageDriver
		}{}))
		Expect(ds.VolumeDriver()).To(BeAssignableToTypeOf(struct {
			*driver.SDKCreateVolumeDriver
			*driver.SDKDeleteVolumeDriver
		}{}))
		Expect(ds.CreateSnapshotDriver()).To(BeAssignableToTypeOf(&driver.SDKSnapshotFromVolumeDriver{}))
		Expect(ds.CreateAmiDriver()).To(BeAssignableToTypeOf(&driver.SDKCreateAmiDriver{}))
	})
})
