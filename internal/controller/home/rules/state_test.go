package rules

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState(t *testing.T) {
	homeId := tado.HomeId(1)
	s := State{mode: action.HomeInHomeMode, homeId: homeId}

	assert.Equal(t, `setting home to home mode`, s.String())
	assert.True(t, s.IsEqual(State{mode: action.HomeInHomeMode}))
	assert.False(t, s.IsEqual(State{mode: action.HomeInAwayMode}))
	assert.Equal(t, action.HomeInHomeMode, s.Mode())

	assert.Equal(t, `[type=home mode=home]`, s.LogValue().String())
}

func TestState_Do(t *testing.T) {
	tests := []struct {
		name               string
		mode               action.Mode
		expectHomePresence tado.HomePresence
		expectErr          assert.ErrorAssertionFunc
	}{
		{
			name:               "move to home mode",
			mode:               action.HomeInHomeMode,
			expectHomePresence: tado.HOME,
			expectErr:          assert.NoError,
		},
		{
			name:               "move to away mode",
			mode:               action.HomeInAwayMode,
			expectHomePresence: tado.AWAY,
			expectErr:          assert.NoError,
		},
		{
			name:      "no action results in error",
			mode:      action.NoAction,
			expectErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeId := tado.HomeId(1)
			s := State{mode: tt.mode, homeId: homeId}

			ctx := context.Background()
			c := mocks.NewTadoClient(t)
			if tt.expectHomePresence != "" {
				c.EXPECT().
					SetPresenceLockWithResponse(ctx, homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &tt.expectHomePresence}).
					Return(nil, nil).
					Once()
			}

			tt.expectErr(t, s.Do(ctx, c))
		})
	}
}
