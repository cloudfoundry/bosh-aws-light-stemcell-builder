package threadsafe

import (
	"errors"
	"fmt"
	"sync"
)

// ErrorCollection is a thread-safe collection of errors that can returned
// later
type ErrorCollection struct {
	sync.Mutex
	errors map[string]error
}

// NewErrorCollection creates a new ErrorCollection
func NewErrorCollection() *ErrorCollection {
	return &ErrorCollection{errors: make(map[string]error)}
}

// AddError adds an error to this ErrorCollection. If an error with this label
// already exists, that error will be overwritten.
func (e *ErrorCollection) AddError(label string, err error) {
	e.Lock()
	defer e.Unlock()

	e.errors[label] = err
}

// BuildError returns an error if and only if there are errors in the collection.
func (e *ErrorCollection) BuildError() error {
	e.Lock()
	defer e.Unlock()
	if len(e.errors) > 0 {
		errString := "Received errors:"
		for label, err := range e.errors {
			errString = errString + "\n" + fmt.Sprintf("%s: %s", label, err.Error())
		}
		return errors.New(errString)
	}
	return nil
}
