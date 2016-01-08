package config_test

import (
	"bytes"
	"encoding/json"
	"light-stemcell-builder/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type configModifier func(*config.Config)

func identityModifier(c *config.Config) { return }

func parseConfig(s string, modify configModifier) (config.Config, error) {
	configJSON := []byte(s)
	configReader := bytes.NewBuffer(configJSON)
	c, err := config.NewFromReader(configReader)
	Expect(err).ToNot(HaveOccurred())

	modify(&c)
	modifiedBytes, err := json.Marshal(c)
	if err != nil {
		return config.Config{}, err
	}

	modifiedConfigReader := bytes.NewBuffer(modifiedBytes)
	return config.NewFromReader(modifiedConfigReader)
}

var _ = Describe("Config", func() {
	baseJSON := `
    {
      "ami_configuration": {
        "description": "Example AMI"
      },
      "regions": [
        {
          "name": "us-region",
          "bucket_name": "test-bucket",
          "credentials": {
            "access_key": "access-key",
            "secret_key": "secret-key"
          }
        }
      ]
    }
  `

	Describe("NewFromReader", func() {
		It("returns a Config, with visibility and virtulization_type defaulted", func() {
			c, err := parseConfig(baseJSON, identityModifier)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.AmiConfiguration.VirtualizationType).To(Equal(config.HardwareAssistedVirtualization))
			Expect(c.AmiConfiguration.Visibility).To(Equal(config.PublicVisibility))
		})

		Context("with an invalid 'ami_configuration' specified", func() {
			It("returns an error when 'description' is missing", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiConfiguration.Description = ""
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("ami_configuration requires a description"))
			})

			It("returns an error when 'virtualization_type' is not 'hvm' or 'pv'", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiConfiguration.VirtualizationType = "bogus"
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("virtualization_type must be one of: ['hvm', 'pv']"))
			})

			It("returns an error when 'visibility' is not valid", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiConfiguration.Visibility = "bogus"
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("visibility must be one of: ['public', 'private']"))
			})
		})

		Context("with an empty 'regions' specified", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.Regions = []config.RegionConfiguration{}
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("regions cannot be empty"))
			})
		})

		Context("given a 'region' config without 'name'", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.Regions[0].Name = ""
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("region must specify name"))
			})
		})

		Context("given a 'region' config with invalid 'credentials'", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.Regions[0].Credentials.AccessKey = ""
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("credentials must specify access_key"))

				_, err = parseConfig(baseJSON, func(c *config.Config) {
					c.Regions[0].Credentials.SecretKey = ""
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("credentials must specify secret_key"))
			})
		})

		Context("given a 'region' config without 'bucket_name'", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.Regions[0].BucketName = ""
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("region must specify bucket_name"))
			})
		})

		Context("when China is involved", func() {
			It("returns an error if a China region is specified in copy destinations", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.Regions[0].Destinations = append(c.Regions[0].Destinations, "cn-north-1")
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("cn-north-1 is an isolated region and cannot be specified as a copy destination"))
			})

			It("returns an error if copy destinations are specified for a China region", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.Regions[0].Name = "cn-north-1"
					c.Regions[0].Destinations = append(c.Regions[0].Destinations, "anything")
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("cn-north-1 is an isolated region and cannot specify copy destinations"))
			})
		})
	})
})
