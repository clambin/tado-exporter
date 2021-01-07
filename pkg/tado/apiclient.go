package tado

import (
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// APIClient structure representing a Tado API client.
//
// Basic example to create a Tado API client:
//     client := tado.APIClient{
//        HTTPClient: &http.Client{},
//        Username: "your-tado-username",
//        Password: "your-tado-password",
//     }
type APIClient struct {
	HTTPClient   *http.Client
	Username     string
	Password     string
	Secret       string
	AccessToken  string
	Expires      time.Time
	RefreshToken string
	HomeID       int
}

// Initialize sets up the client to call the various APIs, i.e. authenticates with tado.com,
// retrieving/updating the Access Token required for the API functions, and retrieving the
// user's Home ID.
//
// Each API function calls this before invoking the API, so normally this doesn't need to be
// called by the calling application.
func (client *APIClient) Initialize() error {
	var err error
	if err = client.authenticate(); err == nil {
		err = client.getHomeID()
	}
	return err
}

// authenticate logs in to tado.com and gets an Access Token to invoke the API functions.
// Once logged in, authenticate renews the Access Token if it's expired since the last call.
//
// Invoked by each API function, so doesn't need to be called by the calling application.
func (client *APIClient) authenticate() error {
	var err error
	if client.AccessToken == "" {
		if client.Secret == "" {
			client.Secret = "wZaRN7rpjn3FoNyF5IFuxg9uMzYJcvOoQ8QWiIqS3hfk6gLhVlG57j5YNoZL2Rtc"
		}
		err = client.doAuthentication("password", client.Password)
	} else if time.Now().After(client.Expires) {
		err = client.doAuthentication("refresh_token", client.RefreshToken)
	}
	return err
}

func (client *APIClient) doAuthentication(grantType, credential string) error {
	var (
		err  error
		resp *http.Response
	)
	form := url.Values{}
	form.Add("client_id", "tado-web-app")
	form.Add("client_secret", client.Secret)
	form.Add("grant_type", grantType)
	form.Add(grantType, credential)
	form.Add("scope", "home.user")
	if grantType == "password" {
		form.Add("username", client.Username)
	}

	req, _ := http.NewRequest("POST", "https://auth.tado.com/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Add("Referer", "https://my.tado.com/")
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	if resp, err = client.HTTPClient.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := ioutil.ReadAll(resp.Body)

			var resp interface{}
			if err = json.Unmarshal(body, &resp); err == nil {
				m := resp.(map[string]interface{})
				client.AccessToken = m["access_token"].(string)
				client.RefreshToken = m["refresh_token"].(string)
				client.Expires = time.Now().Add(time.Second * time.Duration(m["expires_in"].(float64)))
			}
		} else {
			err = errors.New(resp.Status)
		}
	}
	log.WithFields(log.Fields{"err": err, "expires": client.Expires}).Debug("authentication")

	return err
}

// getHomeID gets the user's Home ID, used by the GetZones API
//
// Called by Initialize, so doesn't need to be called by the calling application.
func (client *APIClient) getHomeID() error {
	if client.HomeID > 0 {
		return nil
	}

	var (
		err  error
		resp *http.Response
	)
	req, _ := http.NewRequest("GET", "https://my.tado.com/api/v1/me", nil)
	req.Header.Add("Authorization", "Bearer "+client.AccessToken)

	if resp, err = client.HTTPClient.Do(req); err == nil {
		body, _ := ioutil.ReadAll(resp.Body)

		var resp interface{}
		if err = json.Unmarshal(body, &resp); err == nil {
			m := resp.(map[string]interface{})
			client.HomeID = int(m["homeId"].(float64))
		}
	}
	return err
}

// GetZones retrieves the different Zones configured for the user's Home ID.
//
func (client *APIClient) GetZones() ([]Zone, error) {
	var (
		err  error
		resp *http.Response
	)
	tadoZones := make([]Zone, 0)

	if err = client.Initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/zones"
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Add("Authorization", "Bearer "+client.AccessToken)

		if resp, err = client.HTTPClient.Do(req); err == nil {
			if resp.StatusCode == http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				err = json.Unmarshal(body, &tadoZones)
			}
		}
	}
	return tadoZones, err
}

// GetZoneInfo gets the info for the specified Zone
//
// Retrieved information:
// - Setting.Power:                              power state of the specified zone (0-1)
// - Temperature.Celsius:                        target temperature for the zone, in degrees Celsius
// - OpenWindow:                                 TBD
// - ActivityDataPoints.HeatingPower.Percentage: heating power for the zone (0-100%)
// - SensorDataPoints.Temperature.Celsius:       current temperature, in degrees Celsius
// - SensorDataPoints.Humidity.Percentage:       humidity (0-100%)
func (client *APIClient) GetZoneInfo(zoneID int) (*ZoneInfo, error) {
	var (
		err          error
		resp         *http.Response
		tadoZoneInfo ZoneInfo
	)
	if err = client.Initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/zones/" + strconv.Itoa(zoneID) + "/state"
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Add("Authorization", "Bearer "+client.AccessToken)

		if resp, err = client.HTTPClient.Do(req); err == nil {
			if resp.StatusCode == http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				err = json.Unmarshal(body, &tadoZoneInfo)
			} else {
				err = errors.New(resp.Status)
			}
		}
	}
	return &tadoZoneInfo, err
}

// GetWeatherInfo retrieves weather information for the user's Home.
//
// Retrieved information:
// - OutsideTemperature.Celsius:  outside temperate, in degrees Celsius
// - SolarIntensity.Percentage:   solar intensity (0-100%)
// - WeatherState.Value:          string describing current weather (list TBD)
func (client *APIClient) GetWeatherInfo() (*WeatherInfo, error) {
	var (
		err             error
		resp            *http.Response
		tadoWeatherInfo WeatherInfo
	)
	if err = client.Initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/weather"
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Add("Authorization", "Bearer "+client.AccessToken)

		if resp, err = client.HTTPClient.Do(req); err == nil {
			if resp.StatusCode == http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				err = json.Unmarshal(body, &tadoWeatherInfo)
			} else {
				err = errors.New(resp.Status)
			}
		}
	}
	return &tadoWeatherInfo, err

}
