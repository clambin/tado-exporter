package rules

import (
	"github.com/clambin/tado"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestActions_GetNextState(t *testing.T) {
	tests := []struct {
		name   string
		input  Actions
		expect Action
	}{
		{
			name: "get earliest",
			input: Actions{
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "reason 1"},
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Minute, Reason: "reason 2"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 3"},
			},
			expect: Action{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Minute, Reason: "reason 2"},
		},
		{
			name: "prefer off",
			input: Actions{
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: time.Minute, Reason: "reason 1"},
				{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "reason 2"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 3"},
			},
			expect: Action{ZoneID: 1, ZoneName: "foo", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "reason 2"},
		},
		{
			name: "no action",
			input: Actions{
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 1"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 2"},
				{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 2"},
			},
			expect: Action{ZoneID: 1, ZoneName: "foo", Action: false, Reason: "reason 1, reason 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.input.GetNext())
		})
	}
}

func TestGetZoneState_Panic(t *testing.T) {
	assert.Panics(t, func() {
		a := Actions{}
		_ = a.GetNext()
	})
}
