package poller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func prepareMockAPI(api *mocks.API) {
	api.
		On("GetMobileDevices", mock.Anything).
		Return([]tado.MobileDevice{
			{
				ID:       1,
				Name:     "foo",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{AtHome: true},
			},
			{
				ID:       2,
				Name:     "bar",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{AtHome: false},
			}}, nil).
		Once()
	api.
		On("GetWeatherInfo", mock.Anything).
		Return(tado.WeatherInfo{
			OutsideTemperature: tado.Temperature{Celsius: 3.4},
			SolarIntensity:     tado.Percentage{Percentage: 13.3},
			WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
		}, nil).
		Once()
	api.On("GetZones", mock.Anything).
		Return([]tado.Zone{
			{ID: 1, Name: "foo"},
			{ID: 2, Name: "bar"},
		}, nil).
		Once()
	api.
		On("GetZoneInfo", mock.Anything, 1).
		Return(tado.ZoneInfo{
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 18.5},
			},
		}, nil).
		Once()
	api.
		On("GetZoneInfo", mock.Anything, 2).
		Return(tado.ZoneInfo{
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 15.0},
			},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "OFF",
					Temperature: tado.Temperature{Celsius: 5.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}, nil).
		Once()
}

func TestPoller_Run(t *testing.T) {
	api := &mocks.API{}

	p := poller.New(api)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prepareMockAPI(api)

	go p.Run(ctx, 10*time.Millisecond)
	ch := make(chan *poller.Update)
	p.Register <- ch
	update := <-ch

	require.Len(t, update.UserInfo, 2)
	assert.Equal(t, "foo", update.UserInfo[1].Name)
	device := update.UserInfo[1]
	assert.Equal(t, tado.MobileDeviceLocationState(tado.DeviceHome), (&device).IsHome())
	assert.Equal(t, "bar", update.UserInfo[2].Name)
	device = update.UserInfo[2]
	assert.Equal(t, tado.MobileDeviceLocationState(tado.DeviceAway), (&device).IsHome())

	assert.Equal(t, "CLOUDY_MOSTLY", update.WeatherInfo.WeatherState.Value)
	assert.Equal(t, 3.4, update.WeatherInfo.OutsideTemperature.Celsius)
	assert.Equal(t, 13.3, update.WeatherInfo.SolarIntensity.Percentage)

	require.Len(t, update.Zones, 2)
	assert.Equal(t, "foo", update.Zones[1].Name)
	assert.Equal(t, "bar", update.Zones[2].Name)

	require.Len(t, update.ZoneInfo, 2)
	info := update.ZoneInfo[1]
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), (&info).GetState())
	info = update.ZoneInfo[2]
	assert.Equal(t, tado.ZoneState(tado.ZoneStateOff), (&info).GetState())

	api.AssertExpectations(t)
}

func TestServer_Poll(t *testing.T) {
	api := &mocks.API{}

	p := poller.New(api)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx, time.Minute)

	prepareMockAPI(api)

	ch := make(chan *poller.Update)
	p.Register <- ch
	update := <-ch

	require.Len(t, update.UserInfo, 2)
}

func TestServer_Refresh(t *testing.T) {
	api := &mocks.API{}

	p := poller.New(api)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx, time.Minute)

	prepareMockAPI(api)

	ch := make(chan *poller.Update)
	p.Register <- ch
	update := <-ch
	require.Len(t, update.UserInfo, 2)

	prepareMockAPI(api)
	p.Refresh()
	update = <-ch
	require.Len(t, update.UserInfo, 2)

}
