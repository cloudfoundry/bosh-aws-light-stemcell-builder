package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"light-stemcell-builder/manifest"
)

var _ = Describe("Main", func() {
	var configPath string
	var manifestPath string

	BeforeEach(func() {
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

			awsSession, err := session.NewSession(awsConfig)
			Expect(err).ToNot(HaveOccurred())
			ec2Client := ec2.New(awsSession)

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
				GinkgoWriter.Printf("Encountered error deregistering image %s in %s: %s", amiID, region, err)
			}
			_, err = ec2Client.DeleteSnapshot(&ec2.DeleteSnapshotInput{SnapshotId: snapshotID})
			if err != nil {
				GinkgoWriter.Printf("Encountered error deleting snapshot %s in %s: %s", *snapshotID, region, err)
			}
		}
	})
})
