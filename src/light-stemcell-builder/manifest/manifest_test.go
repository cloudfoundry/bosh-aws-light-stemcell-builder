package manifest_test

import (
	"bytes"
	"light-stemcell-builder/manifest"
	"light-stemcell-builder/resources"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Context("reading and writing the manifest", func() {
		manifestBytes := []byte(`
name: bosh-aws-xen-ubuntu-trusty-go_agent
version: blah
bosh_protocol: 1
sha1: some-sha
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
  root_device_name: /dev/sda1`)

		It("writes the expected YAML file", func() {
			manifestReader := bytes.NewReader(manifestBytes)
			m, err := manifest.NewFromReader(manifestReader)
			Expect(err).ToNot(HaveOccurred())

			m.PublishedAmis = []resources.Ami{
				resources.Ami{
					Region:             "fake-region",
					ID:                 "fake-ami-id",
					VirtualizationType: resources.PvAmiVirtualization,
				},
			}

			writer := &bytes.Buffer{}
			err = m.Write(writer)
			Expect(err).ToNot(HaveOccurred())

			resultManifest := &manifest.Manifest{}
			err = yaml.Unmarshal(writer.Bytes(), resultManifest)
			Expect(err).ToNot(HaveOccurred())

			Expect(resultManifest.Name).To(Equal("bosh-aws-xen-ubuntu-trusty-go_agent"))
			Expect(resultManifest.Version).To(Equal("blah"))
			Expect(resultManifest.BoshProtocol).To(Equal("1"))
			Expect(resultManifest.Sha1).To(Equal("some-sha"))
			Expect(resultManifest.OperatingSystem).To(Equal("ubuntu-trusty"))
			Expect(resultManifest.CloudProperties.Amis).To(HaveLen(1))
			Expect(resultManifest.CloudProperties.Amis["fake-region"]).To(Equal("fake-ami-id"))
		})

		It("adds 'hvm' to name if AMI collection has HVM virtualization type", func() {
			manifestReader := bytes.NewReader(manifestBytes)
			m, err := manifest.NewFromReader(manifestReader)
			Expect(err).ToNot(HaveOccurred())

			m.PublishedAmis = []resources.Ami{
				resources.Ami{
					Region:             "fake-region",
					ID:                 "fake-ami-id",
					VirtualizationType: resources.HvmAmiVirtualization,
				},
			}

			writer := &bytes.Buffer{}
			err = m.Write(writer)
			Expect(err).ToNot(HaveOccurred())

			resultManifest := &manifest.Manifest{}
			err = yaml.Unmarshal(writer.Bytes(), resultManifest)
			Expect(err).ToNot(HaveOccurred())

			Expect(resultManifest.Name).To(Equal("bosh-aws-xen-hvm-ubuntu-trusty-go_agent"))
		})

		Context("given an invalid manifest", func() {
			It("NewFromReader returns an error", func() {
				manifestReader := bytes.NewReader([]byte("key: key: value"))
				_, err := manifest.NewFromReader(manifestReader)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unmarshaling YAML to manifest: "))
			})
		})

		It("returns an error if Amis is not set", func() {
			manifestReader := bytes.NewReader(manifestBytes)
			manifestStruct, err := manifest.NewFromReader(manifestReader)
			Expect(err).ToNot(HaveOccurred())

			outputManifest := &bytes.Buffer{}
			err = manifestStruct.Write(outputManifest)
			Expect(err).To(MatchError("no Amis have been added to the manifest"))
		})
	})
})
