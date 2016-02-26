package manifest_test

import (
	"bytes"
	"fmt"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/manifest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Context("reading and writing the manifest", func() {
		manifestBytes := []byte(`name: bosh-aws-xen-ubuntu-trusty-go_agent
version: blah
bosh_protocol: 1
sha1: some-sha
operating_system: ubuntu-trusty
cloud_properties:
  name: {}
  version: blah
  infrastructure: aws
  hypervisor: xen
  disk: 3072
  disk_format: raw
  container_format: bare
  os_type: linux
  os_distro: ubuntu
  architecture: x86_64
  root_device_name: /dev/sda1`)

		It("correctly preserves the YAML file", func() {
			manifestReader := bytes.NewReader(manifestBytes)
			manifestStruct, err := manifest.NewFromReader(manifestReader)
			Expect(err).ToNot(HaveOccurred())

			outputManifest := &bytes.Buffer{}
			err = manifestStruct.ToYAML(outputManifest)
			Expect(err).ToNot(HaveOccurred())
			Expect(outputManifest.String()).To(ContainSubstring(string(manifestBytes)))
			Expect(outputManifest.String()).ToNot(MatchRegexp("(?m)^  ami:.*$"))
			Expect(outputManifest.String()).ToNot(MatchRegexp("(?m)^    .*$"))
		})

		It("correctly adds the region to AMI map to the YAML file", func() {
			manifestReader := bytes.NewReader(manifestBytes)
			manifestStruct, err := manifest.NewFromReader(manifestReader)
			Expect(err).ToNot(HaveOccurred())

			amiCollection := ec2ami.NewCollection()
			amiCollection.Add("us-east-1", ec2ami.Info{AmiID: "ami-us-east-1"})
			manifestStruct.AddAMICollection(amiCollection)

			outputManifest := &bytes.Buffer{}
			err = manifestStruct.ToYAML(outputManifest)
			Expect(err).ToNot(HaveOccurred())
			Expect(outputManifest.String()).To(ContainSubstring(string(manifestBytes)))
			Expect(outputManifest.String()).To(MatchRegexp("(?m)^  ami:$"))
			Expect(outputManifest.String()).To(MatchRegexp("(?m)^    us-east-1: ami-us-east-1$"))
		})
		Context("SetHVM", func() {
			It("correctly changes the stemcell name", func() {
				manifestReader := bytes.NewReader(manifestBytes)
				manifestStruct, err := manifest.NewFromReader(manifestReader)
				Expect(err).ToNot(HaveOccurred())

				manifestStruct.SetHVM()

				outputManifest := &bytes.Buffer{}
				err = manifestStruct.ToYAML(outputManifest)
				Expect(err).ToNot(HaveOccurred())
				Expect(outputManifest.String()).To(MatchRegexp("(?m)^name: bosh-aws-xen-hvm-ubuntu-trusty-go_agent$"))
				Expect(outputManifest.String()).To(MatchRegexp("(?m)^  name: bosh-aws-xen-hvm-ubuntu-trusty-go_agent$"))
				fmt.Printf("outputManifest: %s", outputManifest.String())
			})
		})
		Context("given an invalid manifest", func() {
			It("NewFromReader returns an error", func() {
				manifestReader := bytes.NewReader([]byte("key: key: value"))
				_, err := manifest.NewFromReader(manifestReader)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unmarshaling YAML to manifest: "))
			})
		})
	})
})
