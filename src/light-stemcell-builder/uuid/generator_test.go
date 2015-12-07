package uuid_test

import (
	"light-stemcell-builder/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("New", func() {
	It("returns a new uuid with prefix", func() {
		uuid, err := uuid.New("BOSH")
		Expect(err).ToNot(HaveOccurred())
		Expect(uuid).To(ContainSubstring("BOSH"))
	})

	It("returns a new uuid without prefix", func() {
		uuid, err := uuid.New("")
		Expect(err).ToNot(HaveOccurred())
		Expect(uuid).ToNot(BeEmpty())
	})
})
