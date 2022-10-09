package scheduler

import (
	"errors"
)

var (
	ErrCanceled = errors.New("job canceled")
	ErrFailed   = &errFailed{}
)

type errFailed struct {
	err error
}

func (e *errFailed) Error() string {
	reason := "unknown reason"
	if e.err != nil {
		reason = e.err.Error()
	}
	return "job failed: " + reason
}

func (e *errFailed) Is(err error) bool {
	return err == ErrFailed
}

func (e *errFailed) Unwrap() error {
	return e.err
}
