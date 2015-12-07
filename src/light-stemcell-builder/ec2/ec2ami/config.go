package ec2ami

import (
	"errors"
	"light-stemcell-builder/uuid"
)

const (
	AmiArchitecture         = "x86_64"
	AmiPublicAccessibility  = "public"
	AmiPrivateAccessibility = "private"
)

type Config struct {
	Description        string
	Public             bool
	VirtualizationType string
	UniqueName         string
}

func (c *Config) Name() (string, error) {
	if c.UniqueName != "" {
		return c.UniqueName, nil
	}

	var err error
	c.UniqueName, err = uuid.New("BOSH")
	if err != nil {
		return "", err
	}

	return c.UniqueName, nil
}

func (c *Config) Validate() error {
	if c.Description == "" {
		return errors.New("Description is required")
	}

	if c.VirtualizationType == "" {
		return errors.New("VirtualizationType is required")
	}

	return nil
}
