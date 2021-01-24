package controller_test

import (
	"errors"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayConfigWhite(t *testing.T) {
	rules, err := controller.ParseRules([]byte(`
autoAway:
  - zoneName: "foo"
    mobileDeviceName: "foo"
    waitTime: 1h
    targetTemperature: 5.0
  - zoneName: "bar"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 15.0
  - zoneName: "bar"
    mobileDeviceName: "not-a-phone"
    waitTime: 1h
    targetTemperature: 7.0
  - zoneName: "not-a-zone"
    mobileDeviceName: "foo"
    waitTime: 1h
    targetTemperature: 7.0
`))
	ctrlr := controller.Controller{
		API:          &mockAPI{},
		Rules:        rules,
		AutoAwayInfo: nil,
	}

	assert.Nil(t, err)
	assert.Nil(t, ctrlr.AutoAwayInfo)

	err = ctrlr.AutoAwayUpdateInfo()
	assert.Nil(t, err)
	assert.NotNil(t, ctrlr.AutoAwayInfo)
	assert.Len(t, ctrlr.AutoAwayInfo, 2)

	actions, err := ctrlr.AutoAwayGetActions()
	assert.Nil(t, err)
	assert.Len(t, actions, 0)

	// "foo" was previously away
	autoAway, _ := ctrlr.AutoAwayInfo[1]
	autoAway.Home = false
	ctrlr.AutoAwayInfo[1] = autoAway

	err = ctrlr.AutoAwayUpdateInfo()
	assert.Nil(t, err)
	actions, err = ctrlr.AutoAwayGetActions()
	assert.Nil(t, err)
	assert.Len(t, actions, 1)
	// "foo" now home, so we need to delete the overlay
	assert.False(t, actions[0].Overlay)
	assert.Equal(t, 1, actions[0].ZoneID)

	// "bar" was previously home
	autoAway, _ = ctrlr.AutoAwayInfo[2]
	autoAway.Since = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	ctrlr.AutoAwayInfo[2] = autoAway

	err = ctrlr.AutoAwayUpdateInfo()
	assert.Nil(t, err)
	actions, err = ctrlr.AutoAwayGetActions()
	assert.Nil(t, err)
	assert.Len(t, actions, 1)
	// "bar" has been away longer than WaitTime, so we need to set an overlay
	assert.True(t, actions[0].Overlay)
	assert.Equal(t, 2, actions[0].ZoneID)
	assert.Equal(t, 15.0, actions[0].TargetTemperature)
}

func TestAutoAwayConfigBlack(t *testing.T) {
	rules, err := controller.ParseRules([]byte(`
autoAway:
  - zoneName: "foo"
    mobileDeviceName: "foo"
    waitTime: 1h
    targetTemperature: 5.0
  - zoneName: "bar"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 15.0
`))
	server := mockAPI{}
	ctrlr := controller.Controller{
		API:          &server,
		Rules:        rules,
		AutoAwayInfo: nil,
	}

	assert.Nil(t, err)
	assert.Nil(t, ctrlr.AutoAwayInfo)

	err = ctrlr.Run()
	assert.Nil(t, err)
	assert.NotNil(t, ctrlr.AutoAwayInfo)
	assert.Len(t, ctrlr.AutoAwayInfo, 2)

	// "foo" was previously away
	autoAway, _ := ctrlr.AutoAwayInfo[1]
	autoAway.Home = false
	ctrlr.AutoAwayInfo[1] = autoAway

	err = ctrlr.Run()
	assert.Nil(t, err)
	assert.Len(t, server.Overlays, 0)

	// "bar" has been away for a long time
	autoAway, _ = ctrlr.AutoAwayInfo[2]
	autoAway.Since = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	ctrlr.AutoAwayInfo[2] = autoAway
	err = ctrlr.Run()
	assert.Nil(t, err)
	assert.Len(t, server.Overlays, 1)
	assert.Equal(t, 15.0, server.Overlays[2])
}

// Mock tado API Client
type mockAPI struct {
	Overlays map[int]float64
}

func (client *mockAPI) GetZones() ([]tado.Zone, error) {
	return []tado.Zone{
		{
			ID:   1,
			Name: "foo",
		},
		{
			ID:   2,
			Name: "bar",
		},
	}, nil
}

func (client *mockAPI) GetZoneInfo(zoneID int) (*tado.ZoneInfo, error) {
	info := tado.ZoneInfo{
		Setting: tado.ZoneInfoSetting{
			Power:       "ON",
			Temperature: tado.Temperature{Celsius: 20.0},
		},
		OpenWindow: tado.ZoneInfoOpenWindow{
			DurationInSeconds:      50,
			RemainingTimeInSeconds: 250,
		},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
			HeatingPower: tado.Percentage{Percentage: 11.0},
		},
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			Temperature: tado.Temperature{Celsius: 19.94},
			Humidity:    tado.Percentage{Percentage: 37.7},
		},
	}

	if zoneID != 1 {
		info.Setting.Temperature.Celsius = 25.0
		info.Overlay = tado.ZoneInfoOverlay{
			Type: "MANUAL",
			Setting: tado.ZoneInfoOverlaySetting{
				Type:        "HEATING",
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 25.0},
			},
		}
	}

	return &info, nil
}

func (client *mockAPI) GetWeatherInfo() (*tado.WeatherInfo, error) {
	return &tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 3.4},
		SolarIntensity:     tado.Percentage{Percentage: 13.3},
		WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
	}, nil
}

func (client *mockAPI) GetMobileDevices() ([]tado.MobileDevice, error) {
	return []tado.MobileDevice{
		{
			ID:       1,
			Name:     "foo",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: true},
		},
		{
			ID:       2,
			Name:     "bar",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: false},
		},
	}, nil
}

func (client *mockAPI) SetZoneManualTemperature(zoneID int, temperature float64) error {
	if client.Overlays == nil {
		client.Overlays = make(map[int]float64)
	}

	client.Overlays[zoneID] = temperature
	return nil
}

func (client *mockAPI) DeleteZoneManualTemperature(zoneID int) error {
	if client.Overlays == nil {
		client.Overlays = make(map[int]float64)
	}

	if _, ok := client.Overlays[zoneID]; ok == false {
		return errors.New("tried to delete non-existing overlay")
	}
	delete(client.Overlays, zoneID)
	return nil
}
