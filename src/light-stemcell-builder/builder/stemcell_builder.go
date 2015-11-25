package builder

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

// Builder is responsible for extracting the contents of a heavy stemcell
// and for publishing an AWS light stemcell from a machine image
type Builder struct {
	workDir string
}

// New returns a new stemcell builder
func New() (*Builder, error) {
	tempDir, err := ioutil.TempDir("", "light-stemcell-builder")
	if err != nil {
		return nil, err
	}

	return &Builder{workDir: tempDir}, nil
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
