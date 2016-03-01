package driversets_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers"
	"light-stemcell-builder/driversets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsolatedAwsRegion", func() {
	It("returns drivers of the correct type", func() {

		creds := config.Credentials{}
		ds := driversets.NewIsolatedRegionDriverSet(GinkgoWriter, creds)

		Expect(ds.CreateMachineImageDriver()).To(BeAssignableToTypeOf(&drivers.SDKMachineImageManifestDriver{}))
		Expect(ds.CreateVolumeDriver()).To(BeAssignableToTypeOf(&drivers.SDKVolumeDriver{}))
		Expect(ds.CreateSnapshotDriver()).To(BeAssignableToTypeOf(&drivers.SDKSnapshotFromVolumeDriver{}))
		Expect(ds.CreateAmiDriver()).To(BeAssignableToTypeOf(&drivers.SDKCreateAmiDriver{}))
	})
})
