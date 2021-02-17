package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayInfo(t *testing.T) {
	info := &AutoAwayInfo{
		MobileDevice: &tado.MobileDevice{
			ID:   1,
			Name: "foo",
			Settings: tado.MobileDeviceSettings{
				GeoTrackingEnabled: true,
			},
			Location: tado.MobileDeviceLocation{
				AtHome: true,
			},
		},
		ZoneID: 0,
		AutoAwayRule: &configuration.AutoAwayRule{
			MobileDeviceID:    1,
			MobileDeviceName:  "foo",
			WaitTime:          5 * time.Minute,
			ZoneID:            1,
			ZoneName:          "bar",
			TargetTemperature: 0,
		},
		state: autoAwayStateUndetermined,
	}

	assert.False(t, info.leftHome())

	info.MobileDevice.Location.AtHome = false
	assert.True(t, info.leftHome())

	info.state = autoAwayStateAway
	info.ActivationTime = time.Now().Add(5 * time.Minute)
	assert.False(t, info.shouldReport())

	info.ActivationTime = time.Now().Add(-10 * time.Minute)
	assert.True(t, info.shouldReport())

	info.state = autoAwayStateExpired
	assert.False(t, info.shouldReport())

	info.state = autoAwayStateReported
	assert.False(t, info.shouldReport())
	assert.True(t, info.isReported())
}
