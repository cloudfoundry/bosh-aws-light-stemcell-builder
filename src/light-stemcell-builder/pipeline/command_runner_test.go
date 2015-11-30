package pipeline_test

import (
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/pipeline"
	"os"
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

			cmds := []*exec.Cmd{ps, head, awk}

			out, err := pipeline.Run(cmds)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("PID"))
		})

		It("sets the standard error stream of each command to STDERR", func() {
			ps := exec.Command("ps")
			awk := exec.Command("awk")

			cmds := []*exec.Cmd{ps, awk}

			out, err := pipeline.Run(cmds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 1 of 2"))
			Expect(out).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("usage: awk"))
		})

		It("stops running commands at the first failure", func() {
			tempFile, err := ioutil.TempFile("", "test-command-failure")
			tempFileName := tempFile.Name()

			Expect(err).ToNot(HaveOccurred())
			err = tempFile.Close()
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tempFileName)

			ps := exec.Command("ps")
			awk := exec.Command("awk")

			// requires a *NIX environment to read 1KB from /dev/random to tempFile
			dd := exec.Command("dd", "if=/dev/random", fmt.Sprintf("of=%s", tempFile.Name()), "bs=1024", "count=1")

			cmds := []*exec.Cmd{ps, awk, dd}

			_, err = pipeline.Run(cmds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 1 of 3"))

			tempFile, err = os.Open(tempFileName)
			Expect(err).ToNot(HaveOccurred())
			defer tempFile.Close()

			tempFileBytes, err := ioutil.ReadAll(tempFile)
			Expect(tempFileBytes).To(BeEmpty())
		})
	})

	Describe("RunWithTimeout()", func() {
		It("chains the STDOUT of one command to the STDIN of the next", func() {
			ps := exec.Command("ps")
			head := exec.Command("head", "-1")
			awk := exec.Command("awk", "{print $1}")

			cmds := []*exec.Cmd{ps, head, awk}

			out, err := pipeline.RunWithTimeout(5*time.Second, cmds)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("PID"))
		})

		It("returns the STDERR if a command in the pipeline fails", func() {
			ps := exec.Command("ps")
			awk := exec.Command("awk")

			cmds := []*exec.Cmd{ps, awk}

			out, err := pipeline.RunWithTimeout(5*time.Second, cmds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 1 of 2"))
			Expect(out).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("usage: awk"))
		})

		It("returns an error when the timeout limit is exceeded", func() {
			sleep := exec.Command("sleep", "0.1") // 100ms

			cmds := []*exec.Cmd{sleep}

			out, err := pipeline.RunWithTimeout(5*time.Millisecond, cmds)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("command timed out after 5ms"))
			Expect(out).To(BeEmpty())
		})
	})
})
