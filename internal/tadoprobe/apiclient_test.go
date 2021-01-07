package tadoprobe_test

import (
	"bytes"
	"github.com/clambin/gotools/httpstub"
	"io/ioutil"
	"tado-exporter/internal/tadoprobe"

	"github.com/stretchr/testify/assert"

	"net/http"
	"testing"
)

func TestAPIClient_Initialization(t *testing.T) {
	client := tadoprobe.APIClient{
		HTTPClient: httpstub.NewTestClient(apiServer),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	var err error
	err = client.Authenticate()
	assert.Nil(t, err)
	assert.Equal(t, "access_token", client.AccessToken)

	err = client.GetHomeID()
	assert.Nil(t, err)
	assert.Equal(t, 242, client.HomeID)
}

func TestAPIClient_Get(t *testing.T) {
	client := tadoprobe.APIClient{
		HTTPClient: httpstub.NewTestClient(apiServer),
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

// loopback functions
func apiServer(req *http.Request) *http.Response {
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
    "temperature": {
      "celsius": 20.00
    }
  },
  "openWindow": null,
  "activityDataPoints": {
    "heatingPower": {
      "percentage": 11.00
    }
  },
  "sensorDataPoints": {
    "insideTemperature": {
      "celsius": 19.94
    },
    "humidity": {
      "percentage": 37.70
    }
  }
}`,
}
