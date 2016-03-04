package publisher

import "light-stemcell-builder/config"

type Config struct {
	config.AmiRegion
	config.AmiConfiguration
}
