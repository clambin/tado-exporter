package scheduler

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrFailed_Is(t *testing.T) {
	err1 := &errFailed{err: errors.New("err1")}
	assert.True(t, errors.Is(err1, ErrFailed))
	assert.Equal(t, "job failed: err1", err1.Error())

	err2 := &errFailed{err: errors.New("err2")}
	assert.True(t, errors.Is(err2, ErrFailed))
	assert.Equal(t, "job failed: err2", err2.Error())

	err3 := &errFailed{}
	assert.True(t, errors.Is(err3, ErrFailed))
	assert.Equal(t, "job failed: unknown reason", err3.Error())

}

func TestErrFailed_Unwrap(t *testing.T) {
	err1 := &errFailed{err: errors.New("err1")}
	assert.True(t, errors.Is(err1, ErrFailed))

	err2 := fmt.Errorf("failed: %w", err1)
	assert.True(t, errors.Is(err2, ErrFailed))
	assert.Equal(t, "failed: job failed: err1", err2.Error())

	err3 := errors.Unwrap(err2)
	assert.True(t, errors.Is(err3, ErrFailed))
	assert.Equal(t, "job failed: err1", err3.Error())

	var err4 *errFailed
	ok := errors.As(err2, &err4)
	assert.True(t, ok)
	assert.True(t, errors.Is(err4, ErrFailed))
	assert.Equal(t, "job failed: err1", err4.Error())
}
