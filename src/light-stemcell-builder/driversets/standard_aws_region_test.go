package driversets_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/drivers"
	"light-stemcell-builder/driversets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StandardAwsRegion", func() {
	It("returns drivers of the correct type", func() {

		creds := config.Credentials{}
		ds := driversets.NewStandardRegionDriverSet(GinkgoWriter, creds)

		Expect(ds.CreateMachineImageDriver()).To(BeAssignableToTypeOf(&drivers.SDKMachineImageDriver{}))
		Expect(ds.CreateSnapshotDriver()).To(BeAssignableToTypeOf(&drivers.SDKSnapshotFromImageDriver{}))
		Expect(ds.CreateAmiDriver()).To(BeAssignableToTypeOf(&drivers.SDKCreateAmiDriver{}))
		Expect(ds.CopyAmiDriver()).To(BeAssignableToTypeOf(&drivers.SDKCopyAmiDriver{}))
	})
})
