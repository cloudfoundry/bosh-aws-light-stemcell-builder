package resources

import "sync"

type MachineImageDriver interface {
	Create(MachineImageDriverConfig) (string, error)
}

type MachineImage struct {
	id           string
	driver       MachineImageDriver
	driverConfig MachineImageDriverConfig
	opErr        error
	once         *sync.Once
}

type MachineImageDriverConfig struct {
	MachineImagePath string
	BucketName       string
}

// WaitForCreation attempts to create a snapshot from a machine image returning the ID or error
func (i *MachineImage) WaitForCreation() (string, error) {
	i.once.Do(func() {
		i.id, i.opErr = i.driver.Create(i.driverConfig)
	})

	return i.id, i.opErr
}

// NewMachineImage serves as a machine image factory, callers call WaitForCreation() to
// create a signed s3 url from a machine image in AWS S3
func NewMachineImage(driver MachineImageDriver, driverConfig MachineImageDriverConfig) MachineImage {
	return MachineImage{driver: driver, driverConfig: driverConfig, once: &sync.Once{}}
}
