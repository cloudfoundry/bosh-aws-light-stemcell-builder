package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"light-stemcell-builder/builder"
	"light-stemcell-builder/config"
	"light-stemcell-builder/ec2/ec2cli"
	"log"
	"os"
)

func usage(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, "Usage of light-stemcell-builder/main.go")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	configPath := flag.String("c", "", "Path to the JSON configuration file")
	inputPath := flag.String("i", "", "Path to the input stemcell")
	outputPath := flag.String("o", "", "Path to the output folder for the light stemcell")

	flag.Parse()

	if *configPath == "" {
		usage("-c flag is required")
	}
	if *inputPath == "" {
		usage("-i flag is required")
	}
	if *outputPath == "" {
		usage("-o flag is required")
	}

	if _, err := os.Stat(*configPath); err != nil {
		usage(fmt.Sprintf("config file was not found: %s", *configPath))
	}
	if _, err := os.Stat(*inputPath); err != nil {
		usage(fmt.Sprintf("input stemcell was not found: %s", *inputPath))
	}
	fileInfo, err := os.Stat(*outputPath)
	if err != nil {
		usage(fmt.Sprintf("output folder was not found: %s", *outputPath))
	}
	if !fileInfo.IsDir() {
		usage(fmt.Sprintf("output folder is not a directory: %s", *outputPath))
	}

	configFile, err := os.Open(*configPath)

	defer func() {
		err = configFile.Close()
		if err != nil {
			logger.Fatalf("Error closing config file: %s", err.Error())
		}
	}()

	if err != nil {
		logger.Fatalf("Error opening config file: %s", err.Error())
	}

	aws := &ec2cli.EC2Cli{}

	c, err := config.NewFromReader(configFile)
	if err != nil {
		logger.Fatalf("Error parsing config file: %s. Message: %s", *configPath, err.Error())
	}

	b := builder.New(aws, c, logger)
	stemcell, amis, err := b.Build(*inputPath, *outputPath)
	if err != nil {
		logger.Fatalf("Error during stemcell builder: %s\n", err)
	}

	amiJSON, err := json.Marshal(amis)
	if err != nil {
		logger.Printf("Error output encoding: %s\n", err)
	}

	logger.Printf("Created AMIs:\n%s", amiJSON)
	logger.Printf("Output saved to: %s\n", stemcell)
}
