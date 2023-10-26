package resources

// KmsDriver abstracts the creation of a snapshot in AWS
//
//counterfeiter:generate . KmsDriver
type KmsDriver interface {
	CreateAlias(KmsCreateAliasDriverConfig) (KmsAlias, error)
	ReplicateKey(KmsReplicateKeyDriverConfig) (KmsKey, error)
}

type KmsAlias struct {
	ARN         string
	TargetKeyId string
}

type KmsKey struct {
	ARN string
}

type KmsCreateAliasDriverConfig struct {
	KmsKeyAliasName string
	KmsKeyId        string
	Region          string
}

type KmsReplicateKeyDriverConfig struct {
	KmsKeyId     string
	SourceRegion string
	TargetRegion string
}
