package builder

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"light-stemcell-builder/config"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/manifest"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Builder is responsible for extracting the contents of a heavy stemcell
// and for publishing an AWS light stemcell from a machine image
type Builder struct {
	aws        ec2.AWS
	config     config.Config
	logger     *log.Logger
	packageDir string
	prepared   bool
}

func New(aws ec2.AWS, c config.Config, logger *log.Logger) *Builder {
	return &Builder{
		aws:    aws,
		config: c,
		logger: logger,
	}
}

func (b *Builder) Build(inputPath string, outputPath string) (string, map[string]ec2ami.Info, error) {
	imagePath, err := b.Prepare(inputPath)
	if err != nil {
		return "", nil, fmt.Errorf("Error during image preparation: %s", err)
	}

	manifestPath := path.Join(b.packageDir, "stemcell.MF")

	amiCollection := ec2ami.NewCollection()
	amiPublisher := AMIPublisher{
		AWS:       b.aws,
		AMIConfig: b.config.AmiConfiguration,
		Logger:    b.logger,
	}

	for _, region := range b.config.AmiRegions {
		result, err := amiPublisher.Publish(imagePath, region)

		if err != nil {
			return "", nil, fmt.Errorf("creating AMI in region %s: %s", region.Name, err)
		}

		for regionName, amiInfo := range result {
			log.Printf("adding AMI: %s for region: %s to AMI collection", regionName, amiInfo.AmiID)
			amiCollection.Add(regionName, amiInfo)
		}
	}

	manifestFileBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return "", nil, err
	}

	manifestFileBuf := bytes.NewBuffer(manifestFileBytes)

	stemcellPath := b.OutputPath(inputPath, outputPath)

	err = b.UpdateManifestFile(manifestFileBuf, amiCollection)
	if err != nil {
		return "", nil, err
	}

	err = ioutil.WriteFile(manifestPath, manifestFileBuf.Bytes(), os.ModePerm)
	if err != nil {
		return "", nil, err
	}

	err = b.Package(stemcellPath)
	if err != nil {
		return "", nil, err
	}

	return stemcellPath, amiCollection.GetAll(), nil
}

// Prepare extracts the machine image from a heavy stemcell and return its path
func (b *Builder) Prepare(stemcellPath string) (string, error) {
	tempDir, err := ioutil.TempDir("", "light-stemcell-builder")
	if err != nil {
		return "", err
	}
	b.packageDir = tempDir

	imageDir, err := ioutil.TempDir("", "input-stemcell-image")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("tar", "-C", b.packageDir, "-xf", stemcellPath)
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	imagePath := path.Join(b.packageDir, "image")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return "", err
	}

	cmd = exec.Command("tar", "-C", imageDir, "-xf", imagePath)
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	rootImgPath := path.Join(imageDir, "root.img")
	if _, err := os.Stat(rootImgPath); os.IsNotExist(err) {
		return "", err
	}

	b.prepared = true

	return rootImgPath, nil
}

func (b *Builder) OutputPath(heavyStemcellPath string, outputPath string) string {
	lightStemcellPath := "light-" + path.Base(heavyStemcellPath)
	if b.config.AmiConfiguration.VirtualizationType == "hvm" {
		lightStemcellPath = strings.Replace(lightStemcellPath, "xen", "xen-hvm", 1)
	}
	return path.Join(outputPath, lightStemcellPath)
}

func (b *Builder) UpdateManifestFile(manifestFile io.ReadWriter, amiCollection *ec2ami.Collection) error {
	manifestStruct, err := manifest.NewFromReader(manifestFile)
	if err != nil {
		return fmt.Errorf("Error while reading stemcell manifest: %s", err)
	}

	if b.config.AmiConfiguration.VirtualizationType == "hvm" {
		manifestStruct.SetHVM()
	}

	err = manifestStruct.AddAMICollection(amiCollection)
	if err != nil {
		return fmt.Errorf("Error while updating stemcell manifest: %s", err)
	}

	err = manifestStruct.ToYAML(manifestFile)
	if err != nil {
		return fmt.Errorf("Error while writing stemcell manifest: %s", err)
	}

	return nil
}

func (b *Builder) Package(outputFile string) error {
	if !b.prepared {
		return fmt.Errorf("Please call Prepare() before Package")
	}
	// Overwrite the image archive with an empty file for building the light stemcell
	imagePath := path.Join(b.packageDir, "image")
	imageFile, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("Error while creating image file: %s", err)
	}
	err = imageFile.Close()
	if err != nil {
		return fmt.Errorf("Error while closing image file: %s", err)
	}

	files, err := ioutil.ReadDir(b.packageDir)
	if err != nil {
		return fmt.Errorf("Error while listing stemcell package files: %s", err)
	}
	var packageFiles []string
	for _, f := range files {
		packageFiles = append(packageFiles, f.Name())
	}
	tarArgs := []string{"-C", b.packageDir, "-czf", outputFile, "--"}
	tarArgs = append(tarArgs, packageFiles...)
	tarStemcellCmd := exec.Command("tar", tarArgs...)

	stderr := &bytes.Buffer{}
	tarStemcellCmd.Stderr = stderr

	err = tarStemcellCmd.Run()
	if err != nil {
		return fmt.Errorf("Error packaging light stemcell: %s, stderr: %s", err.Error(), stderr.String())
	}
	return nil
}
