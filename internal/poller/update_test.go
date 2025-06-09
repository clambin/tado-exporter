package poller

import (
	"testing"

	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
)

func TestZone_GetTargetTemperature(t *testing.T) {
	zone := Zone{
		ZoneState: tado.ZoneState{
			Setting: &tado.ZoneSetting{
				Power:       oapi.VarP(tado.PowerON),
				Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(21))},
			},
		},
	}

	assert.Equal(t, 21, int(zone.GetTargetTemperature()))
	zone.ZoneState.Setting.Power = oapi.VarP(tado.PowerOFF)
	zone.ZoneState.Setting.Temperature = nil
	assert.Equal(t, 0, int(zone.GetTargetTemperature()))
}

func TestZone_GetZoneOverlayTerminationType(t *testing.T) {
	zone := Zone{
		ZoneState: tado.ZoneState{
			Overlay: &tado.ZoneOverlay{Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL)}},
		},
	}
	assert.Equal(t, tado.ZoneOverlayTerminationTypeMANUAL, zone.GetZoneOverlayTerminationType())
	zone.ZoneState.Overlay = nil
	assert.Equal(t, ZoneOverlayTerminationTypeNONE, zone.GetZoneOverlayTerminationType())
}

func TestMobileDevices_LogValue(t *testing.T) {
	tests := []struct {
		name    string
		devices MobileDevices
		want    string
	}{
		{
			name: "geotracked",
			devices: MobileDevices{{
				Id:       oapi.VarP(tado.MobileDeviceId(100)),
				Location: &tado.MobileDeviceLocation{AtHome: oapi.VarP(true)},
				Name:     oapi.VarP("user"),
				Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
			}},
			want: `[user=[geotracked=true home=true]]`,
		},
		{
			name: "not geotracked",
			devices: MobileDevices{{
				Id:       oapi.VarP(tado.MobileDeviceId(100)),
				Name:     oapi.VarP("user"),
				Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(false)},
			}},
			want: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.devices.LogValue().String())
		})
	}
}
