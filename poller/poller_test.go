package poller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPoller_Run(t *testing.T) {
	p := poller.New(&mockapi.MockAPI{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.API.SetZoneOverlay(ctx, 2, 5.0)
	if assert.Nil(t, err) == false {
		return
	}

	go p.Run(ctx, 10*time.Millisecond)

	ch := make(chan *poller.Update)
	p.Register <- ch

	update := <-ch

	if assert.Len(t, update.UserInfo, 2) {
		assert.Equal(t, "foo", update.UserInfo[1].Name)
		device := update.UserInfo[1]
		assert.Equal(t, tado.MobileDeviceLocationState(tado.DeviceHome), (&device).IsHome())
		assert.Equal(t, "bar", update.UserInfo[2].Name)
		device = update.UserInfo[2]
		assert.Equal(t, tado.MobileDeviceLocationState(tado.DeviceAway), (&device).IsHome())
	}

	assert.Equal(t, "CLOUDY_MOSTLY", update.WeatherInfo.WeatherState.Value)
	assert.Equal(t, 3.4, update.WeatherInfo.OutsideTemperature.Celsius)
	assert.Equal(t, 13.3, update.WeatherInfo.SolarIntensity.Percentage)

	if assert.Len(t, update.Zones, 2) {
		assert.Equal(t, "foo", update.Zones[1].Name)
		assert.Equal(t, "bar", update.Zones[2].Name)
	}

	if assert.Len(t, update.ZoneInfo, 2) {
		info := update.ZoneInfo[1]
		assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), (&info).GetState())
		info = update.ZoneInfo[2]
		assert.Equal(t, tado.ZoneState(tado.ZoneStateOff), (&info).GetState())
	}

	time.Sleep(100 * time.Millisecond)
}
