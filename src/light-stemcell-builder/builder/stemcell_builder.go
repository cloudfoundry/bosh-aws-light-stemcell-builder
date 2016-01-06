package builder

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
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
	logger    *log.Logger
	aws       ec2.AWS
	awsConfig AwsConfig
	amiConfig ec2ami.Config
	workDir   string
	prepared  bool
}

// AwsConfig specifies credentials to connect to AWS
type AwsConfig struct {
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
}

// New returns a new stemcell builder using the provided AWS configuration
func New(logger *log.Logger, aws ec2.AWS, awsConfig AwsConfig, amiConfig ec2ami.Config) *Builder {
	return &Builder{
		logger:    logger,
		aws:       aws,
		awsConfig: awsConfig,
		amiConfig: amiConfig,
	}
}

func (b *Builder) BuildLightStemcell(stemcellPath string, outputPath string, copyDests []string) (string, map[string]ec2ami.Info, error) {
	err := b.amiConfig.Validate()
	if err != nil {
		return "", nil, err
	}

	imagePath, err := b.Prepare(stemcellPath)
	if err != nil {
		return "", nil, fmt.Errorf("Error during preparing image: %s", err)
	}

	manifestPath := path.Join(b.workDir, "stemcell.MF")
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		err = manifestFile.Close()
	}()

	var amis map[string]ec2ami.Info
	amis, err = b.BuildAmis(imagePath, copyDests)
	if err != nil {
		return "", nil, fmt.Errorf("Error during creating AMIs: %s", err)
	}

	var regionToAmi = make(map[string]string)
	for region, amiInfo := range amis {
		regionToAmi[region] = amiInfo.AmiID
	}

	outputStemcellPath := b.LightStemcellFilePath(stemcellPath, outputPath)
	err = b.UpdateManifestFile(manifestFile, regionToAmi)
	if err != nil {
		return "", nil, err
	}

	err = b.PackageLightStemcell(outputStemcellPath)
	if err != nil {
		return "", nil, err
	}

	return outputStemcellPath, amis, err
}

func (b *Builder) LightStemcellFilePath(heavyStemcellPath string, outputPath string) string {
	lightStemcellPath := "light-" + path.Base(heavyStemcellPath)
	if b.amiConfig.VirtualizationType == "hvm" {
		lightStemcellPath = strings.Replace(lightStemcellPath, "xen", "xen-hvm", 1)
	}
	return path.Join(outputPath, lightStemcellPath)
}

func (b *Builder) UpdateManifestFile(manifestFile io.ReadWriter, regionToAmi map[string]string) error {
	manifest, err := util.ReadYaml(manifestFile)
	if err != nil {
		return fmt.Errorf("Error while reading stemcell manifest: %s", err)
	}

	err = b.UpdateManifestContent(manifest, regionToAmi)
	if err != nil {
		return fmt.Errorf("Error while updating stemcell manifest: %s", err)
	}

	err = util.WriteYaml(manifestFile, manifest)
	if err != nil {
		return fmt.Errorf("Error while writing stemcell manifest: %s", err)
	}

	return nil
}

func (b *Builder) UpdateManifestContent(manifest map[string]interface{}, regionToAmiMap map[string]string) error {
	var stemcellName string
	if val, ok := manifest["name"]; ok {
		stemcellName = val.(string)
	} else {
		return fmt.Errorf("Manifest missing 'name'")
	}

	var cloudProperties map[string]interface{}
	if val, ok := manifest["cloud_properties"]; ok {
		cloudProperties = val.(map[string]interface{})
	} else {
		return fmt.Errorf("Manifest missing 'cloud_properties'")
	}
	if _, ok := cloudProperties["name"]; !ok {
		return fmt.Errorf("Manifest missing 'cloud_properties: name'")
	}

	cloudProperties["ami"] = regionToAmiMap

	if b.amiConfig.VirtualizationType == "hvm" {
		stemcellName = strings.Replace(stemcellName, "xen", "xen-hvm", 1)
		manifest["name"] = stemcellName
		cloudProperties["name"] = stemcellName
	}
	return nil
}

func (b *Builder) PackageLightStemcell(outputFile string) error {
	if !b.prepared {
		return fmt.Errorf("Please call Prepare() before PackageLightStemcell")
	}
	// Overwrite the image archive with an empty file for building the light stemcell
	imagePath := path.Join(b.workDir, "image")
	imageFile, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("Error while creating image file: %s", err)
	}
	err = imageFile.Close()
	if err != nil {
		return fmt.Errorf("Error while closing image file: %s", err)
	}

	tarStemcellCmd := exec.Command("tar", "-C", b.workDir, "-czf", outputFile, "--", "image", "apply_spec.yml", "stemcell.MF", "stemcell_dpkg_l.txt")
	stderr := &bytes.Buffer{}
	tarStemcellCmd.Stderr = stderr

	err = tarStemcellCmd.Run()
	if err != nil {
		return fmt.Errorf("Error packaging light stemcell: %s, stderr: %s", err.Error(), stderr.String())
	}
	return nil
}

// Prepare extracts the machine image from a heavy stemcell and return its path
func (b *Builder) Prepare(stemcellPath string) (string, error) {
	tempDir, err := ioutil.TempDir("", "light-stemcell-builder")
	if err != nil {
		return "", err
	}
	b.workDir = tempDir

	cmd := exec.Command("tar", "-C", b.workDir, "-xf", stemcellPath)
	err = cmd.Run()
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

	b.prepared = true

	return rootImgPath, nil
}

func (b *Builder) BuildAmis(imagePath string, copyDests []string) (map[string]ec2ami.Info, error) {
	ec2Config := ec2.Config{
		BucketName: b.awsConfig.BucketName,
		Region:     b.awsConfig.Region,
		Credentials: &ec2.Credentials{
			AccessKey: b.awsConfig.AccessKey,
			SecretKey: b.awsConfig.SecretKey,
		},
	}

	b.aws.Configure(ec2Config)

	ebsVolumeStage := ec2stage.NewCreateEBSVolumeStage(ec2.ImportVolume,
		ec2.CleanupImportVolume, ec2.DeleteVolume, b.aws)

	createAmiStage := ec2stage.NewCreateAmiStage(ec2.CreateAmi, ec2.DeleteAmi, b.aws, b.amiConfig)

	copyAmiStage := ec2stage.NewCopyAmiStage(ec2.CopyAmis, ec2.DeleteCopiedAmis, b.aws, copyDests)

	outputData, err := stage.RunStages(b.logger, []stage.Stage{ebsVolumeStage, createAmiStage, copyAmiStage}, imagePath)
	if err != nil {
		return nil, err
	}

	// Delete the EBS Volume created for the AMI; We no longer need it at this point
	err = ec2.DeleteVolume(b.aws, outputData[0].(string))
	if err != nil {
		b.logger.Printf("Unable to clean up volume due to error: %s\n", err.Error())
	}

	copiedAmiCollection := outputData[2].(*ec2ami.Collection)
	resultMap := copiedAmiCollection.GetAll()
	resultMap[b.amiConfig.Region] = outputData[1].(ec2ami.Info)

	return resultMap, nil
}
