package publisher

import (
	"light-stemcell-builder/config"
)

type Config struct {
	config.AmiRegion
	config.AmiConfiguration //nolint:vet
}

type MachineImageConfig struct {
	LocalPath    string
	FileFormat   string
	VolumeSizeGB int64
}
