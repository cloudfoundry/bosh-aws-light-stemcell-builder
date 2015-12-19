package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2cli"
	"light-stemcell-builder/ec2/ec2stage"
	"light-stemcell-builder/stage"
	"light-stemcell-builder/util"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
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

// New returns a new stemcell builder using the provided AWS configuration
func New(c AwsConfig) (*Builder, error) {
	tempDir, err := ioutil.TempDir("", "light-stemcell-builder")
	if err != nil {
		return nil, err
	}

	return &Builder{awsConfig: c, workDir: tempDir}, nil
}

func (b *Builder) BuildLightStemcell(logger *log.Logger, stemcellPath string, outputPath string, amiConfig ec2ami.Config, copyDests []string) (string, error) {
	err := amiConfig.Validate()
	if err != nil {
		return "", err
	}

	imagePath, err := b.PrepareHeavy(stemcellPath)
	if err != nil {
		return "", fmt.Errorf("Error during preparing image: %s", err)
	}

	amis, err := b.BuildAmis(logger, imagePath, amiConfig, copyDests)
	if err != nil {
		return "", fmt.Errorf("Error during creating AMIs: %s", err)
	}

	logger.Printf("Created AMIs:")
	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(amis)

	return b.BuildLightStemcellTarball(b.LightStemcellFilePath(stemcellPath, outputPath), amis, amiConfig)
}

func (b *Builder) LightStemcellFilePath(heavyStemcellPath string, outputPath string) string {
	lightStemcellPath := path.Base(heavyStemcellPath)
	lightStemcellPath = "light-" + strings.Replace(lightStemcellPath, "xen", "xen-hvm", 1)
	return path.Join(outputPath, lightStemcellPath)
}

func (b *Builder) BuildLightStemcellTarball(outputFile string, amis map[string]ec2ami.Info, amiConfig ec2ami.Config) (string, error) {
	var regionToAmiMap = make(map[string]string)
	for region, amiInfo := range amis {
		regionToAmiMap[region] = amiInfo.AmiID
	}

	manifestPath := path.Join(b.workDir, "stemcell.MF")

	manifest, err := b.ReadStemcellManifest(manifestPath)
	if err != nil {
		return "", fmt.Errorf("Error while reading stemcell manifest: %s", err)
	}

	stemcellName, err := b.ModifyAndWriteStemcellManifest(manifestPath, manifest, regionToAmiMap, amiConfig)
	if err != nil {
		return "", fmt.Errorf("Error while writing stemcell manifest: %s", err)
	}

	return b.PackageLightStemcell(outputFile, stemcellName)
}

func (b *Builder) ReadStemcellManifest(manifestPath string) (map[string]interface{}, error) {
	jsonManifest, err := util.YamlToJson(manifestPath)

	var manifest map[string]interface{}
	err = json.Unmarshal(jsonManifest, &manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (b *Builder) ModifyAndWriteStemcellManifest(manifestPath string, manifest map[string]interface{}, regionToAmiMap map[string]string, amiConfig ec2ami.Config) (string, error) {
	cloudProperties := manifest["cloud_properties"].(map[string]interface{})
	cloudProperties["ami"] = regionToAmiMap

	stemcellName := manifest["name"].(string)
	if amiConfig.VirtualizationType == "hvm" {
		stemcellName = strings.Replace(stemcellName, "xen", "xen-hvm", 1)
		manifest["name"] = stemcellName
		cloudProperties["name"] = stemcellName
	}

	outputManifest, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}

	err = util.JsonToYamlFile(outputManifest, manifestPath)
	if err != nil {
		return "", err
	}
	return stemcellName, nil
}

func (b *Builder) PackageLightStemcell(outputFile string, stemcellName string) (string, error) {
	// Overwrite the image archive with an empty file for building the light stemcell
	imagePath := path.Join(b.workDir, "image")
	imageFile, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("Error while creating image file: %s", err)
	}
	imageFile.Close()

	tarStemcellCmd := exec.Command("tar", "-C", b.workDir, "-czf", outputFile, "--", "image", "apply_spec.yml", "stemcell.MF", "stemcell_dpkg_l.txt")
	stderr := &bytes.Buffer{}
	tarStemcellCmd.Stderr = stderr

	err = tarStemcellCmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error packaging light stemcell: %s, stderr: %s", err.Error(), stderr.String())
	}
	return outputFile, nil
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

func (b *Builder) BuildAmis(logger *log.Logger, imagePath string, amiConfig ec2ami.Config, copyDests []string) (map[string]ec2ami.Info, error) {
	ec2Config := ec2.Config{
		BucketName: b.awsConfig.BucketName,
		Region:     b.awsConfig.Region,
		Credentials: &ec2.Credentials{
			AccessKey: b.awsConfig.AccessKey,
			SecretKey: b.awsConfig.SecretKey,
		},
	}

	ec2CLI := &ec2cli.EC2Cli{}
	ec2CLI.Configure(ec2Config)

	ebsVolumeStage := ec2stage.NewCreateEBSVolumeStage(ec2.ImportVolume,
		ec2.CleanupImportVolume, ec2.DeleteVolume, ec2CLI)

	createAmiStage := ec2stage.NewCreateAmiStage(ec2.CreateAmi, ec2.DeleteAmi, ec2CLI, amiConfig)

	copyAmiStage := ec2stage.NewCopyAmiStage(ec2.CopyAmis, ec2.DeleteCopiedAmis, ec2CLI, copyDests)

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
	resultMap[b.awsConfig.Region] = outputData[1].(ec2ami.Info)

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
