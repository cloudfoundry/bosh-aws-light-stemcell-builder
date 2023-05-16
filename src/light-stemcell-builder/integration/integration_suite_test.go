package integration_test

import (
	"os"
	"testing"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var cfg config.Config

var machineImagePath string
var machineImageFormat string
var machineImageSize string

var expectedRegions []string

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedBeforeSuite(
	func() []byte {
		cfg = config.Config{
			AmiConfiguration: config.AmiConfiguration{
				Description:        "Integration Test AMI",
				VirtualizationType: "hvm",
				Visibility:         "private",
			},
		}

		cfg.AmiRegions = append(cfg.AmiRegions, constructUsAmiRegion())

		cnAmiRegion := constructCnAmiRegion()
		if cnAmiRegion.RegionName != "" {
			cfg.AmiRegions = append(cfg.AmiRegions, cnAmiRegion)
		}

		for _, amiRegion := range cfg.AmiRegions {
			expectedRegions = append(expectedRegions, amiRegion.RegionName)
			expectedRegions = append(expectedRegions, amiRegion.Destinations...)
		}

		machineImagePath = os.Getenv("MACHINE_IMAGE_PATH")
		Expect(machineImagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		machineImageFormat = os.Getenv("MACHINE_IMAGE_FORMAT")
		Expect(machineImageFormat).ToNot(BeEmpty(), "MACHINE_IMAGE_FORMAT must be set")

		machineImageSize = os.Getenv("MACHINE_IMAGE_VOLUME_SIZE")
		if machineImageFormat != resources.VolumeRawFormat {
			Expect(machineImageSize).ToNot(BeEmpty(), "MACHINE_IMAGE_VOLUME_SIZE must be set")
		}

		return []byte{}
	},
	func(data []byte) {},
)

func constructUsAmiRegion() config.AmiRegion {
	usAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	Expect(usAccessKey).ToNot(BeEmpty(), "AWS_ACCESS_KEY_ID must be set")
	usSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	Expect(usSecretKey).ToNot(BeEmpty(), "AWS_SECRET_ACCESS_KEY must be set")

	usRegion := os.Getenv("AWS_REGION")
	Expect(usRegion).ToNot(BeEmpty(), "AWS_REGION must be set")

	usDestination := os.Getenv("AWS_DESTINATION_REGION")
	Expect(usDestination).ToNot(BeEmpty(), "AWS_DESTINATION_REGION must be set")

	usBucket := os.Getenv("AWS_BUCKET_NAME")
	Expect(usBucket).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")

	return config.AmiRegion{
		RegionName: usRegion,
		BucketName: usBucket,
		Credentials: config.Credentials{
			AccessKey: usAccessKey,
			SecretKey: usSecretKey,
		},
		Destinations: []string{usDestination},
	}
}

func constructCnAmiRegion() config.AmiRegion {
	cnRegion := os.Getenv("AWS_CN_REGION")
	cnBucket := os.Getenv("AWS_CN_BUCKET_NAME")

	if cnRegion != "" && cnBucket != "" {
		cnAccessKey := os.Getenv("AWS_CN_ACCESS_KEY_ID")
		Expect(cnAccessKey).NotTo(BeEmpty(), "AWS_CN_ACCESS_KEY_ID if AWS_CN_REGION and AWS_CN_BUCKET_NAME are present")
		cnSecretKey := os.Getenv("AWS_CN_SECRET_ACCESS_KEY")
		Expect(cnSecretKey).NotTo(BeEmpty(), "AWS_CN_SECRET_ACCESS_KEY if AWS_CN_REGION and AWS_CN_BUCKET_NAME are present")

		return config.AmiRegion{
			RegionName: cnRegion,
			BucketName: cnBucket,
			Credentials: config.Credentials{
				AccessKey: cnAccessKey,
				SecretKey: cnSecretKey,
			},
		}
	}

	return config.AmiRegion{}
}
