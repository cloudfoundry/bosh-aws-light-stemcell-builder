package main

import (
	"encoding/json"
	"fmt"
	"light-stemcell-builder/builder"
	"log"
	"os"
)

type Config struct {
	AccessKey    string   `json:"access_key"`
	SecretKey    string   `json:"secret_key"`
	BucketName   string   `json:"bucket_name"`
	Region       string   `json:"region"`
	StemcellPath string   `json:"stemcell_path"`
	CopyDests    []string `json:"copy_dests"`
}

func main() {

	if len(os.Args) != 2 {
		log.Fatalln("Usage: light-stemcell-builder path_to_config.json")
	}
	pathToConfig := os.Args[1]

	configFile, err := os.Open(pathToConfig)

	if err != nil {
		log.Fatalf("opening config file: %s\n", err.Error())
	}

	config := &Config{}
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(config); err != nil {
		log.Fatalf("Can't parse %s config file. Error is : %s\n", pathToConfig, err.Error())
	}

	err = validateConfig(config)

	var awsConfig = builder.AwsConfig{
		AccessKey:  config.AccessKey,
		SecretKey:  config.SecretKey,
		BucketName: config.BucketName,
		Region:     config.Region,
	}

	stemcellBuilder, err := builder.New(awsConfig)
	if err != nil {
		log.Fatalf("Error during creating image: %s\n", err)
	}

	imagePath, err := stemcellBuilder.PrepareHeavy(config.StemcellPath)
	if err != nil {
		log.Fatalf("Error during preparing image: %s\n", err)
	}

	AMIs, err := stemcellBuilder.BuildLightStemcells(imagePath, awsConfig, config.CopyDests)
	if err != nil {
		log.Fatalf("Error during creating image: %s\n", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(AMIs)
}

func validateConfig(config *Config) error {
	if config.AccessKey == "" {
		return fmt.Errorf("Access key can't be empty")
	}
	if config.SecretKey == "" {
		return fmt.Errorf("Secret key can't be empty")
	}
	if config.BucketName == "" {
		return fmt.Errorf("Bucket name can't be empty")
	}
	if config.Region == "" {
		return fmt.Errorf("Region can't be empty")
	}
	if config.StemcellPath == "" {
		return fmt.Errorf("Stemcell path can't be empty")
	}
	return nil
}
