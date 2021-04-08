package model_test

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/pkg/tado"
	"testing"
)

func TestZoneState_String(t *testing.T) {
	type fields struct {
		State       model.ZoneStateEnum
		Temperature tado.Temperature
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "unknown",
			fields: fields{State: model.Unknown},
			want:   "unknown",
		},
		{
			name:   "auto",
			fields: fields{State: model.Auto},
			want:   "auto",
		},
		{
			name:   "off",
			fields: fields{State: model.Off},
			want:   "off",
		},
		{
			name:   "manual",
			fields: fields{State: model.Manual, Temperature: tado.Temperature{Celsius: 18.0}},
			want:   "manual (18.0ºC)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := model.ZoneState{
				State:       tt.fields.State,
				Temperature: tt.fields.Temperature,
			}
			if got := state.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
