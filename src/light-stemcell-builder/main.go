package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"light-stemcell-builder/collection"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/manifest"
	"light-stemcell-builder/publisher"
	"light-stemcell-builder/resources"
	"log"
	"os"
	"sync"
)

func usage(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, "Usage of light-stemcell-builder/main.go")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	sharedWriter := &logWriter{
		writer: os.Stderr,
	}

	logger := log.New(sharedWriter, "", log.LstdFlags)

	configPath := flag.String("c", "", "Path to the JSON configuration file")
	machineImagePath := flag.String("image", "", "Path to the input machine image (root.img)")
	machineImageFormat := flag.String("format", resources.VolumeRawFormat, "Format of the input machine image (RAW or vmdk). Defaults to RAW.")
	imageVolumeSize := flag.Int("volume-size", 0, "Block device size (in GB) of the input machine image")
	manifestPath := flag.String("manifest", "", "Path to the input stemcell.MF")

	flag.Parse()

	if *configPath == "" {
		usage("-c flag is required")
	}
	if *machineImagePath == "" {
		usage("--image flag is required")
	}

	if *manifestPath == "" {
		usage("--manifest flag is required")
	}

	if *imageVolumeSize == 0 && *machineImageFormat != resources.VolumeRawFormat {
		usage("--volume-size flag is required for formats other than RAW")
	}

	configFile, err := os.Open(*configPath)
	if err != nil {
		logger.Fatalf("Error opening config file: %s", err)
	}

	defer func() {
		closeErr := configFile.Close()
		if closeErr != nil {
			logger.Fatalf("Error closing config file: %s", closeErr)
		}
	}()

	if err != nil {
		logger.Fatalf("Error opening config file: %s", err)
	}

	c, err := config.NewFromReader(configFile)
	if err != nil {
		logger.Fatalf("Error parsing config file: %s. Message: %s", *configPath, err)
	}

	if _, err := os.Stat(*machineImagePath); os.IsNotExist(err) {
		logger.Fatalf("machine image not found at: %s", *machineImagePath)
	}

	if _, err := os.Stat(*manifestPath); os.IsNotExist(err) {
		logger.Fatalf("manifest not found at: %s", *manifestPath)
	}

	manifestBytes, err := ioutil.ReadFile(*manifestPath)
	if err != nil {
		logger.Fatalf("opening manifest: %s", err)
	}

	m, err := manifest.NewFromReader(bytes.NewReader(manifestBytes))
	if err != nil {
		logger.Fatalf("reading manifest: %s", err)
	}

	amiCollection := collection.Ami{}
	errCollection := collection.Error{}

	var wg sync.WaitGroup
	wg.Add(len(c.AmiRegions))

	imageConfig := publisher.MachineImageConfig{
		LocalPath:    *machineImagePath,
		FileFormat:   *machineImageFormat,
		VolumeSizeGB: int64(*imageVolumeSize),
	}

	for i := range c.AmiRegions {
		go func(regionConfig config.AmiRegion) {
			defer wg.Done()

			switch {
			case regionConfig.IsolatedRegion:
				ds := driverset.NewIsolatedRegionDriverSet(sharedWriter, regionConfig.Credentials)
				p := publisher.NewIsolatedRegionPublisher(sharedWriter, publisher.Config{
					AmiRegion:        regionConfig,
					AmiConfiguration: c.AmiConfiguration,
				})

				amis, err := p.Publish(ds, imageConfig)
				if err != nil {
					errCollection.Add(fmt.Errorf("Error publishing AMIs to %s: %s", regionConfig.RegionName, err))
				} else {
					amiCollection.Merge(amis)
				}
			default:
				ds := driverset.NewStandardRegionDriverSet(sharedWriter, regionConfig.Credentials)
				p := publisher.NewStandardRegionPublisher(sharedWriter, publisher.Config{
					AmiRegion:        regionConfig,
					AmiConfiguration: c.AmiConfiguration,
				})

				amis, err := p.Publish(ds, imageConfig)
				if err != nil {
					errCollection.Add(fmt.Errorf("Error publishing AMIs to %s: %s", regionConfig.RegionName, err))
				} else {
					amiCollection.Merge(amis)
				}
			}
		}(c.AmiRegions[i])
	}

	logger.Println("Waiting for publishers to finish...")
	wg.Wait()

	combinedErr := errCollection.Error()
	if combinedErr != nil {
		logger.Fatal(combinedErr)
	}

	m.PublishedAmis = amiCollection.GetAll()
	err = m.Write(os.Stdout)
	if err != nil {
		logger.Fatalf("writing manifest: %s", err)
	}
	logger.Println("Publishing finished successfully")
}

type logWriter struct {
	sync.Mutex
	writer io.Writer
}

func (l *logWriter) Write(message []byte) (int, error) {
	l.Lock()
	defer l.Unlock()

	return l.writer.Write(message)
}
