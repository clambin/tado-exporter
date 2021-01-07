package tado_test

import (
	"bytes"
	"github.com/clambin/gotools/httpstub"
	"io/ioutil"
	"net/http"
	"tado-exporter/internal/testtools"
	"tado-exporter/pkg/tado"
	"time"

	"github.com/stretchr/testify/assert"

	"testing"
)

func TestTypesToString(t *testing.T) {
	zoneInfo := tado.ZoneInfo{
		Setting: tado.ZoneInfoSetting{
			Power:       "ON",
			Temperature: tado.Temperature{Celsius: 25.0},
		},
		OpenWindow: "",
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			Temperature: tado.Temperature{Celsius: 21.0},
			Humidity:    tado.Percentage{Percentage: 30.0},
		},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
			HeatingPower: tado.Percentage{Percentage: 25.0},
		},
	}

	assert.Equal(t, "target=25.0 power=ON temp=21.0, humidity=30.0, heating=25.0", zoneInfo.String())

	weatherInfo := tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 27.0},
		SolarIntensity:     tado.Percentage{Percentage: 75.0},
		WeatherState:       tado.Value{Value: "SUNNY"},
	}

	assert.Equal(t, "temp=27.0 solar=75.0 weather=SUNNY", weatherInfo.String())
}

func TestAPIClient_Initialization(t *testing.T) {
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(APIServer),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	var err error
	err = client.Initialize()
	assert.Nil(t, err)
	assert.Equal(t, "access_token", client.AccessToken)
	assert.Equal(t, 242, client.HomeID)
}

func TestAPIClient_Authentication(t *testing.T) {
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(APIServer),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	var err error
	err = client.Initialize()
	assert.Nil(t, err)
	assert.Equal(t, "access_token", client.AccessToken)

	client.Expires = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	err = client.Initialize()
	assert.Nil(t, err)
	assert.Greater(t, client.Expires.Unix(), time.Now().Unix())
	assert.Equal(t, 242, client.HomeID)
}

func TestAPIClient_Zones(t *testing.T) {
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(APIServer),
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
	assert.Equal(t, "", tadoZoneInfo.OpenWindow)
	assert.Equal(t, 11.0, tadoZoneInfo.ActivityDataPoints.HeatingPower.Percentage)
	assert.Equal(t, 19.94, tadoZoneInfo.SensorDataPoints.Temperature.Celsius)
	assert.Equal(t, 37.7, tadoZoneInfo.SensorDataPoints.Humidity.Percentage)
}

func TestAPIClient_Weather(t *testing.T) {
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(testtools.APIServer),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	tadoWeatherInfo, err := client.GetWeatherInfo()
	assert.Nil(t, err)
	assert.Equal(t, 3.4, tadoWeatherInfo.OutsideTemperature.Celsius)
	assert.Equal(t, 13.3, tadoWeatherInfo.SolarIntensity.Percentage)
	assert.Equal(t, "CLOUDY_MOSTLY", tadoWeatherInfo.WeatherState.Value)

}

// loopback functions
func APIServer(req *http.Request) *http.Response {
	if response, ok := responses[req.URL.Path]; ok {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString(response)),
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "API" + req.URL.Path + " not implemented",
	}
}

var responses = map[string]string{
	"/oauth/token": `{
  "access_token":"access_token",
  "token_type":"bearer",
  "refresh_token":"refresh_token",
  "expires_in":599,
  "scope":"home.user",
  "jti":"jti"
}`,
	"/api/v1/me": `{
  "name":"Some User",
  "email":"user@example.com",
  "username":"user@example.com",
  "enabled":true,
  "id":"somelongidstring",
  "homeId":242,
  "locale":"en_BE",
  "type":"WEB_USER"
}`,
	"/api/v2/homes/242/zones": `[
  { "id": 1, "name": "Living room" },
  { "id": 2, "name": "Study" },
  { "id": 3, "name": "Bathroom" }
]`,
	"/api/v2/homes/242/zones/1/state": `{
  "setting": {
    "power": "ON",
    "temperature": { "celsius": 20.00 }
  },
  "openWindow": null,
  "activityDataPoints": { "heatingPower": { "percentage": 11.00 } },
  "sensorDataPoints": {
    "insideTemperature": { "celsius": 19.94 },
    "humidity": { "percentage": 37.70 }
  }
}`,
	"/api/v2/homes/242/weather": `{
  "outsideTemperature": { "celsius": 3.4 },
  "solarIntensity": { "percentage": 13.3 },
  "weatherState": { "value": "CLOUDY_MOSTLY" }
}`,
}
