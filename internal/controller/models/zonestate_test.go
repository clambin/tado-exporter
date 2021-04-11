package models_test

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/pkg/tado"
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
