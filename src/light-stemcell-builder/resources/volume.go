package resources

import "sync"

type VolumeDriver interface {
	Create(VolumeDriverConfig) (string, error)
}

type Volume struct {
	id           string
	driver       VolumeDriver
	driverConfig VolumeDriverConfig
	opErr        error
	once         *sync.Once
}

type VolumeDriverConfig struct {
	MachineImageManifestURL string
}

// WaitForCreation attempts to create a snapshot from a machine image returning the ID or error
func (v *Volume) WaitForCreation() (string, error) {
	v.once.Do(func() {
		v.id, v.opErr = v.driver.Create(v.driverConfig)
	})

	return v.id, v.opErr
}

// NewVolume serves as a volume factory, callers call WaitForCreation() to create a volume from a machine image in AWS
func NewVolume(driver VolumeDriver, driverConfig VolumeDriverConfig) Volume {
	return Volume{driver: driver, driverConfig: driverConfig, once: &sync.Once{}}
}
