package resources

// Volume properties which we do not expect to change
const (
	VolumeRawFormat    = "RAW"
	VolumeVMDKFormat   = "vmdk"
	VolumeArchitecture = "x86_64"
)

//counterfeiter:generate . VolumeDriver
type VolumeDriver interface {
	Create(VolumeDriverConfig) (Volume, error)
	Delete(Volume) error
}

type Volume struct {
	ID string
}

type VolumeDriverConfig struct {
	MachineImageManifestURL string
}
