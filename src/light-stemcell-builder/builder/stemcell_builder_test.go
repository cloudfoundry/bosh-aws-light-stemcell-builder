package builder_test

import (
	"bytes"
	"io/ioutil"
	"light-stemcell-builder/builder"
	"light-stemcell-builder/ec2/ec2ami"
	"log"
	"os"
	"os/exec"
	"path"

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
	logger := log.New(os.Stdout, "", log.LstdFlags)
	copyDests := []string{"us-west-1", "us-west-2"}
	stemcellPath := os.Getenv("HEAVY_STEMCELL_TARBALL")
	outputPath := os.Getenv("OUTPUT_STEMCELL_PATH")
	var dummyStemcellPath string

	dummyManifest := &bytes.Buffer{}
	dummyManifest.WriteString("---\n")
	dummyManifest.WriteString("name: bosh-aws-xen-ubuntu-trusty-go_agent\n")
	dummyManifest.WriteString("cloud_properties:\n")
	dummyManifest.WriteString("  name: bosh-aws-xen-ubuntu-trusty-go_agent")

	BeforeSuite(func() {
		// TODO: Test light stemcell building in AWS China
		Expect(awsConfig.Region).ToNot(Equal("cn-north-1"), "Cannot copy stemcells from China to US regions")

		dummyStemcellFolder, err := ioutil.TempDir("", "light-stemcell-builder-test")
		Expect(err).ToNot(HaveOccurred())

		for _, filename := range []string{"root.img", "apply_spec.yml", "stemcell_dpkg_l.txt"} {
			filePath := path.Join(dummyStemcellFolder, filename)
			touchFile, err := os.Create(filePath)
			Expect(err).ToNot(HaveOccurred())
			err = touchFile.Close()
			Expect(err).ToNot(HaveOccurred())
		}
		imagePath := path.Join(dummyStemcellFolder, "image")
		tarCmd := exec.Command("tar", "-C", dummyStemcellFolder, "-czf", imagePath, "--", "root.img")
		err = tarCmd.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(imagePath).To(BeAnExistingFile())

		dummyManifestPath := path.Join(dummyStemcellFolder, "stemcell.MF")
		dummyManifestFile, err := os.Create(dummyManifestPath)
		dummyManifestFile.Write(dummyManifest.Bytes())
		err = dummyManifestFile.Close()
		Expect(err).ToNot(HaveOccurred())

		dummyStemcellPath = path.Join(dummyStemcellFolder, "dummy-xen-stemcell.tgz")
		tarCmd = exec.Command("tar", "-C", dummyStemcellFolder, "-czf", dummyStemcellPath, "--", "image", "apply_spec.yml", "stemcell.MF", "stemcell_dpkg_l.txt")
		err = tarCmd.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(dummyStemcellPath).To(BeAnExistingFile())

		Expect(stemcellPath).ToNot(BeEmpty())
		Expect(stemcellPath).To(BeAnExistingFile())
		Expect(outputPath).ToNot(BeEmpty())
		Expect(outputPath).To(BeADirectory())
	})

	Describe("Importing a machine image into AWS", func() {
		Context("when executed as a 'dry run'", func() {
			amiConfig := ec2ami.Config{
				Description:        "BOSH Stemcell Builder Test AMI",
				Public:             false,
				VirtualizationType: "hvm",
				Region:             awsConfig.Region,
			}
			b := builder.New(logger, awsConfig, amiConfig)
			// TODO: remove this once we implement and use a fake EC2Cli
			b.DryRun()

			It("successfully builds a light stemcell, minus the actual AMI creation", func() {
				outputFile, amis, err := b.BuildLightStemcell(dummyStemcellPath, outputPath, copyDests)
				Expect(err).ToNot(HaveOccurred())

				Expect(amis).To(HaveKey(amiConfig.Region))
				Expect(amis).To(HaveKey("us-west-1"))
				Expect(amis).To(HaveKey("us-west-2"))
				Expect(amis[amiConfig.Region].AmiID).To(MatchRegexp("ami-.*"))
				Expect(amis["us-west-1"].AmiID).To(MatchRegexp("ami-.*"))
				Expect(amis["us-west-2"].AmiID).To(MatchRegexp("ami-.*"))

				Expect(outputFile).To(BeAnExistingFile())
			})
		})
	})

	Describe("Prepare", func() {
		It("prepares the work dir, with contents from the 'heavy' stemcell", func() {
			b := builder.New(logger, awsConfig, ec2ami.Config{})

			imagePath, err := b.Prepare(dummyStemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))

			Expect(imagePath).To(BeAnExistingFile())
		})
	})

	Describe("PackageLightStemcell", func() {
		It("produces a light stemcell tarball", func() {
			b := builder.New(logger, awsConfig, ec2ami.Config{})
			packageDir, err := ioutil.TempDir("", "light-stemcell-builder-package-test")
			Expect(err).ToNot(HaveOccurred())

			_, err = b.Prepare(dummyStemcellPath)
			Expect(err).ToNot(HaveOccurred())

			outputPackage := path.Join(packageDir, "package.tgz")
			err = b.PackageLightStemcell(outputPackage)
			Expect(err).ToNot(HaveOccurred())

			Expect(outputPackage).To(BeAnExistingFile())
			untar := exec.Command("tar", "-C", packageDir, "-xf", outputPackage)
			err = untar.Run()
			Expect(err).ToNot(HaveOccurred())

			imagePath := path.Join(packageDir, "image")
			manifestPath := path.Join(packageDir, "stemcell.MF")
			Expect(imagePath).To(BeAnExistingFile())
			Expect(manifestPath).To(BeAnExistingFile())

			imageInfo, err := os.Stat(imagePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imageInfo.Size()).To(Equal(int64(0)))
		})
		It("errors if the Prepare() has not yet been called", func() {
			b := builder.New(logger, awsConfig, ec2ami.Config{})
			packageDir, err := ioutil.TempDir("", "light-stemcell-builder-package-test")
			Expect(err).ToNot(HaveOccurred())

			outputPackage := path.Join(packageDir, "package.tgz")
			err = b.PackageLightStemcell(outputPackage)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Please call Prepare() before PackageLightStemcell"))
		})
	})

	Describe("UpdateManifestFile", func() {
		amiConfig := ec2ami.Config{VirtualizationType: "hvm"}
		It("correctly updates the manifest file", func() {
			b := builder.New(logger, awsConfig, amiConfig)
			manifestFile := bytes.NewBuffer(dummyManifest.Bytes())

			regionToAmi := map[string]string{
				"us-east-1": "ami-some-id",
			}

			err := b.UpdateManifestFile(manifestFile, regionToAmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(manifestFile.String()).To(MatchRegexp("(?m)^name: bosh-aws-xen-hvm-ubuntu-trusty-go_agent$"))
			Expect(manifestFile.String()).To(MatchRegexp("(?m)^cloud_properties:$"))
			Expect(manifestFile.String()).To(MatchRegexp("(?m)^  name: bosh-aws-xen-hvm-ubuntu-trusty-go_agent$"))
			Expect(manifestFile.String()).To(MatchRegexp("(?m)^  ami:$"))
			Expect(manifestFile.String()).To(MatchRegexp("(?m)^    us-east-1: ami-some-id$"))
		})
	})

	Describe("UpdateManifestContent", func() {
		Context("given a HVM stemcell", func() {
			amiConfig := ec2ami.Config{VirtualizationType: "hvm"}
			It("outputs the correct manifest", func() {
				b := builder.New(logger, awsConfig, amiConfig)

				manifest := map[string]interface{}{
					"name": "stemcell-xen-name",
					"cloud_properties": map[string]interface{}{
						"name": "stemcell-xen-name",
					},
				}

				regionToAmi := map[string]string{
					"us-east-1": "ami-some-id",
				}

				expectedManifest := map[string]interface{}{
					"name": "stemcell-xen-hvm-name",
					"cloud_properties": map[string]interface{}{
						"name": "stemcell-xen-hvm-name",
						"ami": map[string]string{
							"us-east-1": "ami-some-id",
						},
					},
				}

				err := b.UpdateManifestContent(manifest, regionToAmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(manifest).To(Equal(expectedManifest))
			})
		})
		Context("given a non-HVM stemcell", func() {
			amiConfig := ec2ami.Config{VirtualizationType: "non-hvm"}
			It("outputs the correct manifest", func() {
				b := builder.New(logger, awsConfig, amiConfig)

				manifest := map[string]interface{}{
					"name": "stemcell-xen-name",
					"cloud_properties": map[string]interface{}{
						"name": "stemcell-xen-name",
					},
				}

				regionToAmi := map[string]string{
					"us-east-1": "ami-some-id",
				}

				expectedManifest := map[string]interface{}{
					"name": "stemcell-xen-name",
					"cloud_properties": map[string]interface{}{
						"name": "stemcell-xen-name",
						"ami": map[string]string{
							"us-east-1": "ami-some-id",
						},
					},
				}

				err := b.UpdateManifestContent(manifest, regionToAmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(manifest).To(Equal(expectedManifest))
			})
		})
		Context("given an invalid manifest", func() {
			amiConfig := ec2ami.Config{}
			b := builder.New(logger, awsConfig, amiConfig)
			It("errors with missing 'name'", func() {
				manifest := make(map[string]interface{})
				err := b.UpdateManifestContent(manifest, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Manifest missing 'name'"))
			})
			It("errors with missing 'cloud_properties'", func() {
				manifest := map[string]interface{}{
					"name": "stemcell-xen-name",
				}
				err := b.UpdateManifestContent(manifest, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Manifest missing 'cloud_properties'"))
			})
			It("errors with missing 'cloud_properties: name'", func() {
				manifest := map[string]interface{}{
					"name":             "stemcell-xen-name",
					"cloud_properties": make(map[string]interface{}),
				}
				err := b.UpdateManifestContent(manifest, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Manifest missing 'cloud_properties: name'"))
			})
		})
	})
	Describe("LightStemcellFilePath", func() {
		heavyStemcellPath := "some-xen-stemcell.tgz"
		outputPath := "/path/to/stemcell/"

		Context("given a HVM stemcell", func() {
			amiConfig := ec2ami.Config{
				Description:        "BOSH Stemcell Builder Test AMI",
				Public:             false,
				VirtualizationType: "hvm",
				Region:             awsConfig.Region,
			}
			It("returns the expected file path", func() {
				b := builder.New(logger, awsConfig, amiConfig)
				lightStemcellPath := b.LightStemcellFilePath(heavyStemcellPath, outputPath)
				Expect(lightStemcellPath).To(Equal("/path/to/stemcell/light-some-xen-hvm-stemcell.tgz"))
			})
		})
		Context("given a non-HVM stemcell", func() {
			amiConfig := ec2ami.Config{
				Description:        "BOSH Stemcell Builder Test AMI",
				Public:             false,
				VirtualizationType: "non-hvm",
				Region:             awsConfig.Region,
			}
			It("returns the expected file path", func() {
				b := builder.New(logger, awsConfig, amiConfig)
				lightStemcellPath := b.LightStemcellFilePath(heavyStemcellPath, outputPath)
				Expect(lightStemcellPath).To(Equal("/path/to/stemcell/light-some-xen-stemcell.tgz"))
			})
		})
	})
})
