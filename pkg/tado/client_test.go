package tado_test

import (
	"github.com/clambin/gotools/httpstub"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestTypesToString(t *testing.T) {
	zoneInfo := tado.ZoneInfo{
		Setting: tado.ZoneInfoSetting{
			Power:       "ON",
			Temperature: tado.Temperature{Celsius: 25.0},
		},
		OpenWindow: tado.ZoneInfoOpenWindow{
			DurationInSeconds:      900,
			RemainingTimeInSeconds: 250,
		},
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			Temperature: tado.Temperature{Celsius: 21.0},
			Humidity:    tado.Percentage{Percentage: 30.0},
		},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
			HeatingPower: tado.Percentage{Percentage: 25.0},
		},
	}

	assert.Equal(t, `target=25.0ºC, temp=21.0ºC, humidity=30.0%, heating=25.0%, power=ON, openwindow=650s`, zoneInfo.String())

	weatherInfo := tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 27.0},
		SolarIntensity:     tado.Percentage{Percentage: 75.0},
		WeatherState:       tado.Value{Value: "SUNNY"},
	}

	assert.Equal(t, `temp=27.0ºC, solar=75.0%, weather=SUNNY`, weatherInfo.String())

	zone := tado.Zone{
		ID:   1,
		Name: "Living room",
		Devices: []tado.Device{
			{
				DeviceType:      "RU02",
				Firmware:        "67.2",
				ConnectionState: tado.ConnectionState{Value: true},
				BatteryState:    "LOW",
			},
		},
	}

	assert.Equal(t, "id=1 name=Living room devices={type=RU02 firmware=67.2 connection=true battery=LOW}", zone.String())

	mobileDevice := tado.MobileDevice{
		ID:   1,
		Name: "phone",
		Settings: tado.MobileDeviceSettings{
			GeoTrackingEnabled: true,
		},
		Location: tado.MobileDeviceLocation{
			Stale:  false,
			AtHome: true,
		},
	}

	assert.Equal(t, `name=phone, geotrack=true, stale=false, athome=true`, mobileDevice.String())
}

func TestAPIClient_Zones(t *testing.T) {
	server := APIServer{}
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(server.serve),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	tadoZones, err := client.GetZones()
	assert.Nil(t, err)
	assert.Len(t, tadoZones, 3)
	assert.Equal(t, "Living room", tadoZones[0].Name)
	assert.Equal(t, "Study", tadoZones[1].Name)
	assert.Equal(t, "Bathroom", tadoZones[2].Name)

	tadoZoneInfo, err := client.GetZoneInfo(tadoZones[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, 20.0, tadoZoneInfo.Setting.Temperature.Celsius)
	assert.Equal(t, "ON", tadoZoneInfo.Setting.Power)
	assert.Equal(t, 11.0, tadoZoneInfo.ActivityDataPoints.HeatingPower.Percentage)
	assert.Equal(t, 19.94, tadoZoneInfo.SensorDataPoints.Temperature.Celsius)
	assert.Equal(t, 37.7, tadoZoneInfo.SensorDataPoints.Humidity.Percentage)

	tadoZoneInfo, err = client.GetZoneInfo(tadoZones[1].ID)
	assert.Nil(t, err)
	assert.Equal(t, 50, tadoZoneInfo.OpenWindow.DurationInSeconds)
	assert.Equal(t, 250, tadoZoneInfo.OpenWindow.RemainingTimeInSeconds)

}

func TestAPIClient_Weather(t *testing.T) {
	server := &APIServer{}
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(server.serve),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	tadoWeatherInfo, err := client.GetWeatherInfo()
	assert.Nil(t, err)
	assert.Equal(t, 3.4, tadoWeatherInfo.OutsideTemperature.Celsius)
	assert.Equal(t, 13.3, tadoWeatherInfo.SolarIntensity.Percentage)
	assert.Equal(t, "CLOUDY_MOSTLY", tadoWeatherInfo.WeatherState.Value)

}

func TestAPIClient_Devices(t *testing.T) {
	server := &APIServer{}
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(server.serve),
		Username:   "user@example.com",
		Password:   "some-password",
	}

	zones, err := client.GetZones()
	assert.Nil(t, err)
	assert.Equal(t, "Living room", zones[0].Name)
	assert.Len(t, zones[0].Devices, 1)
	assert.Equal(t, true, zones[0].Devices[0].ConnectionState.Value)
	assert.Equal(t, "NORMAL", zones[0].Devices[0].BatteryState)
}

func TestAPIClient_MobileDevices(t *testing.T) {
	server := &APIServer{}
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(server.serve),
		Username:   "user@example.com",
		Password:   "some-password",
	}

	mobileDevices, err := client.GetMobileDevices()
	assert.Nil(t, err)
	assert.Len(t, mobileDevices, 2)
	assert.Equal(t, "device 1", mobileDevices[0].Name)
	assert.True(t, mobileDevices[0].Settings.GeoTrackingEnabled)
	assert.True(t, mobileDevices[0].Location.AtHome)
	assert.Equal(t, "device 2", mobileDevices[1].Name)
	assert.True(t, mobileDevices[1].Settings.GeoTrackingEnabled)
	assert.False(t, mobileDevices[1].Location.AtHome)
}
