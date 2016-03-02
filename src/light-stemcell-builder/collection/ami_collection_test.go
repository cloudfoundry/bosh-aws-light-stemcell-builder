package collection_test

import (
	"light-stemcell-builder/collection"
	"light-stemcell-builder/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ami", func() {
	It("returns all added Amis", func() {
		amiCollection := collection.Ami{}

		fakeAmis := []resources.Ami{
			resources.Ami{
				ID:     "fake-0",
				Region: "fake-region-0",
			},
			resources.Ami{
				ID:     "fake-1",
				Region: "fake-region-1",
			},
		}

		for _, fakeAmi := range fakeAmis {
			amiCollection.Add(fakeAmi)
		}

		Expect(amiCollection.GetAll()).To(Equal(fakeAmis))
	})
})
