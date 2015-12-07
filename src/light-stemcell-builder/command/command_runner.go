package command

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SelectLine returns a *exec.Cmd which uses sed to select an individual line
func SelectLine(n int) (*exec.Cmd, error) {
	if n < 1 {
		return nil, fmt.Errorf("line selection %d is not positive", n)
	}
	return exec.Command("sed", "-n", fmt.Sprintf("%d,%dp", n, n)), nil
}

// SelectFields returns a *exec.Cmd struct which uses awk to select a single field
func SelectField(n int) (*exec.Cmd, error) {
	if n < 1 {
		return nil, fmt.Errorf("field selection %d is not positive", n)
	}

	return exec.Command("awk", fmt.Sprintf("{print $%d}", n)), nil
}

// SelectFields returns a *exec.Cmd struct which uses awk to select multiple fields
func SelectFields(positions []int) (*exec.Cmd, error) {
	if len(positions) < 1 {
		return nil, fmt.Errorf("field selection %v is empty", positions)
	}

	stringPositions := []string{}
	for i := range positions {
		stringPositions = append(stringPositions, fmt.Sprintf("$%d", positions[i]))
	}

	queryString := strings.Join(stringPositions, ",")

	return exec.Command("awk", fmt.Sprintf("{print %s}", queryString)), nil
}

// TimeoutError is thrown when a RunWithTimeout times out
type TimeoutError struct {
	timeout time.Duration
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("command timed out after %s", e.timeout)
}

type result struct {
	out string
	err error
}

// Run chains an ordered collection of commands via standard out.
// Each command in the pipeline will have its standard error sent to STDERR
// Leading and trailing whitespace will be removed from output
func RunPipeline(procs []*exec.Cmd) (string, error) {
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
			return "", fmt.Errorf("running command %d of %d, %s: %s; standard error: %s", i+1, len(procs), procs[i].Path, err, stderr.String())
		}
	}
	return strings.Trim(out.String(), " \n\t"), nil
}

// RunWithTimeout makes sure that the given commands run within the given timeout, or returns an error,
// while keeping the underlying behavior from the Run() function.
func RunPipelineWithTimeout(timeout time.Duration, procs []*exec.Cmd) (string, error) {
	ch := make(chan result, 1)

	go func() {
		out, err := RunPipeline(procs)
		ch <- result{out, err}
	}()

	select {
	case res := <-ch:
		return res.out, res.err
	case <-time.After(timeout):
		return "", TimeoutError{timeout: timeout}
	}
}
