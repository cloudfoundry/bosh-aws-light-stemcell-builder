package ec2

// CleanImportVolume cleans up any artifacts that ImportVolume left behind
// (s3 artifacts)
func CleanupImportVolume(aws AWS, taskID string) error {
	return aws.DeleteDiskImage(taskID)
}
