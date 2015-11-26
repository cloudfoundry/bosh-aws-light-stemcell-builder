package pipeline_test

import (
	"bytes"
	"io/ioutil"
	"light-stemcell-builder/pipeline"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CommandRunner", func() {
	Describe("Run()", func() {
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

	Describe("RunWithTimeout()", func() {
		It("keeps the original Run() behavior when the commands succeeds", func() {
			ps := exec.Command("ps")
			head := exec.Command("head", "-1")
			awk := exec.Command("awk", "{print $1}")

			out, err := pipeline.RunWithTimeout(ioutil.Discard, 5*time.Second, ps, head, awk)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("PID"))
		})

		It("keeps the original Run() behavior when the command fails", func() {
			b := &bytes.Buffer{}
			ps := exec.Command("ps")
			awk := exec.Command("awk")

			out, err := pipeline.RunWithTimeout(b, 5*time.Second, ps, awk)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 1 of 2"))
			Expect(out).To(BeEmpty())
			Expect(b.String()).To(ContainSubstring("usage: awk"))
		})

		It("returns an error when the timeout limit is exceeded", func() {
			sleep := exec.Command("sleep", "0.1") // 100ms

			out, err := pipeline.RunWithTimeout(ioutil.Discard, 5*time.Millisecond, sleep)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("command timed out after 5ms"))
			Expect(out).To(BeEmpty())
		})
	})
})
