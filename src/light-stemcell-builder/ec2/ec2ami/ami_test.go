package ec2ami_test

import (
	"light-stemcell-builder/ec2/ec2ami"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ami", func() {
	Describe("Configuration Validation", func() {
		It("checks that required fields have been set", func() {
			var c ec2ami.Config
			var err error

			c = ec2ami.Config{}

			err = c.Validate()
			Expect(err).To(MatchError("Region is required"))

			c = ec2ami.Config{
				Region: "some-region",
			}

			err = c.Validate()
			Expect(err).To(MatchError("Description is required"))

			c = ec2ami.Config{
				Region:      "some-region",
				Description: "some-description",
			}

			err = c.Validate()
			Expect(err).To(MatchError("VirtualizationType is required"))

			c = ec2ami.Config{
				Region:             "some-region",
				Description:        "some-description",
				VirtualizationType: "some-virtualization-type",
			}

			err = c.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
