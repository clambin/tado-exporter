package rules

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

	assert.Equal(t, "lua: invalid response: test error", err.Error())
	wrappedErr := fmt.Errorf("wrapped: %w", err)
	assert.ErrorIs(t, wrappedErr, &errLuaInvalidResponse{})
	var err2 *errLuaInvalidResponse
	require.ErrorAs(t, wrappedErr, &err2)
	assert.Equal(t, "lua: invalid response: test error", err2.Error())
}

func TestErrLua(t *testing.T) {
	err := &errLua{
		err: errors.New("test error"),
	}

	assert.Equal(t, "lua: test error", err.Error())
	wrappedErr := fmt.Errorf("wrapped: %w", err)
	assert.ErrorIs(t, wrappedErr, &errLua{})
	var err2 *errLua
	require.ErrorAs(t, wrappedErr, &err2)
	assert.Equal(t, "lua: test error", err2.Error())
}
