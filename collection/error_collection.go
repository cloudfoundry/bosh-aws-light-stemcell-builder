package collection

import (
	"fmt"
	"strings"
	"sync"
)

type Error struct {
	sync.Mutex
	errMsgs []string
}

func (e *Error) Add(err error) {
	e.Lock()
	defer e.Unlock()

	e.errMsgs = append(e.errMsgs, err.Error())
}

func (e *Error) Error() error {
	e.Lock()
	defer e.Unlock()

	if len(e.errMsgs) == 0 {
		return nil
	}

	return fmt.Errorf("encountered errors: \n %s", strings.Join(e.errMsgs, "\n"))
}
