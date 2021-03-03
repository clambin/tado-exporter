package autoaway

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayInfo(t *testing.T) {
	info := &DeviceInfo{
		mobileDevice: tado.MobileDevice{
			ID:   1,
			Name: "foo",
			Settings: tado.MobileDeviceSettings{
				GeoTrackingEnabled: true,
			},
			Location: tado.MobileDeviceLocation{
				AtHome: true,
			},
		},
		rule: configuration.AutoAwayRule{
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

	info.mobileDevice.Location.AtHome = false
	assert.True(t, info.leftHome())

	info.state = autoAwayStateAway
	info.activationTime = time.Now().Add(5 * time.Minute)
	assert.False(t, info.shouldReport())

	info.activationTime = time.Now().Add(-10 * time.Minute)
	assert.True(t, info.shouldReport())

	info.state = autoAwayStateExpired
	assert.False(t, info.shouldReport())

	info.state = autoAwayStateReported
	assert.False(t, info.shouldReport())
	assert.True(t, info.isReported())
}
