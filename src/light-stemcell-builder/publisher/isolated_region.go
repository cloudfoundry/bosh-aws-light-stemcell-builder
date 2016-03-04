package publisher

import (
	"fmt"
	"io"
	"light-stemcell-builder/collection"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
	"log"
	"time"
)

type IsolatedRegionPublisher struct {
	Region        string
	BucketName    string
	AmiProperties resources.AmiProperties
	logger        *log.Logger
}

func NewIsolatedRegionPublisher(logDest io.Writer, c Config) *IsolatedRegionPublisher {
	return &IsolatedRegionPublisher{
		Region:     c.RegionName,
		BucketName: c.BucketName,
		AmiProperties: resources.AmiProperties{
			Name:               c.AmiName,
			Description:        c.Description,
			Accessibility:      c.Visibility,
			VirtualizationType: c.VirtualizationType,
		},
		logger: log.New(logDest, "IsolatedRegionPublisher ", log.LstdFlags),
	}
}

func (p *IsolatedRegionPublisher) Publish(ds driverset.IsolatedRegionDriverSet, machineImagePath string) (*collection.Ami, error) {
	createStartTime := time.Now()
	defer func(startTime time.Time) {
		p.logger.Printf("completed Publish() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	machineImageDriverConfig := resources.MachineImageDriverConfig{
		MachineImagePath: machineImagePath,
		BucketName:       p.BucketName,
	}

	machineImageDriver := ds.CreateMachineImageDriver()
	machineImage, err := machineImageDriver.Create(machineImageDriverConfig)
	if err != nil {
		return nil, fmt.Errorf("creating machine image: %s", err)
	}

	volumeDriverConfig := resources.VolumeDriverConfig{
		MachineImageManifestURL: machineImage.GetURL,
	}

	volumeDriver := ds.CreateVolumeDriver()
	volume, err := volumeDriver.Create(volumeDriverConfig)
	if err != nil {
		return nil, fmt.Errorf("creating volume: %s", err)
	}

	snapshotDriverConfig := resources.SnapshotDriverConfig{
		VolumeID: volume.ID,
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

	amis := collection.Ami{
		VirtualizationType: p.AmiProperties.VirtualizationType,
	}
	amis.Add(sourceAmi)

	// TODO: cleanup machine images and volumes

	return &amis, nil
}
