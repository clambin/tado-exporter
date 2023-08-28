package rules_test

import (
	"bytes"
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log/slog"
	"testing"
)

func TestGetZoneState(t *testing.T) {
	tests := []struct {
		name     string
		zoneInfo tado.ZoneInfo
		want     rules.ZoneState
	}{
		{
			name:     "off (auto)",
			zoneInfo: testutil.MakeZoneInfo(),
			want:     rules.ZoneState{Overlay: tado.NoOverlay},
		},
		{
			name:     "off (manual)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay()),
			want:     rules.ZoneState{Overlay: tado.PermanentOverlay},
		},
		{
			name:     "on (auto)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20)),
			want: rules.ZoneState{
				Overlay:           tado.NoOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
		},
		{
			name:     "on (manual)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20), testutil.ZoneInfoPermanentOverlay()),
			want: rules.ZoneState{
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
		},
		{
			name:     "on (timer)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20), testutil.ZoneInfoTimerOverlay()),
			want: rules.ZoneState{
				Overlay:           tado.TimerOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
		},
		{
			name:     "on (next block)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(20, 20), testutil.ZoneInfoNextTimeBlockOverlay()),
			want: rules.ZoneState{
				Overlay:           tado.NextBlockOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneState := rules.GetZoneState(tt.zoneInfo)
			assert.Equal(t, tt.want, zoneState)
		})
	}
}

func TestZoneState_Do(t *testing.T) {
	type fields struct {
		Overlay           tado.OverlayTerminationMode
		TargetTemperature tado.Temperature
	}
	type mockSetup func(m *mocks.TadoSetter)
	tests := []struct {
		name    string
		fields  fields
		setup   mockSetup
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "delete overlay",
			fields: fields{
				Overlay: tado.NoOverlay,
			},
			setup: func(m *mocks.TadoSetter) {
				m.EXPECT().DeleteZoneOverlay(mock.Anything, 10).Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "set overlay",
			fields: fields{
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			setup: func(m *mocks.TadoSetter) {
				m.EXPECT().SetZoneOverlay(mock.Anything, 10, 20.0).Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "off",
			fields: fields{
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 5.0},
			},
			setup: func(m *mocks.TadoSetter) {
				m.EXPECT().SetZoneOverlay(mock.Anything, 10, 5.0).Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "off",
			fields: fields{
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 5.0},
			},
			setup: func(m *mocks.TadoSetter) {
				m.EXPECT().SetZoneOverlay(mock.Anything, 10, 5.0).Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid overlay",
			fields: fields{
				Overlay:           tado.NextBlockOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			wantErr: assert.Error,
		},
	}

	ctx := context.Background()
	api := mocks.NewTadoSetter(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(api)
			}
			s := rules.ZoneState{
				Overlay:           tt.fields.Overlay,
				TargetTemperature: tt.fields.TargetTemperature,
			}
			tt.wantErr(t, s.Do(ctx, api, 10))
		})
	}
}

func TestZoneState_String(t *testing.T) {
	type fields struct {
		TargetTemperature tado.Temperature
		Overlay           tado.OverlayTerminationMode
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "off",
			fields: fields{
				Overlay: tado.PermanentOverlay,
			},
			want: "switching off heating",
		},
		{
			name: "off",
			fields: fields{
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 5.0},
			},
			want: "switching off heating",
		},
		{
			name: "auto",
			fields: fields{
				Overlay: tado.NoOverlay,
			},
			want: "moving to auto mode",
		},
		{
			name: "unknown",
			fields: fields{
				Overlay:           tado.TimerOverlay,
				TargetTemperature: tado.Temperature{Celsius: 10.0},
			},
			want: "unknown action",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := rules.ZoneState{
				Overlay:           tt.fields.Overlay,
				TargetTemperature: tt.fields.TargetTemperature,
			}
			assert.Equal(t, tt.want, s.String())
		})
	}
}

func TestZoneState_LogValue(t *testing.T) {
	type fields struct {
		Overlay           tado.OverlayTerminationMode
		TargetTemperature tado.Temperature
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "no overlay",
			fields: fields{Overlay: tado.NoOverlay},
			want:   `level=INFO msg=state s.overlay="no overlay"`,
		},
		{
			name:   "off",
			fields: fields{Overlay: tado.PermanentOverlay},
			want:   `level=INFO msg=state s.overlay="permanent overlay" s.heating=false`,
		},
		{
			name:   "on",
			fields: fields{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 22.5}},
			want:   `level=INFO msg=state s.overlay="permanent overlay" s.heating=true s.target=22.5`,
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
			assert.Equal(t, tt.want+"\n", out.String())
		})
	}
}
