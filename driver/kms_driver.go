package driver

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

type SDKKmsDriver struct {
	creds  config.Credentials
	logger *log.Logger
}

func NewKmsDriver(logDest io.Writer, creds config.Credentials) *SDKKmsDriver {
	logger := log.New(logDest, "KmsDriver ", log.LstdFlags)

	return &SDKKmsDriver{creds: creds, logger: logger}
}

func (d *SDKKmsDriver) CreateAlias(driverConfig resources.KmsCreateAliasDriverConfig) (resources.KmsAlias, error) {
	if driverConfig.KmsKeyId == "" {
		return resources.KmsAlias{}, nil
	}

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("Completed CreateKeyAlias() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	kmsClient := d.createKmsClient(driverConfig.Region)

	d.logger.Printf("Creating alias: %s\n", driverConfig.KmsKeyAliasName)
	_, err := kmsClient.CreateAlias(&kms.CreateAliasInput{
		AliasName:   &driverConfig.KmsKeyAliasName,
		TargetKeyId: &driverConfig.KmsKeyId,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kms.ErrCodeAlreadyExistsException:
				d.logger.Printf("Alias %s already exists\n", driverConfig.KmsKeyAliasName)
			default:
				return resources.KmsAlias{}, fmt.Errorf("failed to create alias: %s", err)
			}
		} else {
			return resources.KmsAlias{}, fmt.Errorf("failed to create alias: %s", err)
		}
	}

	d.logger.Printf("Checking existence of alias: %s\n", driverConfig.KmsKeyAliasName)
	listAliasResult, err := kmsClient.ListAliases(&kms.ListAliasesInput{
		KeyId: &driverConfig.KmsKeyId,
	})
	if err != nil {
		return resources.KmsAlias{}, fmt.Errorf("checking alias existence: %s", err)
	}

	for i := range listAliasResult.Aliases {
		if *listAliasResult.Aliases[i].AliasName == driverConfig.KmsKeyAliasName {
			d.logger.Printf("Reusing existing alias: %s\n", driverConfig.KmsKeyAliasName)
			return resources.KmsAlias{
				TargetKeyId: *listAliasResult.Aliases[i].TargetKeyId,
				ARN:         *listAliasResult.Aliases[i].AliasArn,
			}, nil
		}
	}

	return resources.KmsAlias{}, fmt.Errorf("could not find existing alias: %s", err)
}

func (d *SDKKmsDriver) ReplicateKey(driverConfig resources.KmsReplicateKeyDriverConfig) (resources.KmsKey, error) {
	if driverConfig.KmsKeyId == "" {
		return resources.KmsKey{}, nil
	}

	createStartTime := time.Now()
	defer func(startTime time.Time) {
		d.logger.Printf("Completed ReplicateKey() in %f minutes\n", time.Since(startTime).Minutes())
	}(createStartTime)

	d.logger.Printf("Replicating kms key: %s\n", driverConfig.KmsKeyId)
	_, err := d.createKmsClient(driverConfig.SourceRegion).ReplicateKey(&kms.ReplicateKeyInput{
		KeyId:         &driverConfig.KmsKeyId,
		ReplicaRegion: &driverConfig.TargetRegion,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kms.ErrCodeAlreadyExistsException:
				d.logger.Printf("Kms key %s already replicated\n", driverConfig.KmsKeyId)
			default:
				return resources.KmsKey{}, fmt.Errorf("failed to replicate key: %s", err)
			}
		} else {
			return resources.KmsKey{}, fmt.Errorf("failed to replicate key: %s", err)
		}
	}

	listKeyResult, err := d.createKmsClient(driverConfig.TargetRegion).ListKeys(&kms.ListKeysInput{})
	for i := range listKeyResult.Keys {
		if strings.HasSuffix(driverConfig.KmsKeyId, *listKeyResult.Keys[i].KeyId) {
			return resources.KmsKey{
				ARN: *listKeyResult.Keys[i].KeyArn,
			}, nil
		}
	}

	return resources.KmsKey{}, fmt.Errorf("could not replicated kms key: %s", err)
}

func (d *SDKKmsDriver) createKmsClient(region string) *kms.KMS {
	creds := config.Credentials{
		AccessKey: d.creds.AccessKey,
		SecretKey: d.creds.SecretKey,
		RoleArn:   d.creds.RoleArn,
		Region:    region,
	}

	awsConfig := creds.GetAwsConfig().
		WithLogger(newDriverLogger(d.logger))

	return kms.New(session.Must(session.NewSession(awsConfig)))
}
