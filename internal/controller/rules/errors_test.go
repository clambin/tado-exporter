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
	var err2 *errLuaInvalidResponse
	assert.True(t, errors.Is(wrappedErr, &errLuaInvalidResponse{}))
	require.True(t, errors.As(wrappedErr, &err2))
	assert.Equal(t, "lua: invalid response: test error", err2.Error())
}

func TestErrLua(t *testing.T) {
	err := &errLua{
		err: errors.New("test error"),
	}

	assert.Equal(t, "lua: test error", err.Error())
	wrappedErr := fmt.Errorf("wrapped: %w", err)
	var err2 *errLua
	assert.True(t, errors.Is(wrappedErr, &errLua{}))
	require.True(t, errors.As(wrappedErr, &err2))
	assert.Equal(t, "lua: test error", err2.Error())
}
