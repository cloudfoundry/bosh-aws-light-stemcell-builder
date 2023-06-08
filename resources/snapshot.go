package resources

// SnapshotDriver abstracts the creation of a snapshot in AWS
//
//counterfeiter:generate -o fakes/fake_snapshot_driver.go . SnapshotDriver
type SnapshotDriver interface {
	Create(SnapshotDriverConfig) (Snapshot, error)
}

// Snapshot represents an EBS snapshot which can be used to create an AMI
type Snapshot struct {
	ID string
}

// SnapshotDriverConfig contains information used to create a snapshot from either an EBS volume or machine image
type SnapshotDriverConfig struct {
	VolumeID string

	MachineImageURL string
	FileFormat      string
}
