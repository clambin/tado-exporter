package testtools

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// TestCases contains expected outcome after using APIServer to get all zones info
var TestCases = []struct {
	Metric string
	Value  float64
}{
	{"tado_zone_target_temp_celsius", 20.0},
	{"tado_zone_power_state", 1.0},
	{"tado_temperature_celsius", 19.94},
	{"tado_heating_percentage", 11.0},
	{"tado_humidity_percentage", 37.7},
	{"tado_outside_temp_celsius", 3.4},
	{"tado_solar_intensity_percentage", 13.3},
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
