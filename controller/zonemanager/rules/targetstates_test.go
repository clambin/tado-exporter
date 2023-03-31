package rules

import (
	"github.com/clambin/tado"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTargetStates_GetNextState(t *testing.T) {
	tests := []struct {
		name   string
		input  TargetStates
		expect TargetState
	}{
		{
			name: "get earliest",
			input: TargetStates{
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Heating: true, Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "reason 1"},
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Heating: true, Overlay: tado.PermanentOverlay}, Delay: time.Minute, Reason: "reason 2"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 3"},
			},
			expect: TargetState{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Heating: true, Overlay: tado.PermanentOverlay}, Delay: time.Minute, Reason: "reason 2"},
		},
		{
			name: "prefer off",
			input: TargetStates{
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Heating: true, Overlay: tado.NoOverlay}, Delay: time.Minute, Reason: "reason 1"},
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Heating: false, Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "reason 2"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 3"},
			},
			expect: TargetState{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Heating: false, Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "reason 2"},
		},
		{
			name: "no action",
			input: TargetStates{
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 1"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 2"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 2"},
			},
			expect: TargetState{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 1, reason 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.input.GetNextState())
		})
	}
}
