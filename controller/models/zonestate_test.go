package models_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/models"
	"reflect"
	"testing"
)

func TestZoneState_String(t *testing.T) {
	type fields struct {
		State       models.ZoneStateEnum
		Temperature tado.Temperature
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "unknown",
			fields: fields{State: models.ZoneUnknown},
			want:   "unknown",
		},
		{
			name:   "auto",
			fields: fields{State: models.ZoneAuto},
			want:   "auto",
		},
		{
			name:   "off",
			fields: fields{State: models.ZoneOff},
			want:   "off",
		},
		{
			name:   "manual",
			fields: fields{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 18.0}},
			want:   "manual (18.0ºC)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := models.ZoneState{
				State:       tt.fields.State,
				Temperature: tt.fields.Temperature,
			}
			if got := state.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetZoneState(t *testing.T) {
	type args struct {
		zoneInfo tado.ZoneInfo
	}
	tests := []struct {
		name      string
		args      args
		wantState models.ZoneState
	}{
		{
			name:      "auto",
			args:      args{zoneInfo: tado.ZoneInfo{}},
			wantState: models.ZoneState{State: models.ZoneAuto},
		},
		{
			name: "manual",
			args: args{zoneInfo: tado.ZoneInfo{
				Overlay: tado.ZoneInfoOverlay{
					Type: "MANUAL",
					Setting: tado.ZoneInfoOverlaySetting{
						Type:        "HEATING",
						Temperature: tado.Temperature{Celsius: 15.0},
					},
					Termination: tado.ZoneInfoOverlayTermination{
						Type: "MANUAL",
					},
				},
			}},
			wantState: models.ZoneState{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 15.0}},
		},
		{
			name: "overlay with automatic termination",
			args: args{zoneInfo: tado.ZoneInfo{
				Overlay: tado.ZoneInfoOverlay{
					Type: "MANUAL",
					Setting: tado.ZoneInfoOverlaySetting{
						Type:        "HEATING",
						Temperature: tado.Temperature{Celsius: 15.0},
					},
					Termination: tado.ZoneInfoOverlayTermination{
						Type: "TIMER",
					},
				},
			}},
			wantState: models.ZoneState{State: models.ZoneAuto},
		},
		{
			name: "off",
			args: args{zoneInfo: tado.ZoneInfo{
				Overlay: tado.ZoneInfoOverlay{
					Type: "MANUAL",
					Setting: tado.ZoneInfoOverlaySetting{
						Type:        "HEATING",
						Temperature: tado.Temperature{Celsius: 5.0},
					},
					Termination: tado.ZoneInfoOverlayTermination{
						Type: "MANUAL",
					},
				},
			}},
			wantState: models.ZoneState{State: models.ZoneOff},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotState := models.GetZoneState(tt.args.zoneInfo); !reflect.DeepEqual(gotState, tt.wantState) {
				t.Errorf("GetZoneState() = %v, want %v", gotState, tt.wantState)
			}
		})
	}
}

func TestZoneState_Equals(t *testing.T) {
	type fields struct {
		State       models.ZoneStateEnum
		Temperature tado.Temperature
	}
	type args struct {
		b models.ZoneState
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "auto - same",
			fields: fields{State: models.ZoneAuto},
			args:   args{b: models.ZoneState{State: models.ZoneAuto}},
			want:   true,
		},
		{
			name:   "manual - same",
			fields: fields{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 15.0}},
			args:   args{b: models.ZoneState{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 15.0}}},
			want:   true,
		},
		{
			name:   "off - same",
			fields: fields{State: models.ZoneOff},
			args:   args{b: models.ZoneState{State: models.ZoneOff}},
			want:   true,
		},
		{
			name:   "auto - different",
			fields: fields{State: models.ZoneAuto},
			args:   args{b: models.ZoneState{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 15.0}}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := models.ZoneState{
				State:       tt.fields.State,
				Temperature: tt.fields.Temperature,
			}
			if got := a.Equals(tt.args.b); got != tt.want {
				t.Errorf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
