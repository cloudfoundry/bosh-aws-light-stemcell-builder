package config_test

import (
	"bytes"
	"encoding/json"

	"light-stemcell-builder/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type configModifier func(*config.Config)

func identityModifier(_ *config.Config) {}

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
      "ami_regions": [
        {
          "name": "ami-region",
          "bucket_name": "ami-bucket",
          "credentials": {
            "access_key": "access-key",
            "secret_key": "secret-key"
          }
        }
      ]
    }
  `

	Describe("NewFromReader", func() {
		It("returns a Config, with ami name, visibility, and virtulization_type defaulted", func() {
			c, err := parseConfig(baseJSON, identityModifier)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.AmiConfiguration.AmiName).To(MatchRegexp("BOSH-.+"))
			Expect(c.AmiConfiguration.VirtualizationType).To(Equal(config.HardwareAssistedVirtualization))
			Expect(c.AmiConfiguration.Visibility).To(Equal(config.PublicVisibility))
		})

		It("sets the name if provided", func() {
			c, err := parseConfig(baseJSON, func(c *config.Config) {
				c.AmiConfiguration.AmiName = "fake-name"
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(c.AmiConfiguration.AmiName).To(Equal("fake-name"))
		})

		Context("with an invalid 'ami_configuration' specified", func() {
			It("returns an error when 'description' is missing", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiConfiguration.Description = ""
				})
				Expect(err).To(MatchError("description must be specified for ami_configuration"))
			})

			It("returns an error when 'virtualization_type' is not 'hvm'", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiConfiguration.VirtualizationType = "bogus"
				})
				Expect(err).To(MatchError("virtualization_type must be one of: ['hvm']"))
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
					c.AmiRegions = []config.AmiRegion{}
				})
				Expect(err).To(MatchError("ami_regions must be specified"))
			})
		})

		Context("given a 'region' config without 'name'", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiRegions[0].RegionName = ""
				})
				Expect(err).To(MatchError("name must be specified for ami_regions entries"))
			})
		})

		Context("when a 'region' config specifies itself as one of the copy destinations", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiRegions[0].RegionName = "us-east-1"
					c.AmiRegions[0].Destinations = append(c.AmiRegions[0].Destinations, "us-east-1")
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("us-east-1 specified as both a source and a copy destination"))
			})
		})

		Context("given a 'region' config without 'bucket_name'", func() {
			It("returns an error", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiRegions[0].BucketName = ""
				})
				Expect(err).To(MatchError("bucket_name must be specified for ami_regions entries"))
			})
		})

		Context("when given a standard region", func() {
			It("sets IsolatedRegion to false", func() {
				standardRegions := []string{"us-east-1", "eu-central-1", "ap-northeast-1"}
				for _, region := range standardRegions {
					c, err := parseConfig(baseJSON, func(c *config.Config) {
						c.AmiRegions[0].RegionName = region
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(c.AmiRegions[0].IsolatedRegion).To(BeFalse())
				}
			})
		})

		Context("when given an isolated region", func() {
			It("sets IsolatedRegion to true", func() {
				isolatedRegions := []string{"cn-north-1"}
				for _, region := range isolatedRegions {
					c, err := parseConfig(baseJSON, func(c *config.Config) {
						c.AmiRegions[0].RegionName = region
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(c.AmiRegions[0].IsolatedRegion).To(BeTrue())
				}
			})

			It("returns an error if an isolated region is specified in copy destinations", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiRegions[0].Destinations = append(c.AmiRegions[0].Destinations, "cn-north-1")
				})
				Expect(err).To(MatchError("cn-north-1 is an isolated region and cannot be specified as a copy destination"))
			})

			It("returns an error if copy destinations are specified for an isolated region", func() {
				_, err := parseConfig(baseJSON, func(c *config.Config) {
					c.AmiRegions[0].RegionName = "cn-north-1"
					c.AmiRegions[0].Destinations = append(c.AmiRegions[0].Destinations, "anything")
				})
				Expect(err).To(MatchError("cn-north-1 is an isolated region and cannot specify copy destinations"))
			})
		})
	})

	Describe("GetAwsConfig", func() {
		var keyID = "test-key-id"
		var keyValue = "test-key-value"
		var token = "test-token"
		var region = "us-east-1"
		var roleArn = "arn:aws:iam::123456789012:role/TestRole"

		Context("when both key fields are provided", func() {
			It("returns static credentials with correct values", func() {
				creds := config.Credentials{
					AccessKey: keyID,
					SecretKey: keyValue,
					Region:    region,
				}

				awsCfg := creds.GetAwsConfig()

				Expect(*awsCfg.Region).To(Equal(region))

				v, err := awsCfg.Credentials.Get()
				Expect(err).NotTo(HaveOccurred())
				Expect(v.AccessKeyID).To(Equal(keyID))
				Expect(v.SecretAccessKey).To(Equal(keyValue))
				Expect(v.SessionToken).To(BeEmpty())
				Expect(v.ProviderName).To(Equal("StaticProvider"))
			})
		})

		Context("when session token is also provided", func() {
			It("includes the token in static credentials", func() {
				creds := config.Credentials{
					AccessKey:    keyID,
					SecretKey:    keyValue,
					SessionToken: token,
					Region:       region,
				}

				awsCfg := creds.GetAwsConfig()

				v, err := awsCfg.Credentials.Get()
				Expect(err).NotTo(HaveOccurred())
				Expect(v.AccessKeyID).To(Equal(keyID))
				Expect(v.SecretAccessKey).To(Equal(keyValue))
				Expect(v.SessionToken).To(Equal(token))
				Expect(v.ProviderName).To(Equal("StaticProvider"))
			})
		})

		Context("when no key fields are provided", func() {
			It("does not use static credentials", func() {
				creds := config.Credentials{
					Region: region,
				}

				awsCfg := creds.GetAwsConfig()

				Expect(*awsCfg.Region).To(Equal(region))
				Expect(awsCfg.Credentials).NotTo(BeNil())
				Expect(awsCfg.Credentials.IsExpired()).To(BeTrue())

				v, err := awsCfg.Credentials.Get()
				if err == nil {
					Expect(v.ProviderName).NotTo(Equal("StaticProvider"))
				}
			})
		})

		Context("when only access key is provided", func() {
			It("does not use static credentials", func() {
				creds := config.Credentials{
					AccessKey: keyID,
					Region:    region,
				}

				awsCfg := creds.GetAwsConfig()

				Expect(awsCfg.Credentials).NotTo(BeNil())
				Expect(awsCfg.Credentials.IsExpired()).To(BeTrue())

				v, err := awsCfg.Credentials.Get()
				if err == nil {
					Expect(v.ProviderName).NotTo(Equal("StaticProvider"))
				}
			})
		})

		Context("when only secret key is provided", func() {
			It("does not use static credentials", func() {
				creds := config.Credentials{
					SecretKey: keyValue,
					Region:    region,
				}

				awsCfg := creds.GetAwsConfig()

				Expect(awsCfg.Credentials).NotTo(BeNil())
				Expect(awsCfg.Credentials.IsExpired()).To(BeTrue())

				v, err := awsCfg.Credentials.Get()
				if err == nil {
					Expect(v.ProviderName).NotTo(Equal("StaticProvider"))
				}
			})
		})

		Context("when role ARN is provided with both key fields", func() {
			It("does not use static credentials directly", func() {
				creds := config.Credentials{
					AccessKey: keyID,
					SecretKey: keyValue,
					RoleArn:   roleArn,
					Region:    region,
				}

				awsCfg := creds.GetAwsConfig()

				Expect(*awsCfg.Region).To(Equal(region))
				Expect(awsCfg.Credentials.IsExpired()).To(BeTrue())

				// STS wraps the static creds, so provider is no longer StaticProvider
				v, err := awsCfg.Credentials.Get()
				if err == nil {
					Expect(v.ProviderName).NotTo(Equal("StaticProvider"))
				}
			})
		})

		Context("when role ARN is provided without key fields", func() {
			It("does not use static credentials", func() {
				creds := config.Credentials{
					RoleArn: roleArn,
					Region:  region,
				}

				awsCfg := creds.GetAwsConfig()

				Expect(*awsCfg.Region).To(Equal(region))
				Expect(awsCfg.Credentials.IsExpired()).To(BeTrue())

				v, err := awsCfg.Credentials.Get()
				if err == nil {
					Expect(v.ProviderName).NotTo(Equal("StaticProvider"))
				}
			})
		})

		Context("when region is set", func() {
			It("always propagates region to the config", func() {
				creds := config.Credentials{
					Region: "eu-west-1",
				}

				awsCfg := creds.GetAwsConfig()

				Expect(*awsCfg.Region).To(Equal("eu-west-1"))
			})
		})
	})
})
