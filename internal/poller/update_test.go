package poller

import (
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
			want: `[user=[geotracked=false]]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.devices.LogValue().String())
		})
	}
}
