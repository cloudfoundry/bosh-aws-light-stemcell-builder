package ec2

import "fmt"

const (
	SnapshotCompletedStatus = "completed"
	SnapshotUnknownStatus   = "unknown"
)

type SnapshotInfo struct {
	SnapshotStatus string
}

type SnapshotResource struct {
	SnapshotID     string
	SnapshotRegion string
}

func (e SnapshotResource) ID() string {
	return e.SnapshotID
}

type NonCompletedSnapshotError struct {
	SnapshotID     string
	SnapshotStatus string
}

func (e NonCompletedSnapshotError) Error() string {
	return fmt.Sprintf("Snapshot with id: %s is not available due to status: %s", e.SnapshotID, e.SnapshotStatus)
}

func (i SnapshotInfo) Status() string {
	return i.SnapshotStatus
}
