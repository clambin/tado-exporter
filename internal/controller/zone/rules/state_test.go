package rules

import (
	"bytes"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState(t *testing.T) {
	testCases := []struct {
		name       string
		state      State
		wantString string
		wantLog    string
	}{
		{
			name: "auto mode",
			state: State{
				zoneID:   10,
				zoneName: "room",
				mode:     action.ZoneInAutoMode,
			},
			wantString: `moving to auto mode`,
			wantLog: `level=INFO msg=state state.type=zone state.name=room state.mode=auto
`,
		},
		{
			name: "overlay mode",
			state: State{
				zoneID:          10,
				zoneName:        "room",
				mode:            action.ZoneInOverlayMode,
				zoneTemperature: 18,
			},
			wantString: `heating to 18.0º`,
			wantLog: `level=INFO msg=state state.type=zone state.name=room state.mode=overlay state.temperature=18
`,
		},
		{
			name: "off",
			state: State{
				zoneID:          10,
				zoneName:        "room",
				mode:            action.ZoneInOverlayMode,
				zoneTemperature: 5,
			},
			wantString: `switching off heating`,
			wantLog: `level=INFO msg=state state.type=zone state.name=room state.mode=overlay state.temperature=5
`,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logOutput bytes.Buffer
			logger := testutil.NewBufferLogger(&logOutput)
			logger.Info("state", "state", tt.state)

			assert.Equal(t, tt.wantString, tt.state.String())
			assert.Equal(t, tt.wantLog, logOutput.String())
		})
	}
}

func TestState_IsEqual(t *testing.T) {
	t1 := State{
		zoneID:          10,
		zoneName:        "room",
		mode:            action.ZoneInOverlayMode,
		zoneTemperature: 18,
	}
	t2 := State{
		zoneID:   10,
		zoneName: "room",
		mode:     action.ZoneInAutoMode,
	}
	assert.True(t, t1.IsEqual(t1))
	assert.False(t, t1.IsEqual(t2))
}
