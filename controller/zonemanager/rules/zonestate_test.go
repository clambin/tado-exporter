package rules_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			zoneInfo: tado.ZoneInfo{Setting: tado.ZonePowerSetting{Power: "OFF"}},
			want: rules.ZoneState{
				Heating: false,
				Overlay: tado.NoOverlay,
			},
		},
		{
			name: "off (manual)",
			zoneInfo: tado.ZoneInfo{
				Setting: tado.ZonePowerSetting{Power: "OFF"},
				Overlay: tado.ZoneInfoOverlay{
					Type:        "HEATING",
					Setting:     tado.ZonePowerSetting{Power: "OFF"},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				},
			},
			want: rules.ZoneState{
				Heating: false,
				Overlay: tado.PermanentOverlay,
			},
		},
		{
			name:     "on (auto)",
			zoneInfo: tado.ZoneInfo{Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}}},
			want: rules.ZoneState{
				Heating:           true,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
				Overlay:           tado.NoOverlay,
			},
		},
		{
			name: "on (manual)",
			zoneInfo: tado.ZoneInfo{
				Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
				Overlay: tado.ZoneInfoOverlay{
					Type:        "HEATING",
					Setting:     tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				},
			},
			want: rules.ZoneState{
				Heating:           true,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
				Overlay:           tado.PermanentOverlay,
			},
		},
		{
			name: "on (timer)",
			zoneInfo: tado.ZoneInfo{
				Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
				Overlay: tado.ZoneInfoOverlay{
					Type:        "HEATING",
					Setting:     tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "TIMER", TypeSkillBasedApp: "TIMER"},
				},
			},
			want: rules.ZoneState{
				Heating:           true,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
				Overlay:           tado.TimerOverlay,
			},
		},
		{
			name: "on (next block)",
			zoneInfo: tado.ZoneInfo{
				Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
				Overlay: tado.ZoneInfoOverlay{
					Type:        "HEATING",
					Setting:     tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "TIMER", TypeSkillBasedApp: "NEXT_TIME_BLOCK"},
				},
			},
			want: rules.ZoneState{
				Heating:           true,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
				Overlay:           tado.NextBlockOverlay,
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
		Heating           bool
		TargetTemperature tado.Temperature
		Overlay           tado.OverlayTerminationMode
	}
	type mockArgs struct {
		on      string
		args    []any
		returns []any
	}
	tests := []struct {
		name    string
		fields  fields
		args    mockArgs
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "delete overlay",
			fields: fields{
				Overlay: tado.NoOverlay,
			},
			args: mockArgs{
				on:      "DeleteZoneOverlay",
				args:    []any{mock.AnythingOfType("*context.emptyCtx"), 10},
				returns: []any{nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "set overlay",
			fields: fields{
				Heating:           true,
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			args: mockArgs{
				on:      "SetZoneOverlay",
				args:    []any{mock.AnythingOfType("*context.emptyCtx"), 10, 20.0},
				returns: []any{nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "off",
			fields: fields{
				Heating:           false,
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 20.0},
			},
			args: mockArgs{
				on:      "SetZoneOverlay",
				args:    []any{mock.AnythingOfType("*context.emptyCtx"), 10, 5.0},
				returns: []any{nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "off",
			fields: fields{
				Heating:           true,
				Overlay:           tado.PermanentOverlay,
				TargetTemperature: tado.Temperature{Celsius: 5.0},
			},
			args: mockArgs{
				on:      "SetZoneOverlay",
				args:    []any{mock.AnythingOfType("*context.emptyCtx"), 10, 5.0},
				returns: []any{nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid overlay",
			fields: fields{
				Heating:           true,
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
			if tt.args.on != "" {
				api.On(tt.args.on, tt.args.args...).Return(tt.args.returns...).Once()
			}
			s := rules.ZoneState{
				Heating:           tt.fields.Heating,
				TargetTemperature: tt.fields.TargetTemperature,
				Overlay:           tt.fields.Overlay,
			}
			tt.wantErr(t, s.Do(ctx, api, 10))
		})
	}
}

func TestZoneState_String(t *testing.T) {
	type fields struct {
		Heating           bool
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
				Heating: false,
			},
			want: "switching off heating",
		},
		{
			name: "off",
			fields: fields{
				Overlay:           tado.PermanentOverlay,
				Heating:           true,
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
				Heating:           true,
				TargetTemperature: tado.Temperature{Celsius: 10.0},
			},
			want: "unknown action",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := rules.ZoneState{
				Heating:           tt.fields.Heating,
				TargetTemperature: tt.fields.TargetTemperature,
				Overlay:           tt.fields.Overlay,
			}
			assert.Equal(t, tt.want, s.String())
		})
	}
}
