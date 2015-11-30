package pipeline

import (
	"bytes"
	"fmt"
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
func Run(procs []*exec.Cmd) (string, error) {
	stderr := &bytes.Buffer{}

	var err error
	lastIndex := len(procs) - 1

	for i := range procs[:lastIndex] {
		b := &bytes.Buffer{}

		procs[i].Stderr = stderr
		procs[i].Stdout = b
		procs[i+1].Stdin = b
	}

	out := &bytes.Buffer{}
	procs[lastIndex].Stderr = stderr
	procs[lastIndex].Stdout = out

	for i := range procs {
		err = procs[i].Run()
		if err != nil {
			return "", fmt.Errorf("running command %d of %d, %s: %s standard error: %s", i, len(procs), procs[i].Path, err, stderr.String())
		}
	}
	return strings.Trim(out.String(), " \n\t"), nil
}

// RunWithTimeout makes sure that the given commands run within the given timeout, or returns an error,
// while keeping the underlying behavior from the Run() function.
func RunWithTimeout(timeout time.Duration, procs []*exec.Cmd) (string, error) {
	ch := make(chan result, 1)

	go func() {
		out, err := Run(procs)
		ch <- result{out, err}
	}()

	select {
	case res := <-ch:
		return res.out, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("command timed out after %s", timeout)
	}
}
