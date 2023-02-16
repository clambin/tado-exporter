package controller

import (
	"context"
	"github.com/clambin/tado"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/tado/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

var (
	zoneCfg []rules.ZoneConfig
)

func TestController_Run(t *testing.T) {
	a := mocks.NewAPI(t)
	prepareMockAPI(a)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	p := poller.New(a)
	wg.Add(1)
	go func() {
		p.Run(ctx, time.Minute)
		wg.Done()
	}()

	b := slackbot.NewSlackBot(t)
	b.On("Register", mock.AnythingOfType("string"), mock.AnythingOfType("slackbot.CommandFunc")).Return(nil)

	c := New(a, zoneCfg, b, p)
	assert.NotNil(t, c)

	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()

	response := c.cmds.DoRefresh(context.Background())
	assert.Len(t, response, 1)

	assert.Eventually(t, func() bool {
		response = c.cmds.ReportUsers(context.Background())
		return len(response) > 0
	}, time.Minute, 100*time.Millisecond)

	cancel()
	wg.Wait()
}

func prepareMockAPI(api *mocks.API) {
	api.
		On("GetMobileDevices", mock.Anything).
		Return([]tado.MobileDevice{
			{
				ID:       10,
				Name:     "foo",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{AtHome: true},
			},
			{
				ID:       11,
				Name:     "bar",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{AtHome: false},
			}}, nil)
	api.
		On("GetWeatherInfo", mock.Anything).
		Return(tado.WeatherInfo{
			OutsideTemperature: tado.Temperature{Celsius: 3.4},
			SolarIntensity:     tado.Percentage{Percentage: 13.3},
			WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
		}, nil)
	api.On("GetZones", mock.Anything).
		Return(tado.Zones{
			{ID: 1, Name: "foo"},
			{ID: 2, Name: "bar"},
		}, nil)
	api.
		On("GetZoneInfo", mock.Anything, 1).
		Return(tado.ZoneInfo{
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 18.5},
			},
		}, nil)
	api.
		On("GetZoneInfo", mock.Anything, 2).
		Return(tado.ZoneInfo{
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 15.0},
			},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZonePowerSetting{
					Type:        "HEATING",
					Power:       "OFF",
					Temperature: tado.Temperature{Celsius: 5.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}, nil)
}
