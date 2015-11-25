package builder_test

import (
	"light-stemcell-builder/builder"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StemcellBuilder", func() {
	Describe("Preparing a heavy stemcell for import", func() {
		It("extracts the machine image and returns the path", func() {
			stemcellPath := os.Getenv("HEAVY_STEMCELL_TARBALL")
			Expect(stemcellPath).ToNot(BeEmpty())

			b, err := builder.New()
			Expect(err).ToNot(HaveOccurred())

			imagePath, err := b.PrepareHeavy(stemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))

			_, err = os.Stat(imagePath)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
