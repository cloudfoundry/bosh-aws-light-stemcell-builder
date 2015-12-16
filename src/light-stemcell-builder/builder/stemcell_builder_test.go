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
	Describe("Importing a machine image into AWS", func() {
		It("Creates an AMI from a heavy stemcell in all desired regions and can delete them", func() {
			Expect(awsConfig.Region).ToNot(Equal("cn-north-1"), "Cannot copy stemcells from China to US regions")

			stemcellPath := os.Getenv("HEAVY_STEMCELL_TARBALL")
			Expect(stemcellPath).ToNot(BeEmpty())

			b, err := builder.New(awsConfig)
			Expect(err).ToNot(HaveOccurred())

			imagePath, err := b.PrepareHeavy(stemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))

			_, err = os.Stat(imagePath)
			Expect(err).ToNot(HaveOccurred())

			copyDests := []string{"us-west-1", "us-west-2"}
			amis, err := b.BuildLightStemcells(imagePath, awsConfig, copyDests)
			Expect(err).ToNot(HaveOccurred())
			Expect(amis).To(HaveKey(awsConfig.Region))
			Expect(amis).To(HaveKey("us-west-1"))
			Expect(amis).To(HaveKey("us-west-2"))
			Expect(amis[awsConfig.Region].AmiID).To(MatchRegexp("ami-.*"))
			Expect(amis["us-west-1"].AmiID).To(MatchRegexp("ami-.*"))
			Expect(amis["us-west-2"].AmiID).To(MatchRegexp("ami-.*"))

			err = b.DeleteLightStemcells(awsConfig, amis)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
