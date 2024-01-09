package tadotools_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetZoneState(t *testing.T) {
	tests := []struct {
		name       string
		zoneInfo   tado.ZoneInfo
		wantState  tadotools.ZoneState
		wantString string
	}{
		{
			name:       "off (auto)",
			zoneInfo:   testutil.MakeZoneInfo(),
			wantState:  tadotools.ZoneState{Overlay: tado.NoOverlay},
			wantString: "off",
		},
		{
			name:       "off (manual)",
			zoneInfo:   testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay()),
			wantState:  tadotools.ZoneState{Overlay: tado.PermanentOverlay},
			wantString: "off",
		},
		{
			name:     "on (auto)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20)),
			wantState: tadotools.ZoneState{
				Overlay:           tado.NoOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			wantString: "target: 20.0",
		},
		{
			name:     "on (manual)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20), testutil.ZoneInfoPermanentOverlay()),
			wantState: tadotools.ZoneState{
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			wantString: "target: 20.0, MANUAL",
		},
		{
			name:     "on (timer)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20), testutil.ZoneInfoTimerOverlay()),
			wantState: tadotools.ZoneState{
				Overlay:           tado.TimerOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			wantString: "target: 20.0, MANUAL for 0s",
		},
		{
			name:     "on (next block)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20), testutil.ZoneInfoNextTimeBlockOverlay()),
			wantState: tadotools.ZoneState{
				Overlay:           tado.NextBlockOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			wantString: "target: 20.0, MANUAL for 0s",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			zoneState := tadotools.GetZoneState(tt.zoneInfo)
			assert.Equal(t, tt.wantState, zoneState)
			assert.Equal(t, tt.wantString, zoneState.String())
		})
	}
}

func TestZoneState_String_Unknown(t *testing.T) {
	s := tadotools.ZoneState{
		Overlay:           -1,
		TargetTemperature: tado.Temperature{Celsius: 18},
	}
	assert.Equal(t, "unknown", s.String())

}
