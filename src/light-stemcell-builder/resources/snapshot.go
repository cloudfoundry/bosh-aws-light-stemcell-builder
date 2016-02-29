package resources

import "sync"

// Volume properties which we do not expect to change
const (
	VolumeFormat       = "RAW"
	VolumeArchitecture = "x86_64"
)

// SnapshotDriver abstracts the creation of a snapshot in AWS
type SnapshotDriver interface {
	Create(SnapshotDriverConfig) (string, error)
}

// Snapshot represents an EBS snapshot which can be used to create an AMI
type Snapshot struct {
	id           string
	driver       SnapshotDriver
	driverConfig SnapshotDriverConfig
	opErr        error
	once         *sync.Once
}

// SnapshotDriverConfig contains information used to create a snapshot from either an EBS volume or machine image
type SnapshotDriverConfig struct {
	VolumeID        string
	MachineImageURL string
}

// WaitForCreation attempts to create a snapshot from an EBS volume returning the ID or error
func (s *Snapshot) WaitForCreation() (string, error) {
	s.once.Do(func() {
		s.id, s.opErr = s.driver.Create(s.driverConfig)
	})

	return s.id, s.opErr
}

// NewSnapshot serves as a snapshot factory, callers call WaitForCreation() to create a snapshot from a volume in AWS
func NewSnapshot(driver SnapshotDriver, driverConfig SnapshotDriverConfig) Snapshot {
	return Snapshot{driver: driver, driverConfig: driverConfig, once: &sync.Once{}}
}
