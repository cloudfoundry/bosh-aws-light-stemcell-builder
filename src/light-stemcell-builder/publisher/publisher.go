package publisher

import "light-stemcell-builder/config"

type Config struct {
	config.AmiRegion
	config.AmiConfiguration
}

type MachineImageConfig struct {
	LocalPath  string
	FileFormat string
}
