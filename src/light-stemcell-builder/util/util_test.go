package util_test

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"light-stemcell-builder/util"
)

var _ = Describe("Util", func() {
	Describe("ReadYaml", func() {
		It("reads the yaml into a map", func() {
			reader := bytes.NewReader([]byte("hi: hello\nhello: there"))
			expected := map[string]interface{}{
				"hi":    "hello",
				"hello": "there",
			}
			content, err := util.ReadYaml(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(Equal(expected))
		})
		Context("given invalid yaml", func() {
			It("returns an error", func() {
				reader := bytes.NewReader([]byte("key: key: value"))
				_, err := util.ReadYaml(reader)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("YamlToJson failed"))
			})
		})
	})
	Describe("WriteYaml", func() {
		It("writes the contents as yaml", func() {
			writer := &bytes.Buffer{}
			content := map[string]interface{}{
				"key":     "value",
				"another": "value",
			}
			err := util.WriteYaml(writer, content)
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.String()).To(ContainSubstring("key: value"))
			Expect(writer.String()).To(ContainSubstring("another: value"))
		})
		It("writes empty contents as yaml", func() {
			writer := &bytes.Buffer{}
			content := make(map[string]interface{})
			err := util.WriteYaml(writer, content)
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.String()).To(Equal("{}\n\n"))
		})
	})
})
