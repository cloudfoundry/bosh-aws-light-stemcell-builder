package resources

// AMI creation constants
const (
	PublicAmiAccessibility  = "public"
	PrivateAmiAccessibility = "private"
	AmiArchitecture         = "x86_64"
	HvmAmiVirtualization    = "hvm"
)

// AmiDriver abstracts the API calls required to build an AMI
//go:generate counterfeiter -o fakes/fake_ami_driver.go . AmiDriver
type AmiDriver interface {
	Create(AmiDriverConfig) (Ami, error)
}

// Ami represents an AMI resource in EC2
type Ami struct {
	ID                 string
	Region             string
	VirtualizationType string
}

// AmiProperties describes what properties the published AMI should have
type AmiProperties struct {
	Accessibility      string
	Description        string
	Name               string
	VirtualizationType string
	Encrypted          bool
	KmsKeyId           string
}

// AmiDriverConfig allows an AmiDriver to create an AMI from either a snapshot ID or an existing AMI (copy)
type AmiDriverConfig struct {
	SnapshotID        string
	ExistingAmiID     string
	DestinationRegion string
	AmiProperties
}
