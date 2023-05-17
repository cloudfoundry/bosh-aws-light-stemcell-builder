package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	uuid "github.com/satori/go.uuid"
)

const (
	PublicVisibility  = "public"
	PrivateVisibility = "private"
)

const (
	HardwareAssistedVirtualization = "hvm"
)

var isolated = map[string]bool{
	"cn-north-1": true,
}

// Convention:
// 1. required
// 2. optional, defaulted
// 3. optional
type AmiConfiguration struct {
	AmiName            string            `json:"name"`
	Description        string            `json:"description"`
	VirtualizationType string            `json:"virtualization_type"`
	Encrypted          bool              `json:"encrypted"`
	KmsKeyId           string            `json:"kms_key_id"`
	Visibility         string            `json:"visibility"`
	Tags               map[string]string `json:"tags,omitempty"`
}

type AmiRegion struct {
	RegionName           string      `json:"name"`
	Credentials          Credentials `json:"credentials"`
	BucketName           string      `json:"bucket_name"`
	ServerSideEncryption string      `json:"server_side_encryption"`
	Destinations         []string    `json:"destinations"`
	IsolatedRegion       bool        `json:"-"`
}

type Credentials struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	RoleArn   string `json:"role_arn"`
	Region    string `json:"-"`
}

type Config struct {
	AmiConfiguration AmiConfiguration `json:"ami_configuration"`
	AmiRegions       []AmiRegion      `json:"ami_regions"`
}

func NewFromReader(r io.Reader) (Config, error) {
	c := Config{}

	b, err := io.ReadAll(r)
	if err != nil {
		return Config{}, err
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		return Config{}, err
	}

	if c.AmiConfiguration.AmiName == "" {
		if err != nil {
			return Config{}, fmt.Errorf("Unable to generate amiName: %s", err.Error())
		}
		c.AmiConfiguration.AmiName = fmt.Sprintf("BOSH-%s", uuid.NewV4().String())
	}

	if c.AmiConfiguration.VirtualizationType == "" {
		c.AmiConfiguration.VirtualizationType = HardwareAssistedVirtualization
	}

	if c.AmiConfiguration.Visibility == "" {
		c.AmiConfiguration.Visibility = PublicVisibility
	}

	for i := range c.AmiRegions {
		region := &c.AmiRegions[i]
		region.Credentials.Region = region.RegionName
		region.IsolatedRegion = isolated[region.RegionName]
	}

	err = c.validate()
	if err != nil {
		return Config{}, err
	}

	return c, nil
}

func (config *Config) validate() error {
	if config.AmiConfiguration.Description == "" {
		return errors.New("description must be specified for ami_configuration")
	}

	validVirtualization := map[string]bool{
		HardwareAssistedVirtualization: true,
	}
	if !validVirtualization[config.AmiConfiguration.VirtualizationType] {
		return errors.New("virtualization_type must be one of: ['hvm']")
	}

	validVisibility := map[string]bool{
		PublicVisibility:  true,
		PrivateVisibility: true,
	}
	if !validVisibility[config.AmiConfiguration.Visibility] {
		return errors.New("visibility must be one of: ['public', 'private']")
	}

	regions := config.AmiRegions
	if len(regions) == 0 {
		return errors.New("ami_regions must be specified")
	}

	for i := range regions {
		err := regions[i].validate()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *AmiRegion) validate() error {
	if r.RegionName == "" {
		return errors.New("name must be specified for ami_regions entries")
	}

	if r.BucketName == "" {
		return errors.New("bucket_name must be specified for ami_regions entries")
	}

	if r.Credentials.Region == "" {
		return errors.New("region must be specified for credentials")
	}

	for _, destinationRegion := range r.Destinations {
		if isolated[destinationRegion] {
			return fmt.Errorf("%s is an isolated region and cannot be specified as a copy destination", destinationRegion)
		}

		if r.RegionName == destinationRegion {
			return fmt.Errorf("%s specified as both a source and a copy destination", destinationRegion)
		}
	}

	if isolated[r.RegionName] && len(r.Destinations) != 0 {
		return fmt.Errorf("%s is an isolated region and cannot specify copy destinations", r.RegionName)
	}

	return nil
}

func (configCredentials *Credentials) GetAwsConfig() *aws.Config {
	var awsCredentials *credentials.Credentials

	if configCredentials.AccessKey != "" && configCredentials.SecretKey != "" {
		awsCredentials = credentials.NewStaticCredentialsFromCreds(
			credentials.Value{AccessKeyID: configCredentials.AccessKey, SecretAccessKey: configCredentials.SecretKey},
		)

		if configCredentials.RoleArn != "" {
			staticConfig := aws.NewConfig().WithRegion(configCredentials.Region).WithCredentials(awsCredentials)
			awsCredentials = stscreds.NewCredentials(
				session.Must(session.NewSession(staticConfig)),
				configCredentials.RoleArn,
			)
		}
	} else {
		awsCredentials = credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.Must(session.NewSession())),
		})
	}

	return aws.NewConfig().WithRegion(configCredentials.Region).WithCredentials(awsCredentials)
}
