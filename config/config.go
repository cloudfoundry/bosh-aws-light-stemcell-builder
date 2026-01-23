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
	AmiName            string `json:"name"`
	Description        string `json:"description"`
	VirtualizationType string `json:"virtualization_type"`
	Efi                bool   `json:"efi"`

	// Encrypted has to be set to true if encrypted stemcells should be created.
	// If set to true, then the EBS key, that is assigned to the AWS account, is used for the encryption by default.
	Encrypted bool `json:"encrypted"`

	// KmsKeyId can be used to provide a KMS key that should be used for the stemcell encryption.
	//
	// The KmsKeyId can be the:
	//   - ARN of a custom multi region KMS key,
	//   - ARN of a custom single region KMS key,
	//   - ID of the AWS managed EBS key.
	//
	// To produce an encrypted stemcell that can be shared accross regions one has to provide the ARN of a multi region KMS key.
	KmsKeyId string `json:"kms_key_id"`

	// KmsKeyAliasName can be used to provide an alias name for the custom KMS key.
	// The alias name defaults to 'light-stemcell-builder' if a KmsKeyAliasName is not provided.
	KmsKeyAliasName string `json:"kms_key_alias_name"`

	// Visibility enables the creation of either a public or a private stemcell.
	// The Visibility can be 'public' or 'private' but it defaults to public.
	Visibility string `json:"visibility"`

	// Tags that should be set on the created light stemcell.
	Tags map[string]string `json:"tags,omitempty"`

	// SharedWithAccounts allows to provide a list of AWS account IDs.
	// Private stemcells are then shared with these account IDs.
	SharedWithAccounts []string `json:"shared_with_accounts"`
}

type AmiRegion struct {
	// RegionName allows to configures the region where a stemcell should be produced.
	RegionName string `json:"name"`

	// Credentials allows to configure the access credentials for the configured RegionName.
	Credentials Credentials `json:"credentials"`

	// BucketName provides the name of the bucket where created machine images are stored.
	BucketName string `json:"bucket_name"`

	ServerSideEncryption string `json:"server_side_encryption"`

	// Destinations allows to configure multiple regions where produced stemcells should be copied to.
	Destinations []string `json:"destinations"`

	IsolatedRegion bool `json:"-"`
}

type Credentials struct {
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	SessionToken string `json:"session_token"`
	RoleArn      string `json:"role_arn"`
	Region       string `json:"-"`
}

type Config struct {
	// AmiConfiguration allows to configure some basic properties like description, encryption or visibility of the light stemcell
	// that should be produced.
	AmiConfiguration AmiConfiguration `json:"ami_configuration"`

	// AmiRegion allows to configure region specific properties.
	// For example the region where a light stemcell should be produced or where it should be copied to.
	AmiRegions []AmiRegion `json:"ami_regions"`
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
			return Config{}, fmt.Errorf("Unable to generate amiName: %s", err.Error()) //nolint:staticcheck
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
	var creds *credentials.Credentials

	switch {
	case configCredentials.AccessKey != "" && configCredentials.SecretKey != "":
		// Static or temporary credentials
		creds = credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     configCredentials.AccessKey,
			SecretAccessKey: configCredentials.SecretKey,
			SessionToken:    configCredentials.SessionToken,
		})
	case configCredentials.RoleArn != "":
		// EC2 role credentials, will assume RoleArn
		creds = credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.Must(session.NewSession())),
		})
	default:
		// EC2 role credentials, no role assumption
		creds = credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.Must(session.NewSession())),
		})
	}

	awsCfg := aws.NewConfig().WithRegion(configCredentials.Region).WithCredentials(creds)

	// If RoleArn is set and we have base credentials, assume the role
	if configCredentials.RoleArn != "" && (configCredentials.AccessKey != "" || configCredentials.SecretKey != "") {
		awsCfg.Credentials = stscreds.NewCredentials(
			session.Must(session.NewSession(awsCfg)),
			configCredentials.RoleArn,
		)
	} else if configCredentials.RoleArn != "" && configCredentials.AccessKey == "" && configCredentials.SecretKey == "" {
		awsCfg.Credentials = stscreds.NewCredentials(
			session.Must(session.NewSession(awsCfg)),
			configCredentials.RoleArn,
		)
	}

	return awsCfg
}