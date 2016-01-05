package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

func yamlToJson(reader io.Reader) ([]byte, error) {
	yamlToJsonCmd := exec.Command("yaml2json")
	yamlToJsonCmd.Stdin = reader

	errBuff := &bytes.Buffer{}
	yamlToJsonCmd.Stderr = errBuff

	output, err := yamlToJsonCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("YamlToJson failed: %s, stderr: %s", err.Error(), errBuff.String())
	}
	return output, nil
}

func ReadYaml(manifestReader io.Reader) (map[string]interface{}, error) {
	jsonManifest, err := yamlToJson(manifestReader)
	if err != nil {
		return nil, err
	}

	var manifest map[string]interface{}
	err = json.Unmarshal(jsonManifest, &manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func jsonToYaml(inputJson []byte, yamlWriter io.Writer) error {
	jsonToYamlCmd := exec.Command("json2yaml")
	stdin, err := jsonToYamlCmd.StdinPipe()
	if err != nil {
		return err
	}

	_, err = stdin.Write(inputJson)
	if err != nil {
		return err
	}
	err = stdin.Close()
	if err != nil {
		return err
	}

	jsonToYamlCmd.Stdout = yamlWriter
	errBuff := &bytes.Buffer{}

	jsonToYamlCmd.Stderr = errBuff

	err = jsonToYamlCmd.Run()
	if err != nil {
		return fmt.Errorf("YamlToJson failed: %s, stderr: %s", err.Error(), errBuff.String())
	}
	return nil
}

func WriteYaml(manifestWriter io.Writer, manifest map[string]interface{}) error {
	outputManifest, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	err = jsonToYaml(outputManifest, manifestWriter)
	if err != nil {
		return err
	}
	return nil
}
