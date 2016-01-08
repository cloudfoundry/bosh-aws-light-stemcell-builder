package builder_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/builder"
	"light-stemcell-builder/config"
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

// TODO: dry this up with the one in ami_publisher_test.
func makeStubbedAWS() *fakes.FakeAWS {
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

var _ = Describe("StemcellBuilder", func() {
	var dummyStemcellPath string
	dummyAWS := &fakes.FakeAWS{}
	dummyConfig := config.Config{}
	logger := log.New(os.Stdout, "", log.LstdFlags)
	outputPath := os.Getenv("OUTPUT_STEMCELL_PATH")

	dummyManifest := &bytes.Buffer{}
	_, _ = dummyManifest.WriteString("---\n")
	_, _ = dummyManifest.WriteString("name: bosh-aws-xen-ubuntu-trusty-go_agent\n")
	_, _ = dummyManifest.WriteString("cloud_properties:\n")
	_, _ = dummyManifest.WriteString("  name: bosh-aws-xen-ubuntu-trusty-go_agent")

	BeforeSuite(func() {
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
			b := builder.New(dummyAWS, dummyConfig, logger)

			imagePath, err := b.Prepare(dummyStemcellPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(imagePath).To(ContainSubstring("/root.img"))
			Expect(imagePath).To(BeAnExistingFile())
		})
	})

	Describe("Package", func() {
		It("produces a light stemcell tarball", func() {
			b := builder.New(dummyAWS, dummyConfig, logger)
			packageDir, err := ioutil.TempDir("", "light-stemcell-builder-package-test")
			Expect(err).ToNot(HaveOccurred())

			_, err = b.Prepare(dummyStemcellPath)
			Expect(err).ToNot(HaveOccurred())

			outputPackage := path.Join(packageDir, "package.tgz")
			err = b.Package(outputPackage)
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
			b := builder.New(dummyAWS, dummyConfig, logger)
			packageDir, err := ioutil.TempDir("", "light-stemcell-builder-package-test")
			Expect(err).ToNot(HaveOccurred())

			outputPackage := path.Join(packageDir, "package.tgz")
			err = b.Package(outputPackage)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Please call Prepare() before Package"))
		})
	})

	Describe("UpdateManifestFile", func() {
		builderConfig := config.Config{
			AmiConfiguration: config.AmiConfiguration{
				VirtualizationType: "hvm",
			},
		}

		It("correctly updates the manifest file", func() {
			b := builder.New(dummyAWS, builderConfig, logger)
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
			builderConfig := config.Config{
				AmiConfiguration: config.AmiConfiguration{
					VirtualizationType: "hvm",
				},
			}

			It("outputs the correct manifest", func() {
				b := builder.New(dummyAWS, builderConfig, logger)
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
			builderConfig := config.Config{
				AmiConfiguration: config.AmiConfiguration{
					VirtualizationType: "non-hvm",
				},
			}

			It("outputs the correct manifest", func() {
				b := builder.New(dummyAWS, builderConfig, logger)

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
			b := builder.New(dummyAWS, dummyConfig, logger)
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

	Describe("OutputPath", func() {
		heavyStemcellPath := "some-xen-stemcell.tgz"
		outputPath := "/path/to/stemcell/"

		Context("given a HVM stemcell", func() {
			builderConfig := config.Config{
				AmiConfiguration: config.AmiConfiguration{
					VirtualizationType: "hvm",
				},
			}

			It("returns the expected file path", func() {
				b := builder.New(dummyAWS, builderConfig, logger)
				lightStemcellPath := b.OutputPath(heavyStemcellPath, outputPath)
				Expect(lightStemcellPath).To(Equal("/path/to/stemcell/light-some-xen-hvm-stemcell.tgz"))
			})
		})

		Context("given a non-HVM stemcell", func() {
			builderConfig := config.Config{
				AmiConfiguration: config.AmiConfiguration{
					VirtualizationType: "non-hvm",
				},
			}

			It("returns the expected file path", func() {
				b := builder.New(dummyAWS, builderConfig, logger)
				lightStemcellPath := b.OutputPath(heavyStemcellPath, outputPath)
				Expect(lightStemcellPath).To(Equal("/path/to/stemcell/light-some-xen-stemcell.tgz"))
			})
		})
	})

	Describe("Build", func() {
		aws := makeStubbedAWS()
		logger := log.New(os.Stdout, "", log.LstdFlags)

		Context("given a single region", func() {
			configJSON := []byte(`
		    {
		      "ami_configuration": {
		        "description": "Example AMI",
						"visibility": "private"
		      },
		      "regions": [
		        {
		          "name": "region-1",
		          "bucket_name": "bucket-name",
		          "credentials": {
		            "access_key": "access-key",
		            "secret_key": "secret-key"
		          },
							"destinations": ["destination-1", "destination-2"]
		        }
		      ]
		    }
		  `)
			configReader := bytes.NewBuffer(configJSON)

			It("publishes AMIs and builds a light stemcell", func() {
				c, err := config.NewFromReader(configReader)
				Expect(err).ToNot(HaveOccurred())

				b := builder.New(aws, c, logger)
				stemcell, amis, err := b.Build(dummyStemcellPath, outputPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(amis).To(HaveKey("region-1"))
				Expect(amis).To(HaveKey("destination-1"))
				Expect(amis).To(HaveKey("destination-2"))
				Expect(amis["region-1"].AmiID).To(MatchRegexp("ami-.*"))
				Expect(amis["destination-1"].AmiID).To(MatchRegexp("ami-.*"))
				Expect(amis["destination-2"].AmiID).To(MatchRegexp("ami-.*"))

				Expect(stemcell).To(BeAnExistingFile())
			})
		})

		Context("given a standard and an isolated region", func() {
			configJSON := []byte(fmt.Sprintf(`
		    {
		      "ami_configuration": {
		        "description": "Example AMI",
						"visibility": "private"
		      },
		      "regions": [
		        {
		          "name": "region-1",
		          "bucket_name": "bucket-name",
		          "credentials": {
		            "access_key": "access-key",
		            "secret_key": "secret-key"
		          },
							"destinations": ["destination-1", "destination-2"]
		        },
		        {
		          "name": "%s",
		          "bucket_name": "bucket-name",
		          "credentials": {
		            "access_key": "access-key",
		            "secret_key": "secret-key"
		          }
		        }
		      ]
		    }
		  `, config.IsolatedChinaRegion))
			configReader := bytes.NewBuffer(configJSON)

			It("publishes AMIs and builds a light stemcell", func() {
				c, err := config.NewFromReader(configReader)
				Expect(err).ToNot(HaveOccurred())

				b := builder.New(aws, c, logger)
				stemcell, amis, err := b.Build(dummyStemcellPath, outputPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(amis).To(HaveKey("region-1"))
				Expect(amis).To(HaveKey("destination-1"))
				Expect(amis).To(HaveKey("destination-2"))
				Expect(amis["region-1"].AmiID).To(MatchRegexp("ami-.*"))
				Expect(amis["destination-1"].AmiID).To(MatchRegexp("ami-.*"))
				Expect(amis["destination-2"].AmiID).To(MatchRegexp("ami-.*"))

				Expect(amis).To(HaveKey(config.IsolatedChinaRegion))
				Expect(amis[config.IsolatedChinaRegion].AmiID).To(MatchRegexp("ami-.*"))

				Expect(stemcell).To(BeAnExistingFile())
			})
		})
	})
})
