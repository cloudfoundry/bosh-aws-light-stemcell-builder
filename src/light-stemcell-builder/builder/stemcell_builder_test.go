package builder_test

import (
	"light-stemcell-builder/builder"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var awsConfig = builder.AwsConfig{
	AccessKey:  os.Getenv("AWS_ACCESS_KEY_ID"),
	SecretKey:  os.Getenv("AWS_ACCESS_SECRET_KEY"),
	BucketName: os.Getenv("AWS_BUCKET_NAME"),
	Region:     os.Getenv("AWS_REGION"),
}

var _ = Describe("StemcellBuilder", func() {
	Describe("Preparing a heavy stemcell for import", func() {
		It("extracts the machine image and returns the path", func() {
			stemcellPath := os.Getenv("HEAVY_STEMCELL_TARBALL")
			Expect(stemcellPath).ToNot(BeEmpty())

			b, err := builder.New(awsConfig)
			Expect(err).ToNot(HaveOccurred())

			imagePath, err := b.PrepareHeavy(stemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))

			_, err = os.Stat(imagePath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Importing a machine image into AWS", func() {
		It("Creates an EBS volume inside of AWS", func() {
			stemcellPath := os.Getenv("HEAVY_STEMCELL_TARBALL")
			Expect(stemcellPath).ToNot(BeEmpty())

			b, err := builder.New(awsConfig)
			Expect(err).ToNot(HaveOccurred())

			imagePath, err := b.PrepareHeavy(stemcellPath)
			Expect(err).ToNot(HaveOccurred())

			amiID, err := b.ImportImage(imagePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(amiID).To(ContainSubstring("import-vol-"))
		})

		It("Creates an AMI from an EBS volume", func() {

		})

		It("Makes the AMI public if desired", func() {

		})

		It("Clones the AMI to other regions if desired", func() {

		})

		It("Creates an AMI from a Machine Image in all desired regions", func() {

		})
	})
})
