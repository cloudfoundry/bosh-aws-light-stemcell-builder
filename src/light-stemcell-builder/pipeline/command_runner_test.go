package pipeline_test

import (
	"bytes"
	"io/ioutil"
	"light-stemcell-builder/pipeline"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CommandRunner", func() {
	It("chains the STDOUT of one command to the STDIN of the next", func() {
		ps := exec.Command("ps")
		head := exec.Command("head", "-1")
		awk := exec.Command("awk", "{print $1}")

		out, err := pipeline.Run(ioutil.Discard, ps, head, awk)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal("PID"))
	})

	It("sets the standard error stream of each command to STDERR", func() {
		b := &bytes.Buffer{}
		ps := exec.Command("ps")
		awk := exec.Command("awk")

		out, err := pipeline.Run(b, ps, awk)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("running command 1 of 2"))
		Expect(out).To(BeEmpty())
		Expect(b.String()).To(ContainSubstring("usage: awk"))
	})
})
