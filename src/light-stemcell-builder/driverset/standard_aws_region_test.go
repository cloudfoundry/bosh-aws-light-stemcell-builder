package driverset_test

import (
	"light-stemcell-builder/config"
	"light-stemcell-builder/driver"
	"light-stemcell-builder/driverset"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StandardAwsRegion", func() {
	It("returns drivers of the correct type", func() {

		creds := config.Credentials{}
		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)

		Expect(ds.MachineImageDriver()).To(BeAssignableToTypeOf(struct {
			*driver.SDKCreateMachineImageDriver
			*driver.SDKDeleteMachineImageDriver
		}{}))
		Expect(ds.CreateSnapshotDriver()).To(BeAssignableToTypeOf(&driver.SDKSnapshotFromImageDriver{}))
		Expect(ds.CreateAmiDriver()).To(BeAssignableToTypeOf(&driver.SDKCreateAmiDriver{}))
		Expect(ds.CopyAmiDriver()).To(BeAssignableToTypeOf(&driver.SDKCopyAmiDriver{}))
	})
})
