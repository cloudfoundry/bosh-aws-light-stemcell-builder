package manifest

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"light-stemcell-builder/resources"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Manifest represents the stemcell manifest. We don't care about anything
// other than cloud_properties and the name
type Manifest struct {
	Name            string          `yaml:"name"`
	Version         string          `yaml:"version"`
	ApiVersion      int             `yaml:"api_version,omitempty"`
	BoshProtocol    string          `yaml:"bosh_protocol"`
	Sha1            string          `yaml:"sha1"`
	OperatingSystem string          `yaml:"operating_system"`
	StemcellFormats []string        `yaml:"stemcell_formats"`
	CloudProperties CloudProperties `yaml:"cloud_properties"`
	PublishedAmis   []resources.Ami `yaml:"-"`
}

// RegionToAmiMapping is a simple map of AWS region to AMI ID in that region
type RegionToAmiMapping map[string]string

// CloudProperties contains our region to AMI ID mapping and Infrastructure
type CloudProperties struct {
	Infrastructure string             `yaml:"infrastructure"`
	Amis           RegionToAmiMapping `yaml:"ami"`
}

// NewFromReader creates a new manifest from the YAML stored in the reader
func NewFromReader(reader io.Reader) (*Manifest, error) {
	manifestBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %s", err)
	}

	m := &Manifest{}
	err = yaml.Unmarshal(manifestBytes, m)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling YAML to manifest: %s", err)
	}

	return m, nil
}

// Write writes the YAML representation of this manifest to the io.Writer
func (m *Manifest) Write(writer io.Writer) error {
	if len(m.PublishedAmis) == 0 {
		return errors.New("no Amis have been added to the manifest")
	}

	m.StemcellFormats = []string{
		"aws-light",
	}

	m.CloudProperties.Amis = make(RegionToAmiMapping)

	for i := range m.PublishedAmis {
		ami := m.PublishedAmis[i]
		m.CloudProperties.Amis[ami.Region] = ami.ID
	}

	virtualizationType := m.PublishedAmis[0].VirtualizationType
	if virtualizationType == resources.HvmAmiVirtualization && !strings.Contains(m.Name, "-hvm") {
		m.Name = strings.Replace(m.Name, "xen", "xen-hvm", 1)
	}

	output, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling manifest to YAML: %s", err)
	}
	_, err = writer.Write(output)
	if err != nil {
		return fmt.Errorf("writing YAML: %s", err)
	}
	return nil
}
