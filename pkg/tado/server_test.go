package tado_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// APIServer implements an authenticating API server
type APIServer struct {
	counter      int
	accessToken  string
	refreshToken string
	expires      time.Time
	failRefresh  bool
}

func (apiServer *APIServer) serve(req *http.Request) *http.Response {
	if req.URL.Path == "/oauth/token" {
		return apiServer.respondAuth(req)
	}

	if apiServer.validate(req) == false {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Status:     "Forbidden",
		}
	}

	if response, ok := responses[req.URL.Path]; ok {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString(response)),
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "API " + req.URL.Path + " not implemented",
	}
}

func getGrantType(body io.Reader) string {
	content, _ := ioutil.ReadAll(body)
	if params, err := url.ParseQuery(string(content)); err == nil {
		if tokenType, ok := params["grant_type"]; ok == true {
			return tokenType[0]
		}
	}
	panic("grant_type not found in body")
}

func (apiServer *APIServer) respondAuth(req *http.Request) *http.Response {
	const authResponse = `{
  "access_token":"%s",
  "token_type":"bearer",
  "refresh_token":"%s",
  "expires_in":%d,
  "scope":"home.user",
  "jti":"jti"
}`

	defer req.Body.Close()
	grantType := getGrantType(req.Body)

	if grantType == "refresh_token" {
		if apiServer.failRefresh {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "Forbidden: test server in failRefresh mode",
			}
		}
		apiServer.counter++
	} else {
		apiServer.counter = 1
	}

	apiServer.accessToken = fmt.Sprintf("token_%d", apiServer.counter)
	apiServer.refreshToken = apiServer.accessToken
	apiServer.expires = time.Now().Add(20 * time.Second)

	return &http.Response{
		StatusCode: http.StatusOK,
		Body: ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf(authResponse,
			apiServer.accessToken,
			apiServer.refreshToken,
			20,
		))),
	}
}

func (apiServer *APIServer) validate(req *http.Request) bool {
	if apiServer.accessToken == "" {
		return false
	}
	bearer := req.Header.Get("Authorization")
	if bearer != "Bearer "+apiServer.accessToken {
		return false
	}
	if time.Now().After(apiServer.expires) {
		return false
	}

	return true
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
  { 
    "id": 1, 
    "name": "Living room", 
    "devices": [ 
		{
		  "deviceType": "RU02",
		  "currentFwVersion": "67.2", 
		  "connectionState": { 
			"value": true 
		  }, 
		  "batteryState": "NORMAL" 
		}
    ]
  },
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
	"/api/v2/homes/242/zones/2/state": `{
  "setting": {
    "power": "ON",
    "temperature": { "celsius": 20.00 }
  },
  "openWindow": {
    "durationInSeconds": 50,
    "remainingTimeInSeconds": 250
  },
  "activityDataPoints": { "heatingPower": { "percentage": 11.00 } },
  "sensorDataPoints": {
    "insideTemperature": { "celsius": 19.94 },
    "humidity": { "percentage": 37.70 }
  }
}`,
	//type ZoneInfoOpenWindow struct {
	//	DetectedTime           time.Time `json:"detectedTime"`
	//	DurationInSeconds      int       `json:"durationInSeconds"`
	//	Expiry                 time.Time `json:"expiry"`
	//	RemainingTimeInSeconds int       `json:"remainingTimeInSeconds"`
	//}
	"/api/v2/homes/242/weather": `{
  "outsideTemperature": { "celsius": 3.4 },
  "solarIntensity": { "percentage": 13.3 },
  "weatherState": { "value": "CLOUDY_MOSTLY" }
}`,
	"/api/v2/homes/242/mobileDevices": `[{
	"id": 1,
	"name": "device 1",
	"settings": {
		"geoTrackingEnabled": true
	},
	"location": {
		"stale": false,
		"atHome": true
	}
}, {
	"id": 2,
	"name": "device 2",
	"settings": {
		"geoTrackingEnabled": true
	},
	"location": {
		"stale": false,
		"atHome": false
	}
}]`,
	// TODO: this doesn't test whether PUT/DELETE were used, nor validates the payload
	"/api/v2/homes/242/zones/2/overlay": `{}`,
}
