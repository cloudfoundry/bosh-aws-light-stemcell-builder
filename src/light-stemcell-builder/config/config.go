package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

const (
	PublicVisibility  = "public"
	PrivateVisibility = "private"
)

const (
	HardwareAssistedVirtualization = "hvm"
	Paravirtualization             = "paravirtual"
)

const (
	IsolatedChinaRegion = "cn-north-1"
)

var isolated = map[string]bool{IsolatedChinaRegion: true}

// Convention:
// 1. required
// 2. optional, defaulted
// 3. optional
type AmiConfiguration struct {
	Description        string `json:"description"`
	VirtualizationType string `json:"virtualization_type"`
	Visibility         string `json:"visibility"`
}

type AmiRegion struct {
	Name         string      `json:"name"`
	Credentials  Credentials `json:"credentials"`
	BucketName   string      `json:"bucket_name"`
	Destinations []string    `json:"destinations"`
}

type Credentials struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type Config struct {
	AmiConfiguration AmiConfiguration `json:"ami_configuration"`
	AmiRegions       []AmiRegion      `json:"ami_regions"`
}

func NewFromReader(r io.Reader) (Config, error) {
	c := Config{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, err
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		return Config{}, err
	}

	if c.AmiConfiguration.VirtualizationType == "" {
		c.AmiConfiguration.VirtualizationType = HardwareAssistedVirtualization
	}

	if c.AmiConfiguration.Visibility == "" {
		c.AmiConfiguration.Visibility = PublicVisibility
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
		Paravirtualization:             true,
	}
	if !validVirtualization[config.AmiConfiguration.VirtualizationType] {
		return errors.New("virtualization_type must be one of: ['hvm', 'paravirtual']")
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
	if r.Name == "" {
		return errors.New("name must be specified for ami_regions entries")
	}

	if r.BucketName == "" {
		return errors.New("bucket_name must be specified for ami_regions entries")
	}

	if r.Credentials.AccessKey == "" {
		return errors.New("access_key must be specified for credentials")
	}

	if r.Credentials.SecretKey == "" {
		return errors.New("secret_key must be specified for credentials")
	}

	for _, destinationRegion := range r.Destinations {
		if isolated[destinationRegion] {
			return fmt.Errorf("%s is an isolated region and cannot be specified as a copy destination", destinationRegion)
		}

		if r.Name == destinationRegion {
			return fmt.Errorf("%s specified as both a source and a copy destination", destinationRegion)
		}
	}

	if isolated[r.Name] && len(r.Destinations) != 0 {
		return fmt.Errorf("%s is an isolated region and cannot specify copy destinations", r.Name)
	}

	return nil
}
