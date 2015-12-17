package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func YamlToJson(pathToFile string) ([]byte, error) {
	yamlFile, err := os.Open(pathToFile)
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()

	yamlToJsonCmd := exec.Command("yaml2json")
	yamlToJsonCmd.Stdin = yamlFile

	errBuff := &bytes.Buffer{}
	yamlToJsonCmd.Stderr = errBuff

	output, err := yamlToJsonCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("YamlToJson failed: %s, stderr: %s", err.Error(), errBuff.String())
	}
	return output, nil
}

func JsonToYamlFile(inputJson []byte, pathToOutputFile string) error {
	outFile, err := os.Create(pathToOutputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	jsonToYamlCmd := exec.Command("json2yaml")
	stdin, err := jsonToYamlCmd.StdinPipe()
	if err != nil {
		return err
	}

	_, err = stdin.Write(inputJson)
	if err != nil {
		return err
	}
	stdin.Close()
	if err != nil {
		return err
	}

	jsonToYamlCmd.Stdout = outFile
	errBuff := &bytes.Buffer{}

	jsonToYamlCmd.Stderr = errBuff

	err = jsonToYamlCmd.Run()
	if err != nil {
		return fmt.Errorf("YamlToJson failed: %s, stderr: %s", err.Error(), errBuff.String())
	}
	return nil
}
