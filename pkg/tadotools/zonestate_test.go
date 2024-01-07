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

/*
func TestZoneState_LogValue(t *testing.T) {
	type fields struct {
		Overlay           tado.OverlayTerminationMode
		TargetTemperature tado.Temperature
	}
	tests := []struct {
		name   string
		fields fields
		wantState   string
	}{
		{
			name:   "no overlay",
			fields: fields{Overlay: tado.NoOverlay},
			wantState:   `level=INFO msg=state s.overlay="no overlay"`,
		},
		{
			name:   "off",
			fields: fields{Overlay: tado.PermanentOverlay},
			wantState:   `level=INFO msg=state s.overlay="permanent overlay" s.heating=false`,
		},
		{
			name:   "on",
			fields: fields{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 22.5}},
			wantState:   `level=INFO msg=state s.overlay="permanent overlay" s.heating=true s.target=22.5`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := rules.ZoneState{
				Overlay:           tt.fields.Overlay,
				TargetTemperature: tt.fields.TargetTemperature,
			}

			out := bytes.NewBufferString("")
			opt := slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				// Remove time from the output for predictable test output.
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			}}
			l := slog.New(slog.NewTextHandler(out, &opt))

			l.Log(context.Background(), slog.LevelInfo, "state", "s", s)
			assert.Equal(t, tt.wantState+"\n", out.String())
		})
	}
}
*/
