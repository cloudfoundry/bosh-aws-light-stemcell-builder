package pipeline

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type result struct {
	out string
	err error
}

// Run chains an ordered collection of commands via standard out.
// Each command in the pipeline will have its standard error sent to STDERR
// Leading and trailing whitespace will be removed from output
func Run(errStream io.Writer, procs ...*exec.Cmd) (string, error) {
	var err error
	lastIndex := len(procs) - 1

	for i := range procs[:lastIndex] {
		procs[i].Stderr = errStream
		procs[i+1].Stdin, err = procs[i].StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("opening pipe for command %d of %d, %s: %s", i, len(procs), procs[i].Path, err)
		}
	}

	out := &bytes.Buffer{}
	procs[lastIndex].Stderr = errStream
	procs[lastIndex].Stdout = out

	for i := range procs {
		err = procs[i].Start()
		if err != nil {
			return "", fmt.Errorf("starting command %d of %d, %s: %s", i, len(procs), procs[i].Path, err)
		}
	}

	for i := range procs {
		err = procs[i].Wait()
		if err != nil {
			return "", fmt.Errorf("running command %d of %d, %s: %s", i, len(procs), procs[i].Path, err)
		}
	}

	return strings.Trim(out.String(), " \n\t"), nil
}

// RunWithTimeout makes sure that the given commands run within the given timeout, or returns an error,
// while keeping the underlying behavior from the Run() function.
func RunWithTimeout(errStream io.Writer, timeout time.Duration, procs ...*exec.Cmd) (string, error) {
	ch := make(chan result, 1)

	go func() {
		out, err := Run(errStream, procs...)
		ch <- result{out, err}
	}()

	select {
	case res := <-ch:
		return res.out, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("command timed out after %s", timeout)
	}
}
