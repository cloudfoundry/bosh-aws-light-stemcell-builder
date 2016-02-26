package manifest

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"light-stemcell-builder/ec2/ec2ami"
	"strings"

	"gopkg.in/yaml.v2"
)

// Manifest represents the stemcell manifest. We don't care about anything
// other than cloud_properties and the name
type Manifest struct {
	Name            string        `yaml:"name"`
	Version         interface{}   `yaml:"version"`
	BoshProtocol    interface{}   `yaml:"bosh_protocol"`
	Sha1            interface{}   `yaml:"sha1"`
	OperatingSystem interface{}   `yaml:"operating_system"`
	CloudProperties yaml.MapSlice `yaml:"cloud_properties"`
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

// ToYAML writes the YAML representation of this manifest to the io.Writer
func (m *Manifest) ToYAML(writer io.Writer) error {
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

// AddAMICollection adds the collection to the manifest
func (m *Manifest) AddAMICollection(a *ec2ami.Collection) error {
	if a == nil {
		return errors.New("AMI Collection is nil")
	}
	m.CloudProperties = append(m.CloudProperties, yaml.MapItem{Key: "ami", Value: a})
	return nil
}

// SetHVM designates this stemcell as an HVM stemcell and sets the name accordingly
func (m *Manifest) SetHVM() {
	name := strings.Replace(m.Name, "xen", "xen-hvm", 1)
	index := -1
	for i, item := range m.CloudProperties {
		if item.Key == "name" {
			index = i
			break
		}
	}
	if index >= 0 {
		m.CloudProperties = append(m.CloudProperties[:index], m.CloudProperties[index+1:]...)
	}
	m.CloudProperties = append(yaml.MapSlice{yaml.MapItem{Key: "name", Value: name}}, m.CloudProperties...)
	m.Name = name
}
