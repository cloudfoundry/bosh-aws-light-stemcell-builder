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

	"light-stemcell-builder/ec2/fakes"

	"light-stemcell-builder/ec2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var awsConfig = builder.AwsConfig{
	AccessKey:  "DUMMY",
	SecretKey:  "DUMMY",
	BucketName: "DUMMY",
	Region:     "DUMMY",
}

var _ = Describe("StemcellBuilder", func() {
	dummyAWS := &fakes.FakeAWS{}

	makeStubbedAWS := func() *fakes.FakeAWS {
		aws := &fakes.FakeAWS{}
		conversionTaskInfo := ec2.ConversionTaskInfo{
			ConversionStatus: ec2.ConversionTaskCompletedStatus,
			EBSVolumeID:      "volume-id",
			TaskID:           "task-id",
		}
		describeSnapshotInfo := ec2.SnapshotInfo{
			SnapshotStatus: ec2.SnapshotCompletedStatus,
		}

		// setup for ImportVolume
		aws.ImportVolumeReturns("task-id", nil)
		aws.DescribeConversionTaskReturns(conversionTaskInfo, nil)

		// setup for DeleteVolume
		aws.DescribeVolumeReturns(ec2.VolumeInfo{}, ec2.NonAvailableVolumeError{})

		// setup for CreateAmi
		aws.CreateSnapshotReturns("snapshot-id", nil)
		aws.DescribeSnapshotReturns(describeSnapshotInfo, nil)
		aws.RegisterImageReturns("ami-id", nil)
		aws.RegisterImageStub = func(inputConfig ec2ami.Config, snapshotID string) (string, error) {
			return "ami-" + inputConfig.Region, nil
		}
		aws.DescribeImageStub = func(amiResource ec2.StatusResource) (ec2.StatusInfo, error) {
			amiConfig := amiResource.(*ec2ami.Config)
			describeImageInfo := ec2ami.Info{
				AmiID:       amiConfig.AmiID,
				Region:      amiConfig.Region,
				InputConfig: *amiConfig,
				ImageStatus: ec2ami.AmiAvailableStatus,
			}
			return describeImageInfo, nil
		}

		// setup for CopyAmi
		aws.CopyImageStub = func(inputConfig ec2ami.Config, dest string) (string, error) {
			return "ami-" + dest, nil
		}

		return aws
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	outputPath := os.Getenv("OUTPUT_STEMCELL_PATH")
	var dummyStemcellPath string

	dummyManifest := &bytes.Buffer{}
	_, _ = dummyManifest.WriteString("---\n")
	_, _ = dummyManifest.WriteString("name: bosh-aws-xen-ubuntu-trusty-go_agent\n")
	_, _ = dummyManifest.WriteString("cloud_properties:\n")
	_, _ = dummyManifest.WriteString("  name: bosh-aws-xen-ubuntu-trusty-go_agent")

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
		_, err = dummyManifestFile.Write(dummyManifest.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = dummyManifestFile.Close()
		Expect(err).ToNot(HaveOccurred())

		dummyStemcellPath = path.Join(dummyStemcellFolder, "dummy-xen-stemcell.tgz")
		tarCmd = exec.Command("tar", "-C", dummyStemcellFolder, "-czf", dummyStemcellPath, "--", "image", "apply_spec.yml", "stemcell.MF", "stemcell_dpkg_l.txt")
		err = tarCmd.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(dummyStemcellPath).To(BeAnExistingFile())

		Expect(outputPath).ToNot(BeEmpty())
		Expect(outputPath).To(BeADirectory())
	})

	Describe("Prepare", func() {
		It("prepares the work dir, with contents from the 'heavy' stemcell", func() {
			b := builder.New(logger, dummyAWS, awsConfig, ec2ami.Config{})

			imagePath, err := b.Prepare(dummyStemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))

			Expect(imagePath).To(BeAnExistingFile())
		})
	})

	Describe("PackageLightStemcell", func() {
		It("produces a light stemcell tarball", func() {
			b := builder.New(logger, dummyAWS, awsConfig, ec2ami.Config{})
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
			b := builder.New(logger, dummyAWS, awsConfig, ec2ami.Config{})
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
			b := builder.New(logger, dummyAWS, awsConfig, amiConfig)
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
				b := builder.New(logger, dummyAWS, awsConfig, amiConfig)

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
				b := builder.New(logger, dummyAWS, awsConfig, amiConfig)

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
			b := builder.New(logger, dummyAWS, awsConfig, amiConfig)
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
				b := builder.New(logger, dummyAWS, awsConfig, amiConfig)
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
				b := builder.New(logger, dummyAWS, awsConfig, amiConfig)
				lightStemcellPath := b.LightStemcellFilePath(heavyStemcellPath, outputPath)
				Expect(lightStemcellPath).To(Equal("/path/to/stemcell/light-some-xen-stemcell.tgz"))
			})
		})
	})

	Describe("BuildAmis", func() {
		var err error

		origAmiConfig := ec2ami.Config{
			Region:             "dest-0",
			Description:        "Dummy AMI",
			VirtualizationType: "hvm",
		}

		copyAmiConfig := ec2ami.Config{
			Region:             "dest-0",
			Description:        "Dummy AMI",
			VirtualizationType: "hvm",
			AmiID:              "ami-dest-0",
		}

		imagePath := "path/to/image"

		expectAWSIntegration := func(aws *fakes.FakeAWS, callback func()) {
			Expect(aws.ImportVolumeCallCount()).To(Equal(1))
			importImagePath := aws.ImportVolumeArgsForCall(0)
			Expect(importImagePath).To(Equal(imagePath))

			Expect(aws.DeleteDiskImageCallCount()).To(Equal(1))
			cleanupTaskID := aws.DeleteDiskImageArgsForCall(0)
			Expect(cleanupTaskID).To(Equal("task-id"))

			Expect(aws.CreateSnapshotCallCount()).To(Equal(1))
			volume := aws.CreateSnapshotArgsForCall(0)
			Expect(volume).To(Equal("volume-id"))

			Expect(aws.RegisterImageCallCount()).To(Equal(1))
			registerConfig, registerSnapshot := aws.RegisterImageArgsForCall(0)
			Expect(registerConfig).To(Equal(origAmiConfig))
			Expect(registerSnapshot).To(Equal("snapshot-id"))

			Expect(aws.DeleteVolumeCallCount()).To(Equal(1))
			deleteVolumeID := aws.DeleteVolumeArgsForCall(0)
			Expect(deleteVolumeID).To(Equal("volume-id"))

			callback()
		}

		BeforeEach(func() {
			err = origAmiConfig.Validate()
			Expect(err).ToNot(HaveOccurred())

			err = copyAmiConfig.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("integrates with AWS", func() {
			aws := makeStubbedAWS()
			b := builder.New(logger, aws, awsConfig, origAmiConfig)
			copyDests := []string{"dest-1", "dest-2"}

			regionToAmi, err := b.BuildAmis(imagePath, copyDests)
			Expect(err).ToNot(HaveOccurred())

			expectAWSIntegration(aws, func() {
				Expect(aws.CopyImageCallCount()).To(Equal(2))
				copy1Config, copy1Dest := aws.CopyImageArgsForCall(0)
				copy2Config, copy2Dest := aws.CopyImageArgsForCall(1)
				copyImageDestinations := []string{copy1Dest, copy2Dest}
				Expect(copy1Config).To(Equal(copyAmiConfig))
				Expect(copy2Config).To(Equal(copyAmiConfig))
				Expect(copyImageDestinations).To(ConsistOf(copyDests))

				Expect(regionToAmi).To(HaveKey("dest-0"))
				Expect(regionToAmi).To(HaveKey("dest-1"))
				Expect(regionToAmi).To(HaveKey("dest-2"))
				Expect(regionToAmi["dest-0"].AmiID).To(Equal("ami-dest-0"))
				Expect(regionToAmi["dest-1"].AmiID).To(Equal("ami-dest-1"))
				Expect(regionToAmi["dest-2"].AmiID).To(Equal("ami-dest-2"))
			})
		})

		Context("when no target copy destinations are provided", func() {
			It("works as expected", func() {
				aws := makeStubbedAWS()
				b := builder.New(logger, aws, awsConfig, origAmiConfig)
				copyDests := []string{}

				regionToAmi, err := b.BuildAmis(imagePath, copyDests)
				Expect(err).ToNot(HaveOccurred())

				expectAWSIntegration(aws, func() {
					Expect(aws.CopyImageCallCount()).To(Equal(0))

					Expect(regionToAmi).To(HaveKey("dest-0"))
					Expect(regionToAmi["dest-0"].AmiID).To(Equal("ami-dest-0"))
				})
			})
		})
	})

	Describe("Importing a machine image into AWS", func() {
		amiConfig := ec2ami.Config{
			Description:        "BOSH Stemcell Builder Test AMI",
			Public:             false,
			VirtualizationType: "hvm",
			Region:             awsConfig.Region,
		}
		copyDests := []string{"us-west-1", "us-west-2"}
		aws := makeStubbedAWS()
		b := builder.New(logger, aws, awsConfig, amiConfig)

		It("publishes AMIs and builds a light stemcell", func() {
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
