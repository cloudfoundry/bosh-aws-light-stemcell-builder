package publisher

import (
	"fmt"
	"light-stemcell-builder/collection"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"sync"
)

func NewStandardRegionPublisher(c Config) *StandardRegionPublisher {
	return &StandardRegionPublisher{
		Region:           c.RegionName,
		BucketName:       c.BucketName,
		CopyDestinations: c.Destinations,
		AmiProperties: resources.AmiProperties{
			Name:               c.AmiName,
			Description:        c.Description,
			Accessibility:      c.Visibility,
			VirtualizationType: c.VirtualizationType,
		},
	}
}

type StandardRegionPublisher struct {
	Region           string
	BucketName       string
	AmiProperties    resources.AmiProperties
	CopyDestinations []string
}

type Config struct {
	config.AmiRegion
	config.AmiConfiguration
}

func (p *StandardRegionPublisher) Publish(ds driverset.StandardRegionDriverSet, machineImagePath string) (*collection.Ami, error) {
	machineImageDriverConfig := resources.MachineImageDriverConfig{
		MachineImagePath: machineImagePath,
		BucketName:       p.BucketName,
	}

	machineImageDriver := ds.CreateMachineImageDriver()
	machineImage, err := machineImageDriver.Create(machineImageDriverConfig)

	if err != nil {
		return nil, fmt.Errorf("creating machine image: %s", err)
	}

	snapshotDriverConfig := resources.SnapshotDriverConfig{
		MachineImageURL: machineImage.GetURL,
	}

	snapshotDriver := ds.CreateSnapshotDriver()
	snapshot, err := snapshotDriver.Create(snapshotDriverConfig)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot: %s", err)
	}

	createAmiDriver := ds.CreateAmiDriver()
	createAmiDriverConfig := resources.AmiDriverConfig{
		SnapshotID:    snapshot.ID,
		AmiProperties: p.AmiProperties,
	}

	sourceAmi, err := createAmiDriver.Create(createAmiDriverConfig)
	if err != nil {
		return nil, fmt.Errorf("creating ami: %s", err)
	}

	amis := collection.Ami{}
	amis.Add(sourceAmi)

	copyAmiDriver := ds.CopyAmiDriver()

	procGroup := sync.WaitGroup{}
	procGroup.Add(len(p.CopyDestinations))

	errCol := collection.Error{}

	for i := range p.CopyDestinations {
		go func(dstRegion string) {
			defer procGroup.Done()

			copyAmiDriverConfig := resources.AmiDriverConfig{
				ExistingAmiID:     sourceAmi.ID,
				DestinationRegion: dstRegion,
				AmiProperties:     p.AmiProperties,
			}

			copiedAmi, copyErr := copyAmiDriver.Create(copyAmiDriverConfig)
			if copyErr != nil {
				errCol.Add(fmt.Errorf("copying source ami: %s to destination region: %s: %s", sourceAmi.ID, dstRegion, copyErr))
				return
			}

			amis.Add(copiedAmi)
		}(p.CopyDestinations[i])
	}

	procGroup.Wait()

	// TODO: cleanup machine images

	return &amis, errCol.Error()
}
