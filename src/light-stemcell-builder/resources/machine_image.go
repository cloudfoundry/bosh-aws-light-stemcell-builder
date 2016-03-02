package resources

//go:generate counterfeiter -o fakes/fake_machine_image_driver.go . MachineImageDriver
type MachineImageDriver interface {
	Create(MachineImageDriverConfig) (MachineImage, error)
}

type MachineImage struct {
	GetURL    string
	DeleteURL string
}

type MachineImageDriverConfig struct {
	MachineImagePath string
	BucketName       string
}
