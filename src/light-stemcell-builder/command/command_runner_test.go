package command_test

import (
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/command"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CommandRunner", func() {
	Describe("NewSelectLine", func() {
		It("retuns a command which selects a given line", func() {
			secondLine, err := command.SelectLine(2)
			Expect(err).ToNot(HaveOccurred())

			f, err := ioutil.TempFile("", "3linefile")
			Expect(err).ToNot(HaveOccurred())
			fileName := f.Name()
			defer os.Remove(fileName)

			cat := exec.Command("cat", fileName)

			_, err = f.Write([]byte("one\ntwo\nthree"))
			Expect(err).ToNot(HaveOccurred())

			err = f.Close()
			Expect(err).ToNot(HaveOccurred())

			out, err := command.RunPipeline([]*exec.Cmd{cat, secondLine})
			Expect(err).ToNot(HaveOccurred())

			Expect(out).To(Equal("two"))
		})

		It("returns an error for non-positive line selections", func() {
			_, err := command.SelectLine(0)
			Expect(err).To(MatchError("line selection 0 is not positive"))

			_, err = command.SelectLine(-1)
			Expect(err).To(MatchError("line selection -1 is not positive"))
		})
	})

	Describe("NewSelectField", func() {
		It("returns a command which selects a given field", func() {
			secondField, err := command.SelectField(2)
			Expect(err).ToNot(HaveOccurred())

			f, err := ioutil.TempFile("", "3fieldfile")
			Expect(err).ToNot(HaveOccurred())
			fileName := f.Name()
			defer os.Remove(fileName)

			cat := exec.Command("cat", fileName)

			_, err = f.Write([]byte("one   two three"))
			Expect(err).ToNot(HaveOccurred())

			err = f.Close()
			Expect(err).ToNot(HaveOccurred())

			out, err := command.RunPipeline([]*exec.Cmd{cat, secondField})
			Expect(err).ToNot(HaveOccurred())

			Expect(out).To(Equal("two"))
		})

		It("returns an error for non-positive field selections", func() {
			_, err := command.SelectField(0)
			Expect(err).To(MatchError("field selection 0 is not positive"))

			_, err = command.SelectField(-1)
			Expect(err).To(MatchError("field selection -1 is not positive"))
		})
	})

	Describe("RunPipeline()", func() {
		It("chains the STDOUT of one command to the STDIN of the next", func() {
			ps := exec.Command("ps")
			head := exec.Command("head", "-1")
			awk := exec.Command("awk", "{print $1}")

			cmds := []*exec.Cmd{ps, head, awk}

			out, err := command.RunPipeline(cmds)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("PID"))
		})

		It("sets the standard error stream of each command to STDERR", func() {
			ps := exec.Command("ps")
			awk := exec.Command("awk")

			cmds := []*exec.Cmd{ps, awk}

			out, err := command.RunPipeline(cmds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 2 of 2"))
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

			_, err = command.RunPipeline(cmds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 2 of 3"))

			tempFile, err = os.Open(tempFileName)
			Expect(err).ToNot(HaveOccurred())
			defer tempFile.Close()

			tempFileBytes, err := ioutil.ReadAll(tempFile)
			Expect(tempFileBytes).To(BeEmpty())
		})
	})

	Describe("RunPipelineWithTimeout()", func() {
		It("chains the STDOUT of one command to the STDIN of the next", func() {
			ps := exec.Command("ps")
			head := exec.Command("head", "-1")
			awk := exec.Command("awk", "{print $1}")

			cmds := []*exec.Cmd{ps, head, awk}

			out, err := command.RunPipelineWithTimeout(5*time.Second, cmds)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("PID"))
		})

		It("returns the STDERR if a command in the pipeline fails", func() {
			ps := exec.Command("ps")
			awk := exec.Command("awk")

			cmds := []*exec.Cmd{ps, awk}

			out, err := command.RunPipelineWithTimeout(5*time.Second, cmds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("running command 2 of 2"))
			Expect(out).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("usage: awk"))
		})

		It("returns an error when the timeout limit is exceeded", func() {
			sleep := exec.Command("sleep", "0.1") // 100ms

			cmds := []*exec.Cmd{sleep}

			out, err := command.RunPipelineWithTimeout(5*time.Millisecond, cmds)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("command timed out after 5ms"))
			Expect(out).To(BeEmpty())
		})
	})
})
