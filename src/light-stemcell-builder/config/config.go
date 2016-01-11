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
	// Credentials...
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
		return errors.New("ami_configuration requires a description")
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
		return errors.New("regions cannot be empty")
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
		return errors.New("region must specify name")
	}

	if r.BucketName == "" {
		return errors.New("region must specify bucket_name")
	}

	if r.Credentials.AccessKey == "" {
		return errors.New("credentials must specify access_key")
	}

	if r.Credentials.SecretKey == "" {
		return errors.New("credentials must specify secret_key")
	}

	for _, region := range r.Destinations {
		if isolated[region] {
			return fmt.Errorf("%s is an isolated region and cannot be specified as a copy destination", region)
		}
	}

	if isolated[r.Name] && len(r.Destinations) != 0 {
		return fmt.Errorf("%s is an isolated region and cannot specify copy destinations", r.Name)
	}

	return nil
}
