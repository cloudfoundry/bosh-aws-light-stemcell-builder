package builder_test

import (
	"light-stemcell-builder/builder"
	"light-stemcell-builder/config"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/fakes"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AMIPublisher", func() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	Describe("Publish", func() {
		imagePath := "path/to/image"
		publisherAMIConfig := config.AmiConfiguration{
			Description:        "Example AMI",
			VirtualizationType: "hvm",
			Visibility:         "private",
		}
		inputAMIConfig := ec2ami.Config{
			Region:             "example-region",
			Description:        "Example AMI",
			VirtualizationType: "hvm",
		}
		outputAMIConfig := ec2ami.Config{
			Region:             "example-region",
			Description:        "Example AMI",
			VirtualizationType: "hvm",
			AmiID:              "ami-example-region",
		}

		assertIntegration := func(aws *fakes.FakeAWS, additionalAssertions func()) {
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
			Expect(registerConfig).To(Equal(inputAMIConfig))
			Expect(registerSnapshot).To(Equal("snapshot-id"))

			Expect(aws.DeleteVolumeCallCount()).To(Equal(1))
			deleteVolumeID := aws.DeleteVolumeArgsForCall(0)
			Expect(deleteVolumeID).To(Equal("volume-id"))

			additionalAssertions()
		}

		stubbedAWS := func() *fakes.FakeAWS {
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

		It("integrates with AWS", func() {
			aws := stubbedAWS()
			regionConfig := config.RegionConfiguration{
				Name:       "example-region",
				BucketName: "example-bucket",
				Credentials: config.AwsCredentials{
					AccessKey: "access-key",
					SecretKey: "secret-key",
				},
				Destinations: []string{"destination-1", "destination-2"},
			}
			p := builder.AMIPublisher{
				AWS:       aws,
				AMIConfig: publisherAMIConfig,
				Logger:    logger,
			}

			result, err := p.Publish(imagePath, regionConfig)
			Expect(err).ToNot(HaveOccurred())

			assertIntegration(aws, func() {
				Expect(aws.CopyImageCallCount()).To(Equal(2))
				copy1Config, destination1 := aws.CopyImageArgsForCall(0)
				copy2Config, destination2 := aws.CopyImageArgsForCall(1)
				destinations := []string{destination1, destination2}

				Expect(copy1Config).To(Equal(outputAMIConfig))
				Expect(copy2Config).To(Equal(outputAMIConfig))
				Expect(destinations).To(ConsistOf(regionConfig.Destinations))
			})

			Expect(len(result)).To(Equal(3))
			Expect(result).To(HaveKey("example-region"))
			Expect(result).To(HaveKey("destination-1"))
			Expect(result).To(HaveKey("destination-2"))
			Expect(result["example-region"].AmiID).To(Equal("ami-example-region"))
			Expect(result["destination-1"].AmiID).To(Equal("ami-destination-1"))
			Expect(result["destination-2"].AmiID).To(Equal("ami-destination-2"))
		})

		Context("when no destinations are provided", func() {
			It("works as expected", func() {
				aws := stubbedAWS()
				regionConfig := config.RegionConfiguration{
					Name:       "example-region",
					BucketName: "example-bucket",
					Credentials: config.AwsCredentials{
						AccessKey: "access-key",
						SecretKey: "secret-key",
					},
				}
				p := builder.AMIPublisher{
					AWS:       aws,
					AMIConfig: publisherAMIConfig,
					Logger:    logger,
				}

				result, err := p.Publish(imagePath, regionConfig)
				Expect(err).ToNot(HaveOccurred())

				assertIntegration(aws, func() {
					Expect(aws.CopyImageCallCount()).To(Equal(0))

					Expect(len(result)).To(Equal(1))
					Expect(result).To(HaveKey("example-region"))
					Expect(result["example-region"].AmiID).To(Equal("ami-example-region"))
				})
			})
		})
	})
})
