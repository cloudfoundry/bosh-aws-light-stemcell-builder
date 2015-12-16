package ec2ami

import (
	"errors"
	"fmt"
	"light-stemcell-builder/uuid"
	"sync"
)

const (
	AmiArchitecture         = "x86_64"
	AmiPublicAccessibility  = "public"
	AmiPrivateAccessibility = "private"
	AmiAvailableStatus      = "available"
	AmiUnknownStatus        = "unknown" // we don't actually know whether the AMI was deregistered or never existed
)

type NonAvailableAmiError struct {
	AmiID     string
	AmiStatus string
}

func (e NonAvailableAmiError) Error() string {
	return fmt.Sprintf("AMI with id: %s is not available due to status: %s", e.AmiID, e.AmiStatus)
}

type Config struct {
	Description        string
	Public             bool
	VirtualizationType string
	UniqueName         string
	Region             string
	AmiID              string
}

type Info struct {
	InputConfig        Config `json:"-"`
	AmiID              string `json:"ami_id"`
	Region             string `json:"region"`
	SnapshotID         string `json:"snapshot_id"`
	Accessibility      string `json:"accessibility"`
	Name               string `json:"name"`
	ImageStatus        string `json:"-"`
	KernelId           string `json:"-"`
	Architecture       string `json:"-"`
	VirtualizationType string `json:"virtualization_type"`
	StorageType        string `json:"-"`
}

func (i Info) Status() string {
	return i.ImageStatus
}

func (c *Config) ID() string {
	return c.AmiID
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
	if c.Region == "" {
		return errors.New("Region is required")
	}

	if c.Description == "" {
		return errors.New("Description is required")
	}

	if c.VirtualizationType == "" {
		return errors.New("VirtualizationType is required")
	}

	return nil
}

func NewCollection() *Collection {
	return &Collection{amis: make(map[string]Info)}
}

type Collection struct {
	sync.Mutex
	amis map[string]Info
}

func (a *Collection) Add(region string, amiInfo Info) {
	a.Lock()
	defer a.Unlock()

	a.amis[region] = amiInfo
}

func (a *Collection) Get(region string) Info {
	a.Lock()
	defer a.Unlock()

	return a.amis[region]
}

func (a *Collection) GetAll() map[string]Info {
	a.Lock()
	defer a.Unlock()

	return a.amis
}
