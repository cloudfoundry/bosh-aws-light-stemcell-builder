package builder_test

import (
	"light-stemcell-builder/builder"
	"light-stemcell-builder/ec2/ec2ami"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var awsConfig = builder.AwsConfig{
	AccessKey:  os.Getenv("AWS_ACCESS_KEY_ID"),
	SecretKey:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
	BucketName: os.Getenv("AWS_BUCKET_NAME"),
	Region:     os.Getenv("AWS_REGION"),
}

var _ = Describe("StemcellBuilder", func() {
	Describe("Importing a machine image into AWS", func() {
		It("Creates an AMI from a heavy stemcell in all desired regions and can delete them", func() {
			Expect(awsConfig.Region).ToNot(Equal("cn-north-1"), "Cannot copy stemcells from China to US regions")

			logger := log.New(os.Stdout, "", log.LstdFlags)

			stemcellPath := os.Getenv("HEAVY_STEMCELL_TARBALL")
			Expect(stemcellPath).ToNot(BeEmpty())

			outputPath := os.Getenv("OUTPUT_STEMCELL_PATH")
			Expect(outputPath).ToNot(BeEmpty())

			_, err := os.Stat(outputPath)
			Expect(err).To(HaveOccurred(), "Make sure the output stemcell file does not exist before running this test.")

			b, err := builder.New(awsConfig)
			Expect(err).ToNot(HaveOccurred())

			imagePath, err := b.PrepareHeavy(stemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))

			_, err = os.Stat(imagePath)
			Expect(err).ToNot(HaveOccurred())

			copyDests := []string{"us-west-1", "us-west-2"}
			amiConfig := ec2ami.Config{
				Description:        "BOSH Stemcell Builder Test AMI",
				Public:             false,
				VirtualizationType: "hvm",
				Region:             awsConfig.Region,
			}
			amis, err := b.BuildAmis(logger, imagePath, amiConfig, copyDests)
			Expect(err).ToNot(HaveOccurred())
			Expect(amis).To(HaveKey(awsConfig.Region))
			Expect(amis).To(HaveKey("us-west-1"))
			Expect(amis).To(HaveKey("us-west-2"))
			Expect(amis[awsConfig.Region].AmiID).To(MatchRegexp("ami-.*"))
			Expect(amis["us-west-1"].AmiID).To(MatchRegexp("ami-.*"))
			Expect(amis["us-west-2"].AmiID).To(MatchRegexp("ami-.*"))

			err = b.BuildLightStemcellTarball(outputPath, amis)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(outputPath)
			Expect(err).ToNot(HaveOccurred())

			err = b.DeleteLightStemcells(awsConfig, amis)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
