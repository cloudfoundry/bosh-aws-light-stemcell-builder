package resources

//counterfeiter:generate . MachineImageDriver
type MachineImageDriver interface {
	Create(MachineImageDriverConfig) (MachineImage, error)
	Delete(MachineImage) error
}

type MachineImage struct {
	GetURL     string
	DeleteURLs []string
}

type MachineImageDriverConfig struct {
	MachineImagePath     string
	BucketName           string
	ServerSideEncryption string
	FileFormat           string
	VolumeSizeGB         int64
}
