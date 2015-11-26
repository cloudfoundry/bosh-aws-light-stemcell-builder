package builder

import (
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/pipeline"
	"os"
	"os/exec"
	"path"
)

// Builder is responsible for extracting the contents of a heavy stemcell
// and for publishing an AWS light stemcell from a machine image
type Builder struct {
	awsConfig AwsConfig
	workDir   string
}

// AwsConfig specifies credentials to connect to AWS
type AwsConfig struct {
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
}

var regionToEndpointMapping = map[string]string{
	"us-east-1":      "https://ec2.us-east-1.amazonaws.com",
	"us-west-2":      "https://ec2.us-west-2.amazonaws.com",
	"us-west-1":      "https://ec2.us-west-1.amazonaws.com",
	"eu-west-1":      "https://ec2.eu-west-1.amazonaws.com",
	"eu-central-1":   "https://ec2.eu-central-1.amazonaws.com",
	"ap-southeast-1": "https://ec2.ap-southeast-1.amazonaws.com",
	"ap-southeast-2": "https://ec2.ap-southeast-2.amazonaws.com",
	"ap-northeast-1": "https://ec2.ap-northeast-1.amazonaws.com",
	"sa-east-1":      "https://ec2.sa-east-1.amazonaws.com",
	"cn-north-1":     "https://ec2.cn-north-1.amazonaws.com.cn",
}

// New returns a new stemcell builder using the provided AWS configuration
func New(c AwsConfig) (*Builder, error) {
	tempDir, err := ioutil.TempDir("", "light-stemcell-builder")
	if err != nil {
		return nil, err
	}

	return &Builder{awsConfig: c, workDir: tempDir}, nil
}

// PrepareHeavy extracts the machine image from a heavy stemcell and return its path
func (b *Builder) PrepareHeavy(stemcellPath string) (string, error) {
	cmd := exec.Command("tar", "-C", b.workDir, "-xf", stemcellPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	imagePath := path.Join(b.workDir, "image")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return "", err
	}

	cmd = exec.Command("tar", "-C", b.workDir, "-xf", imagePath)
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	rootImgPath := path.Join(b.workDir, "root.img")
	if _, err := os.Stat(rootImgPath); os.IsNotExist(err) {
		return "", err
	}

	return rootImgPath, nil
}

// ImportImage creates a single AMI from a source machine image
func (b *Builder) ImportImage(imagePath string) (string, error) {
	taskID, _ := b.importVolume(imagePath)
	return taskID, nil
}

func (b *Builder) importVolume(imagePath string) (string, error) {
	zone := fmt.Sprintf("%sa", b.awsConfig.Region)

	importImage := exec.Command(
		"ec2-import-volume",
		"-f", "RAW",
		"-b", b.awsConfig.BucketName,
		"-o", b.awsConfig.AccessKey,
		"-w", b.awsConfig.SecretKey,
		"-O", b.awsConfig.AccessKey,
		"-W", b.awsConfig.SecretKey,
		"-z", zone,
		"-U", regionToEndpointMapping[b.awsConfig.Region],
		"--no-upload",
		imagePath,
	)

	// We expect to parse output of the form:
	//
	// Requesting volume size: 3 GB
	// TaskType  IMPORTVOLUME  TaskId  import-vol-fggu8ihs ExpirationTime  2015-12-01T21:51:13Z  Status  active  StatusMessage Pending
	// DISKIMAGE DiskImageFormat RAW DiskImageSize 3221225472  VolumeSize  3 AvailabilityZone  cn-north-1b ApproximateBytesConverted 0
	sed := exec.Command("sed", "-n", "2,2p")
	awk := exec.Command("awk", "{print $4}")

	taskID, err := pipeline.Run(os.Stderr, importImage, sed, awk)
	if err != nil {
		return "", fmt.Errorf("creating import volume task: %s", err)
	}

	return taskID, nil
}

func (b *Builder) uploadImage(taskID string) (string, error) {
	return "", nil
}

func (b *Builder) describeTask(taskID string) (string, error) {
	describeTask := exec.Command("ec2-describe-conversion-tasks", taskID)

	// We expect to parse output of the form:
	//
	// TaskType	IMPORTVOLUME	TaskId	import-vol-fg1rl0n6	ExpirationTime	2015-12-02T23:43:30Z	Status	active	StatusMessage	Pending
	// DISKIMAGE	DiskImageFormat	RAW	DiskImageSize	3221225472	VolumeSize	3	AvailabilityZone	cn-north-1a	ApproximateBytesConverted	0
	head := exec.Command("head", "-1")
	awk := exec.Command("awk", "{print $8}")

	taskStatus, err := pipeline.Run(os.Stderr, describeTask, head, awk)
	if err != nil {
		return "", fmt.Errorf("describing conversion task: %s", err)
	}

	return taskStatus, nil
}
