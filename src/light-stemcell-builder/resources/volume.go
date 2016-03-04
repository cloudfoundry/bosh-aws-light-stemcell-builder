package resources

//go:generate counterfeiter -o fakes/fake_volume_driver.go . VolumeDriver
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
