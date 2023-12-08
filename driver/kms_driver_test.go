package driver_test

import (
	"math/rand"
	"strconv"
	"strings"

	"light-stemcell-builder/driverset"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("KmsDriver", func() {
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
			awsSession, _ := session.NewSession(creds.GetAwsConfig())
			kmsClient := kms.New(awsSession)
			_, _ = kmsClient.DeleteAlias(&kms.DeleteAliasInput{
				AliasName: &aliasName,
			})
		}(aliasName, aliasCreationResult)

		awsSession, err := session.NewSession(creds.GetAwsConfig())
		Expect(err).ToNot(HaveOccurred())
		kmsClient := kms.New(awsSession)
		listAliasResult, err := kmsClient.ListAliases(&kms.ListAliasesInput{
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

		//defer cleanup of the created key replica, sadly we can only schedule it to be deleted after 7 days
		//therefore this test will reuse the replicated key for 7 days and only afterward create a new one
		defer func(aliasCreationResult resources.KmsKey) {
			destinationKeyId := strings.ReplaceAll(multiRegionKeyReplicationTest, originalRegion, destinationRegion)
			awsSession, _ := session.NewSession(creds.GetAwsConfig())
			kmsClient := kms.New(awsSession)

			_, _ = kmsClient.ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
				KeyId:               &destinationKeyId,
				PendingWindowInDays: aws.Int64(7),
			})
		}(replicateKeyResult)

		awsSession, err := session.NewSession(creds.GetAwsConfig())
		Expect(err).ToNot(HaveOccurred())
		kmsClient := kms.New(awsSession)
		listKeyResult, err := kmsClient.ListKeys(&kms.ListKeysInput{})
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
