package publisher

import (
	"fmt"
	"light-stemcell-builder/collection"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"sync"
)

//TODO: consolidate with config passed at CLI
type Config struct {
	MachineImagePath string
	BucketName       string
	AmiProperties    resources.AmiProperties
	CopyDestinations []string
}

func NewStandardRegionPublisher(c Config) *StandardRegionPublisher {
	return &StandardRegionPublisher{
		c: c,
	}
}

type StandardRegionPublisher struct {
	c Config
}

func (p *StandardRegionPublisher) Publish(ds driverset.StandardRegionDriverSet) (*collection.Ami, error) {
	machineImageDriverConfig := resources.MachineImageDriverConfig{
		MachineImagePath: p.c.MachineImagePath,
		BucketName:       p.c.BucketName,
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
		AmiProperties: p.c.AmiProperties,
	}

	sourceAmi, err := createAmiDriver.Create(createAmiDriverConfig)
	if err != nil {
		return nil, fmt.Errorf("creating ami: %s", err)
	}

	amis := collection.Ami{}
	amis.Add(sourceAmi)

	copyAmiDriver := ds.CopyAmiDriver()

	procGroup := sync.WaitGroup{}
	procGroup.Add(len(p.c.CopyDestinations))

	errCol := collection.Error{}

	for i := range p.c.CopyDestinations {
		go func(dstRegion string) {
			defer procGroup.Done()

			copyAmiDriverConfig := resources.AmiDriverConfig{
				ExistingAmiID:     sourceAmi.ID,
				DestinationRegion: dstRegion,
				AmiProperties:     p.c.AmiProperties,
			}

			copiedAmi, copyErr := copyAmiDriver.Create(copyAmiDriverConfig)
			if copyErr != nil {
				errCol.Add(fmt.Errorf("copying source ami: %s to destination region: %s: %s", sourceAmi.ID, dstRegion, copyErr))
				return
			}

			amis.Add(copiedAmi)
		}(p.c.CopyDestinations[i])
	}

	procGroup.Wait()

	// TODO: cleanup machine images

	return &amis, errCol.Error()
}
