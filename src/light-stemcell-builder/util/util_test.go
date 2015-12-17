package util_test

import (
	"io/ioutil"
	"light-stemcell-builder/util"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("YamlToJson", func() {
		It("should return the json converted output of the yaml file", func() {
			tempDir, err := ioutil.TempDir("", "light-stemcell-builder-util-test")
			Expect(err).ToNot(HaveOccurred())
			testYamlPath := path.Join(tempDir, "test.yml")
			yamlFile, err := os.Create(testYamlPath)
			Expect(err).ToNot(HaveOccurred())
			yamlFile.Write([]byte("hi: hello\nhello: there"))
			err = yamlFile.Close()
			Expect(err).ToNot(HaveOccurred())

			output, err := util.YamlToJson(testYamlPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(MatchJSON(`{"hi": "hello","hello": "there"}`))
		})
	})
	Describe("JsonToYamlFile", func() {
		It("should write the yaml version of the json input", func() {
			tempDir, err := ioutil.TempDir("", "light-stemcell-builder-util-test")
			Expect(err).ToNot(HaveOccurred())
			testYamlPath := path.Join(tempDir, "test.yml")

			jsonInput := []byte(`{"hi": "hello","hello": "there"}`)

			err = util.JsonToYamlFile(jsonInput, testYamlPath)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(testYamlPath)
			Expect(err).ToNot(HaveOccurred())

			yamlContents, err := ioutil.ReadFile(testYamlPath)
			Expect(string(yamlContents)).To(ContainSubstring("hi: hello"))
			Expect(string(yamlContents)).To(ContainSubstring("hello: there"))
		})
	})
})
