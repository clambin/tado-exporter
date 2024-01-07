package rules

import (
	"bytes"
	"github.com/clambin/tado-exporter/internal/controller/internal/testutil"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState(t *testing.T) {
	s := State{mode: action.HomeInHomeMode}

	var out bytes.Buffer
	l := testutil.NewBufferLogger(&out)

	l.Info("test", "state", s)
	assert.Equal(t, "level=INFO msg=test state.type=home state.mode=home\n", out.String())

	assert.Equal(t, `setting home to home mode`, s.String())
	assert.True(t, s.IsEqual(State{mode: action.HomeInHomeMode}))
	assert.False(t, s.IsEqual(State{mode: action.HomeInAwayMode}))
	assert.Equal(t, action.HomeInHomeMode, s.Mode())
}
