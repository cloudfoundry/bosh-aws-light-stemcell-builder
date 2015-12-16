package ec2

import "fmt"

type VolumeInfo struct {
	VolumeStatus string
}

type VolumeResource struct {
	VolumeID string
}

func (e VolumeResource) ID() string {
	return e.VolumeID
}

const (
	VolumeAvailableStatus = "available"
	VolumeDeletingStatus  = "deleting"
	VolumeUnknownStatus   = "unknown" // we don't actually know whether the volume was deleted or never existed
)

type NonAvailableVolumeError struct {
	VolumeID     string
	VolumeStatus string
}

func (e NonAvailableVolumeError) Error() string {
	return fmt.Sprintf("volume with id: %s is not available due to status: %s", e.VolumeID, e.VolumeStatus)
}

func (info VolumeInfo) Status() string {
	return info.VolumeStatus
}
