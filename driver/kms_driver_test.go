package driver_test

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("KmsDriver", func() {
	BeforeEach(func() {
		if os.Getenv("SKIP_REPLICATION_TESTS") != "" {
			Skip("Skipping test, found 'SKIP_REPLICATION_TESTS'")
		}
	})
	It("creates an alias for a given kms key", func() {
		aliasName := "alias/" + strconv.Itoa(rand.Int())

		driverConfig := resources.KmsCreateAliasDriverConfig{
			KmsKeyAliasName: aliasName,
			KmsKeyId:        kmsKeyId,
			Region:          creds.Region,
		}
		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)
		driver := ds.KmsDriver()

		aliasCreationResult, err := driver.CreateAlias(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		//defer cleanup of the created alias
		defer func(aliasName string, aliasCreationResult resources.KmsAlias) {
			kmsClient := kms.NewFromConfig(creds.GetAwsConfig())
			kmsClient.DeleteAlias(context.Background(), &kms.DeleteAliasInput{ //nolint:errcheck
				AliasName: &aliasName,
			})
		}(aliasName, aliasCreationResult)

		kmsClient := kms.NewFromConfig(creds.GetAwsConfig())
		listAliasResult, err := kmsClient.ListAliases(context.Background(), &kms.ListAliasesInput{
			KeyId: &kmsKeyId,
		})
		Expect(err).ToNot(HaveOccurred())

		aliasCount := 0
		for i := range listAliasResult.Aliases {
			if *listAliasResult.Aliases[i].AliasName == aliasName {
				aliasCount++
			}
		}
		Expect(aliasCount).To(Equal(1))
	})

	It("replicates a given kms key to another region", func() {
		driverConfig := resources.KmsReplicateKeyDriverConfig{
			KmsKeyId:     multiRegionKeyReplicationTest,
			SourceRegion: creds.Region,
			TargetRegion: destinationRegion,
		}
		ds := driverset.NewStandardRegionDriverSet(GinkgoWriter, creds)
		driver := ds.KmsDriver()

		replicateKeyResult, err := driver.ReplicateKey(driverConfig)
		Expect(err).ToNot(HaveOccurred())

		originalRegion := creds.Region
		creds.Region = destinationRegion

		//defer cleanup of the created key replica
		defer func(aliasCreationResult resources.KmsKey) {
			destinationKeyId := strings.ReplaceAll(multiRegionKeyReplicationTest, originalRegion, destinationRegion)
			kmsClient := kms.NewFromConfig(creds.GetAwsConfig())

			kmsClient.ScheduleKeyDeletion(context.Background(), &kms.ScheduleKeyDeletionInput{ //nolint:errcheck
				KeyId:               &destinationKeyId,
				PendingWindowInDays: aws.Int32(7),
			})
		}(replicateKeyResult)

		kmsClient := kms.NewFromConfig(creds.GetAwsConfig())
		listKeyResult, err := kmsClient.ListKeys(context.Background(), &kms.ListKeysInput{})
		Expect(err).ToNot(HaveOccurred())

		keysCount := 0
		for i := range listKeyResult.Keys {
			if strings.HasSuffix(driverConfig.KmsKeyId, *listKeyResult.Keys[i].KeyId) {
				keysCount++
			}
		}
		Expect(keysCount).To(Equal(1))
	})
})
