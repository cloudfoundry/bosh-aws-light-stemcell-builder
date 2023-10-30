package publisher

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"light-stemcell-builder/collection"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"
)

type StandardRegionPublisher struct {
	Region               string
	BucketName           string
	ServerSideEncryption string
	AmiProperties        resources.AmiProperties
	CopyDestinations     []string
	logger               *log.Logger
}

func NewStandardRegionPublisher(logDest io.Writer, c Config) *StandardRegionPublisher {
	return &StandardRegionPublisher{
		Region:               c.RegionName,
		BucketName:           c.BucketName,
		ServerSideEncryption: c.ServerSideEncryption,
		CopyDestinations:     c.Destinations,
		AmiProperties: resources.AmiProperties{
			Name:               c.AmiName,
			Description:        c.Description,
			Accessibility:      c.Visibility,
			VirtualizationType: c.VirtualizationType,
			Encrypted:          c.Encrypted,
			KmsKeyId:           c.KmsKeyId,
			Tags:               c.Tags,
		},
		logger: log.New(logDest, "StandardRegionPublisher ", log.LstdFlags),
	}
}

func (p *StandardRegionPublisher) Publish(ds driverset.StandardRegionDriverSet, machineImageConfig MachineImageConfig) (*collection.Ami, error) {

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		p.logger.Printf("completed Publish() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	machineImageDriverConfig := resources.MachineImageDriverConfig{
		MachineImagePath:     machineImageConfig.LocalPath,
		FileFormat:           machineImageConfig.FileFormat,
		BucketName:           p.BucketName,
		ServerSideEncryption: p.ServerSideEncryption,
	}

	machineImageDriver := ds.MachineImageDriver()
	machineImage, err := machineImageDriver.Create(machineImageDriverConfig)
	if err != nil {
		return nil, fmt.Errorf("creating machine image: %s", err)
	}
	defer func() {
		err := machineImageDriver.Delete(machineImage)
		if err != nil {
			p.logger.Printf("Failed to delete machine image %s: %s", machineImage.GetURL, err)
		}
	}()

	snapshotDriverConfig := resources.SnapshotDriverConfig{
		MachineImageURL: machineImage.GetURL,
		FileFormat:      machineImageConfig.FileFormat,
		AmiProperties:   p.AmiProperties,
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

	return &amis, errCol.Error()
}
