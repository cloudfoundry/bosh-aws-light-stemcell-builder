package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/manifest"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {

	var cfg config.Config
	var configPath string
	var manifestPath string
	var machineImagePath string
	var machineImageFormat string
	var machineImageSize string
	var expectedRegions []string

	BeforeEach(func() {

		// US Region
		usAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		usSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

		usRegion := os.Getenv("AWS_REGION")
		Expect(usRegion).ToNot(BeEmpty(), "AWS_REGION must be set")

		usDestination := os.Getenv("AWS_DESTINATION_REGION")
		Expect(usDestination).ToNot(BeEmpty(), "AWS_DESTINATION_REGION must be set")

		usBucket := os.Getenv("AWS_BUCKET_NAME")
		Expect(usBucket).ToNot(BeEmpty(), "AWS_BUCKET_NAME must be set")

		usDestinations := []string{usDestination}

		machineImagePath = os.Getenv("MACHINE_IMAGE_PATH")
		Expect(machineImagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		machineImageFormat = os.Getenv("MACHINE_IMAGE_FORMAT")
		Expect(machineImageFormat).ToNot(BeEmpty(), "MACHINE_IMAGE_FORMAT must be set")

		machineImageSize = os.Getenv("MACHINE_IMAGE_VOLUME_SIZE")
		if machineImageFormat != resources.VolumeRawFormat {
			Expect(machineImageSize).ToNot(BeEmpty(), "MACHINE_IMAGE_VOLUME_SIZE must be set")
		}

		// China Region
		cnAccessKey := os.Getenv("AWS_CN_ACCESS_KEY_ID")
		cnSecretKey := os.Getenv("AWS_CN_SECRET_ACCESS_KEY")
		cnRegion := os.Getenv("AWS_CN_REGION")
		cnBucket := os.Getenv("AWS_CN_BUCKET_NAME")

		cfg = config.Config{
			AmiConfiguration: config.AmiConfiguration{
				Description:        "Integration Test AMI",
				VirtualizationType: "hvm",
				Visibility:         "private",
			},
			AmiRegions: []config.AmiRegion{
				{
					RegionName: usRegion,
					Credentials: config.Credentials{
						AccessKey: usAccessKey,
						SecretKey: usSecretKey,
					},
					BucketName:   usBucket,
					Destinations: usDestinations,
				},
			},
		}

		expectedRegions = append(usDestinations, usRegion)

		if cnRegion != "" && cnBucket != "" {
			cfg.AmiRegions = append(cfg.AmiRegions, config.AmiRegion{
				RegionName: cnRegion,
				Credentials: config.Credentials{
					AccessKey: cnAccessKey,
					SecretKey: cnSecretKey,
				},
				BucketName: cnBucket,
			})

			expectedRegions = append(expectedRegions, cnRegion)
		}

		integrationConfig, err := json.Marshal(cfg)
		Expect(err).ToNot(HaveOccurred())

		configFile, err := os.CreateTemp("", "integration-config.json")
		Expect(err).ToNot(HaveOccurred())
		defer configFile.Close() //nolint:errcheck

		_, err = configFile.Write(integrationConfig)
		Expect(err).ToNot(HaveOccurred())

		configPath = configFile.Name()

		rawManifest := `
name: bosh-aws-xen-ubuntu-trusty-go_agent
version: 9999
bosh_protocol: 1
sha1: 123456789
operating_system: ubuntu-trusty
stemcell_formats:
- aws-raw
cloud_properties:
  name: bosh-aws-xen-ubuntu-trusty-go_agent
  version: blah
  infrastructure: aws
  hypervisor: xen
  disk: 3072
  disk_format: raw
  container_format: bare
  os_type: linux
  os_distro: ubuntu
  architecture: x86_64
  root_device_name: /dev/sda1
`

		manifestFile, err := os.CreateTemp("", "stemcell.MF")
		Expect(err).ToNot(HaveOccurred())
		defer manifestFile.Close() //nolint:errcheck

		_, err = manifestFile.Write([]byte(rawManifest))
		Expect(err).ToNot(HaveOccurred())

		manifestPath = manifestFile.Name()
	})

	AfterEach(func() {
		_ = os.RemoveAll(configPath)
		_ = os.RemoveAll(manifestPath)
	})

	It("publishes to the configured regions and outputs to stdout", func() {
		pathToBinary, err := gexec.Build("light-stemcell-builder")
		defer gexec.CleanupBuildArtifacts()
		Expect(err).ToNot(HaveOccurred())

		args := []string{fmt.Sprintf("-c=%s", configPath),
			fmt.Sprintf("--image=%s", machineImagePath),
			fmt.Sprintf("--manifest=%s", manifestPath),
		}
		if machineImageFormat != "RAW" {
			args = append(args,
				fmt.Sprintf("--format=%s", machineImageFormat),
				fmt.Sprintf("--volume-size=%s", machineImageSize),
			)
		}
		command := exec.Command(pathToBinary, args...)

		gexecSession, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		gexecSession.Wait(60 * time.Minute)
		Expect(gexecSession.ExitCode()).To(BeZero())

		stdout := bytes.NewReader(gexecSession.Out.Contents())
		m, err := manifest.NewFromReader(stdout)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Name).To(Equal("bosh-aws-xen-hvm-ubuntu-trusty-go_agent"))
		Expect(m.Version).To(Equal("9999"))
		Expect(m.BoshProtocol).To(Equal("1"))
		Expect(m.Sha1).To(Equal("da39a3ee5e6b4b0d3255bfef95601890afd80709"))
		Expect(m.OperatingSystem).To(Equal("ubuntu-trusty"))
		Expect(m.StemcellFormats).To(HaveLen(1))
		Expect(m.StemcellFormats).To(ContainElement("aws-light"))

		amis := m.CloudProperties.Amis
		Expect(amis).To(HaveLen(len(expectedRegions)))

		for _, region := range expectedRegions {
			Expect(amis).To(HaveKey(region))
			Expect(amis[region]).ToNot(BeEmpty())
		}

		for region, amiID := range amis {
			var awsConfig *aws.Config
			if region == "cn-north-1" {
				cnCreds := credentials.NewStaticCredentials(cfg.AmiRegions[1].Credentials.AccessKey, cfg.AmiRegions[1].Credentials.SecretKey, "")
				awsConfig = aws.NewConfig().
					WithCredentials(cnCreds).
					WithRegion(region)
			} else {
				usCreds := credentials.NewStaticCredentials(cfg.AmiRegions[0].Credentials.AccessKey, cfg.AmiRegions[0].Credentials.SecretKey, "")
				awsConfig = aws.NewConfig().
					WithCredentials(usCreds).
					WithRegion(region)
			}

			newSession, err := session.NewSession()
			Expect(err).ToNot(HaveOccurred())
			ec2Client := ec2.New(newSession, awsConfig)

			reqOutput, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{aws.String(amiID)}})
			Expect(err).ToNot(HaveOccurred())

			Expect(reqOutput.Images).To(HaveLen(1))
			Expect(reqOutput.Images[0].Tags).To(HaveLen(4))
			for _, tag := range reqOutput.Images[0].Tags {
				if tag.Key == aws.String("name") {
					Expect(tag.Value).To(Equal(aws.String("ubuntu-trusty-9999")))
				}
			}
			snapshotID := reqOutput.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId
			Expect(snapshotID).ToNot(BeNil())
			Expect(aws.BoolValue(reqOutput.Images[0].EnaSupport)).To(BeTrue())

			_, err = ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: aws.String(amiID)})
			if err != nil {
				GinkgoWriter.Write([]byte(fmt.Sprintf("Encountered error deregistering image %s in %s: %s", amiID, region, err))) //nolint:errcheck
			}
			_, err = ec2Client.DeleteSnapshot(&ec2.DeleteSnapshotInput{SnapshotId: snapshotID})
			if err != nil {
				GinkgoWriter.Write([]byte(fmt.Sprintf("Encountered error deleting snapshot %s in %s: %s", *snapshotID, region, err))) //nolint:errcheck
			}
		}
	})
})
