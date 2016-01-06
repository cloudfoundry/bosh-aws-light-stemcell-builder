package ec2

const (
	ConversionTaskCompletedStatus = "completed"
)

type ConversionTaskInfo struct {
	ConversionStatus string
	EBSVolumeID      string
	ManifestUrl      string
	TaskID           string
}

func (i ConversionTaskInfo) Status() string {
	return i.ConversionStatus
}

type ConversionTaskResource struct {
	TaskID string
}

func (e ConversionTaskResource) ID() string {
	return e.TaskID
}
