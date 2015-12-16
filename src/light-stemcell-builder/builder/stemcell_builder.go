package builder

import (
	"io/ioutil"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2cli"
	"light-stemcell-builder/ec2/ec2stage"
	"light-stemcell-builder/stage"
	"log"
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

func (b *Builder) BuildLightStemcells(imagePath string, awsConfig AwsConfig, copyDests []string) (map[string]ec2ami.Info, error) {
	ec2Config := ec2.Config{
		BucketName: awsConfig.BucketName,
		Region:     awsConfig.Region,
		Credentials: &ec2.Credentials{
			AccessKey: awsConfig.AccessKey,
			SecretKey: awsConfig.SecretKey,
		},
	}
	amiConfig := ec2ami.Config{
		Region:             awsConfig.Region,
		VirtualizationType: "hvm",
		Description:        "BOSH CI test AMI",
	}

	ec2CLI := &ec2cli.EC2Cli{}
	ec2CLI.Configure(ec2Config)

	ebsVolumeStage := ec2stage.NewCreateEBSVolumeStage(ec2.ImportVolume,
		ec2.CleanupImportVolume, ec2.DeleteVolume, ec2CLI)

	createAmiStage := ec2stage.NewCreateAmiStage(ec2.CreateAmi, ec2.DeleteAmi, ec2CLI, amiConfig)

	copyAmiStage := ec2stage.NewCopyAmiStage(ec2.CopyAmis, ec2.DeleteCopiedAmis, ec2CLI, copyDests)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	outputData, err := stage.RunStages(logger, []stage.Stage{ebsVolumeStage, createAmiStage, copyAmiStage}, imagePath)
	if err != nil {
		return nil, err
	}

	// Delete the EBS Volume created for the AMI; We no longer need it at this point
	err = ec2.DeleteVolume(ec2CLI, outputData[0].(string))
	if err != nil {
		logger.Printf("Unable to clean up volume due to error: %s\n", err.Error())
	}

	copiedAmiCollection := outputData[2].(*ec2ami.Collection)
	resultMap := copiedAmiCollection.GetAll()
	resultMap[awsConfig.Region] = outputData[1].(ec2ami.Info)

	return resultMap, nil
}

func (b *Builder) DeleteLightStemcells(awsConfig AwsConfig, amis map[string]ec2ami.Info) error {
	ec2Config := ec2.Config{
		BucketName: awsConfig.BucketName,
		Region:     awsConfig.Region,
		Credentials: &ec2.Credentials{
			AccessKey: awsConfig.AccessKey,
			SecretKey: awsConfig.SecretKey,
		},
	}
	ec2CLI := &ec2cli.EC2Cli{}
	ec2CLI.Configure(ec2Config)

	for _, amiInfo := range amis {
		err := ec2.DeleteAmi(ec2CLI, amiInfo)
		if err != nil {
			return err
		}
	}
	return nil
}
