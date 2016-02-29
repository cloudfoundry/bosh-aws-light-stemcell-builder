package resources

import "sync"

// AMI creation constants
const (
	PublicAmiAccessibility  = "public"
	PrivateAmiAccessibility = "private"
	AmiArchitecture         = "x86_64"
	HvmAmiVirtualization    = "hvm"
	PvAmiVirtualization     = "paravirtual"
)

// AmiDriver abstracts the API calls required to build an AMI
type AmiDriver interface {
	Create(AmiDriverConfig) (string, error)
}

// Ami represents an AMI resource in EC2
type Ami struct {
	id           string
	driver       AmiDriver
	driverConfig AmiDriverConfig
	opErr        error
	once         *sync.Once
}

type AmiProperties struct {
	Accessibility      string
	Description        string
	Name               string
	VirtualizationType string
}

// AmiDriverConfig allows an AmiDriver to create an AMI from either a snapshot ID or an existing AMI (copy)
type AmiDriverConfig struct {
	SnapshotID        string
	ExistingAmiID     string
	DestinationRegion string
	AmiProperties
}

// WaitForCreation attempts to create an AMI from a snapshot or existng AMI returning the ID or error
func (a *Ami) WaitForCreation() (string, error) {
	a.once.Do(func() {
		a.id, a.opErr = a.driver.Create(a.driverConfig)
	})

	return a.id, a.opErr
}

// NewAmi serves as am AMI builder, callers call WaitForCreation() to create an AMI from a snapshot or existng AMI in AWS
func NewAmi(driver AmiDriver, driverConfig AmiDriverConfig) Ami {
	return Ami{driver: driver, driverConfig: driverConfig, once: &sync.Once{}}
}
