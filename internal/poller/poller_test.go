package poller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestTadoPoller_Run(t *testing.T) {
	api := mocks.NewTadoGetter(t)

	p := poller.New(api, time.Minute, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())

	prepareMockAPI(api)

	ch := p.Register()
	errCh := make(chan error)
	go func() {
		errCh <- p.Run(ctx)
	}()
	p.Refresh()
	update := <-ch

	require.Len(t, update.UserInfo, 2)
	assert.Equal(t, "foo", update.UserInfo[1].Name)
	device := update.UserInfo[1]
	assert.Equal(t, tado.DeviceHome, (&device).IsHome())
	assert.Equal(t, "bar", update.UserInfo[2].Name)
	device = update.UserInfo[2]
	assert.Equal(t, tado.DeviceAway, (&device).IsHome())

	assert.Equal(t, "CLOUDY_MOSTLY", update.WeatherInfo.WeatherState.Value)
	assert.Equal(t, 3.4, update.WeatherInfo.OutsideTemperature.Celsius)
	assert.Equal(t, 13.3, update.WeatherInfo.SolarIntensity.Percentage)

	require.Len(t, update.Zones, 2)
	assert.Equal(t, "foo", update.Zones[1].Name)
	assert.Equal(t, "bar", update.Zones[2].Name)

	require.Len(t, update.ZoneInfo, 2)
	assert.True(t, update.Home)

	p.Unregister(ch)

	cancel()
	assert.NoError(t, <-errCh)
}

func prepareMockAPI(api *mocks.TadoGetter) {
	api.EXPECT().
		GetMobileDevices(mock.Anything).
		Return([]tado.MobileDevice{
			testutil.MakeMobileDevice(1, "foo", testutil.Home(true)),
			testutil.MakeMobileDevice(2, "bar", testutil.Home(false)),
		}, nil).
		Once()
	api.EXPECT().
		GetWeatherInfo(mock.Anything).
		Return(tado.WeatherInfo{
			OutsideTemperature: tado.Temperature{Celsius: 3.4},
			SolarIntensity:     tado.Percentage{Percentage: 13.3},
			WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
		}, nil).
		Once()
	api.EXPECT().
		GetZones(mock.Anything).
		Return(tado.Zones{
			{ID: 1, Name: "foo"},
			{ID: 2, Name: "bar"},
		}, nil).
		Once()
	api.EXPECT().
		GetZoneInfo(mock.Anything, 1).
		Return(testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18.5)), nil).
		Once()
	api.EXPECT().
		GetZoneInfo(mock.Anything, 2).
		Return(testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 5), testutil.ZoneInfoPermanentOverlay()), nil).
		Once()
	api.EXPECT().
		GetHomeState(mock.Anything).
		Return(tado.HomeState{Presence: "HOME"}, nil).
		Once()
}
