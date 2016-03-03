package main

import (
	"flag"
	"fmt"
	"io"
	"light-stemcell-builder/config"
	"light-stemcell-builder/driverset"
	"light-stemcell-builder/manifest"
	"light-stemcell-builder/publisher"
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

	f, err := os.Open(*manifestPath)
	if err != nil {
		logger.Fatalf("opening manifest: %s", err)
	}

	m, err := manifest.NewFromReader(f)
	if err != nil {
		logger.Fatalf("reading manifest: %s", err)
	}

	regionConfig := c.AmiRegions[0]

	ds := driverset.NewStandardRegionDriverSet(sharedWriter, regionConfig.Credentials)

	p := publisher.NewStandardRegionPublisher(publisher.Config{
		AmiRegion:        regionConfig,
		AmiConfiguration: c.AmiConfiguration,
	})

	amiCollection, err := p.Publish(ds, *machineImagePath)
	if err != nil {
		logger.Fatalf("Error publishing AMIs: %s", err)
	}

	m.PublishedAmis = amiCollection.GetAll()
	m.Write(os.Stdout)
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
