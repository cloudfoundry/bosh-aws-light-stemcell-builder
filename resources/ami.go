package resources

// You only need **one** of these per package!
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// AMI creation constants
const (
	PublicAmiAccessibility  = "public"
	PrivateAmiAccessibility = "private"
	AmiArchitecture         = "x86_64"
	HvmAmiVirtualization    = "hvm"
)

// AmiDriver abstracts the API calls required to build an AMI
//
//counterfeiter:generate . AmiDriver
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
	Efi                bool
	Encrypted          bool
	KmsKeyId           string
	KmsKeyAliasName    string
	KmsKeyAlias        string
	Tags               map[string]string
	SharedWithAccounts []string
}

// AmiDriverConfig allows an AmiDriver to create an AMI from either a snapshot ID or an existing AMI (copy)
type AmiDriverConfig struct {
	SnapshotID        string
	ExistingAmiID     string
	DestinationRegion string
	AmiProperties
	KmsKey
}
