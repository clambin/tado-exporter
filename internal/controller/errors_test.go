package controller

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErrLuaInvalidResponse_Error(t *testing.T) {
	err := &errLuaInvalidResponse{
		err: errors.New("test error"),
	}

	assert.Equal(t, "invalid lua response: test error", err.Error())

	assert.True(t, errors.Is(err, &errLuaInvalidResponse{}))
	var err2 *errLuaInvalidResponse
	require.True(t, errors.As(err, &err2))
	assert.Equal(t, "invalid lua response: test error", err2.Error())

	wrappedErr := fmt.Errorf("wrapped: %w", err)
	assert.True(t, errors.Is(wrappedErr, &errLuaInvalidResponse{}))
	require.True(t, errors.As(wrappedErr, &err2))
	assert.Equal(t, "invalid lua response: test error", err2.Error())

}
