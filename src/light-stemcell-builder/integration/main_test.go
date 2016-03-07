package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/config"
	"light-stemcell-builder/manifest"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {

	var configPath string
	var manifestPath string
	var machineImagePath string
	var expectedRegions []string

	BeforeEach(func() {

		// US Region
		usAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		Expect(usAccessKey).ToNot(BeEmpty(), "AWS_ACCESS_KEY_ID must be set")

		usSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		Expect(usSecretKey).ToNot(BeEmpty(), "AWS_SECRET_ACCESS_KEY must be set")

		usRegion := os.Getenv("AWS_REGION")
		Expect(usRegion).ToNot(BeEmpty(), "US_AMI_REGION must be set")

		usDestination := os.Getenv("AWS_DESTINATION_REGION")
		Expect(usDestination).ToNot(BeEmpty(), "AWS_DESTINATION_REGION must be set")

		usBucket := os.Getenv("AWS_BUCKET_NAME")
		Expect(usBucket).ToNot(BeEmpty(), "US_AMI_BUCKET_NAME must be set")

		usDestinations := []string{usDestination}

		machineImagePath = os.Getenv("MACHINE_IMAGE_PATH")
		Expect(machineImagePath).ToNot(BeEmpty(), "MACHINE_IMAGE_PATH must be set")

		// China Region
		cnAccessKey := os.Getenv("AWS_CN_ACCESS_KEY_ID")
		Expect(cnAccessKey).ToNot(BeEmpty(), "AWS_CN_ACCESS_KEY_ID must be set")

		cnSecretKey := os.Getenv("AWS_CN_SECRET_ACCESS_KEY")
		Expect(cnSecretKey).ToNot(BeEmpty(), "AWS_CN_SECRET_ACCESS_KEY must be set")

		cnRegion := os.Getenv("AWS_CN_REGION")
		Expect(cnRegion).ToNot(BeEmpty(), "AWS_CN_REGION must be set")

		cnBucket := os.Getenv("AWS_CN_BUCKET_NAME")
		Expect(cnBucket).ToNot(BeEmpty(), "AWS_CN_BUCKET_NAME must be set")

		c := config.Config{
			AmiConfiguration: config.AmiConfiguration{
				Description:        "Integration Test AMI",
				VirtualizationType: "hvm",
				Visibility:         "private",
			},
			AmiRegions: []config.AmiRegion{
				config.AmiRegion{
					RegionName: usRegion,
					Credentials: config.Credentials{
						AccessKey: usAccessKey,
						SecretKey: usSecretKey,
					},
					BucketName:   usBucket,
					Destinations: usDestinations,
				},
				config.AmiRegion{
					RegionName: cnRegion,
					Credentials: config.Credentials{
						AccessKey: cnAccessKey,
						SecretKey: cnSecretKey,
					},
					BucketName: cnBucket,
				},
			},
		}

		expectedRegions = append(usDestinations, usRegion, cnRegion)

		integrationConfig, err := json.Marshal(c)
		Expect(err).ToNot(HaveOccurred())

		configFile, err := ioutil.TempFile("", "integration-config.json")
		Expect(err).ToNot(HaveOccurred())
		defer configFile.Close()

		_, err = configFile.Write(integrationConfig)
		Expect(err).ToNot(HaveOccurred())

		configPath = configFile.Name()

		rawManifest := `
name: bosh-aws-xen-ubuntu-trusty-go_agent
version: 9999
bosh_protocol: 1
sha1: 123456789
operating_system: ubuntu-trusty
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

		manifestFile, err := ioutil.TempFile("", "stemcell.MF")
		Expect(err).ToNot(HaveOccurred())
		defer manifestFile.Close()

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

		command := exec.Command(pathToBinary,
			fmt.Sprintf("-c=%s", configPath),
			fmt.Sprintf("--image=%s", machineImagePath),
			fmt.Sprintf("--manifest=%s", manifestPath),
		)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		session.Wait(30 * time.Minute)
		Expect(session.ExitCode()).To(BeZero())

		stdout := bytes.NewReader(session.Out.Contents())
		m, err := manifest.NewFromReader(stdout)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Name).To(Equal("bosh-aws-xen-hvm-ubuntu-trusty-go_agent"))
		Expect(m.Version).To(Equal("9999"))
		Expect(m.BoshProtocol).To(Equal("1"))
		Expect(m.Sha1).To(Equal("123456789"))
		Expect(m.OperatingSystem).To(Equal("ubuntu-trusty"))

		amis := m.CloudProperties.Amis
		Expect(amis).To(HaveLen(len(expectedRegions)))

		for _, region := range expectedRegions {
			Expect(amis).To(HaveKey(region))
			Expect(amis[region]).ToNot(BeEmpty())
		}
	})
})
