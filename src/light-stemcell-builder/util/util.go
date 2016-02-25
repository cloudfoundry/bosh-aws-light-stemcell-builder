package util

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ReadYaml maps YAML from a reader to a map and outputs errors when YAML is
// invalid
func ReadYaml(manifestReader io.Reader) (map[string]interface{}, error) {
	byteArray, err := ioutil.ReadAll(manifestReader)
	if err != nil {
		return nil, fmt.Errorf("ReadYaml failed: %s", err)
	}

	manifest := make(map[string]interface{})
	err = yaml.Unmarshal(byteArray, manifest)

	if err != nil {
		return nil, fmt.Errorf("ReadYaml failed: Invalid YAML: %s", err)
	}
	return manifest, nil
}

// WriteYaml writes YAML to a writer
func WriteYaml(manifestWriter io.Writer, manifest map[string]interface{}) error {
	output, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("WriteYaml failed: %s", err.Error())
	}

	_, err = manifestWriter.Write(output)
	if err != nil {
		return fmt.Errorf("WriteYaml failed: %s", err.Error())
	}

	return nil
}
