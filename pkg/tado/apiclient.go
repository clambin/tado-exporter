// Package tado provides an API Client for the Tadoº smart thermostat devices
//
// Using this package typically involves creating an APIClient as follows:
//
//     client := tado.APIClient{
//        HTTPClient: &http.Client{},
//        Username: "your-tado-username",
//        Password: "your-tado-password",
//     }
//
// Once a client has been created, you can query tado.com for information about your different Tado devices.
// Currently the following three APIs are supported:
//
//   GetZones:       get the different zones (rooms) defined in your home
//   GetZoneInfo:    get metrics for a specified zone in your home
//   GetWeatherInfo: get overall weather information
//
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

// APIClient represents a Tado API client.
//
// Basic example to create a Tado API client:
//     client := tado.APIClient{
//        HTTPClient: &http.Client{},
//        Username: "your-tado-username",
//        Password: "your-tado-password",
//     }
//
// If the default Client Secret does not work, you can provide your own secret:
//     client := tado.APIClient{
//        HTTPClient:    &http.Client{},
//        Username:     "your-tado-username",
//        Password:     "your-tado-password",
//        ClientSecret: "your-client-secret",
//     }
//
// where your-client-secret can be found by visiting https://my.tado.com/webapp/env.js after logging in to my.tado.com
type APIClient struct {
	HTTPClient   *http.Client
	Username     string
	Password     string
	ClientSecret string
	AccessToken  string
	Expires      time.Time
	RefreshToken string
	HomeID       int
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
		body []byte
	)

	apiURL := "https://my.tado.com/api/v1/me"
	if body, err = client.call(apiURL); err == nil {
		var resp interface{}
		if err = json.Unmarshal(body, &resp); err == nil {
			m := resp.(map[string]interface{})
			client.HomeID = int(m["homeId"].(float64))
		}
	}
	return err
}

// GetZones retrieves the different Zones configured for the user's Home ID
func (client *APIClient) GetZones() ([]Zone, error) {
	var (
		err  error
		body []byte
	)
	zones := make([]Zone, 0)

	if err = client.initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/zones"
		if body, err = client.call(apiURL); err == nil {
			err = json.Unmarshal(body, &zones)
		}
	}

	for _, zone := range zones {
		log.WithFields(log.Fields{"err": err, "zone": zone}).Debug("GetZones")
	}
	return zones, err
}

// GetZoneInfo gets the info for the specified Zone
func (client *APIClient) GetZoneInfo(zoneID int) (*ZoneInfo, error) {
	var (
		err          error
		body         []byte
		tadoZoneInfo ZoneInfo
	)
	if err = client.initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/zones/" + strconv.Itoa(zoneID) + "/state"
		if body, err = client.call(apiURL); err == nil {
			err = json.Unmarshal(body, &tadoZoneInfo)
		}
	}
	return &tadoZoneInfo, err
}

// GetWeatherInfo retrieves weather information for the user's Home.
func (client *APIClient) GetWeatherInfo() (*WeatherInfo, error) {
	var (
		err             error
		tadoWeatherInfo WeatherInfo
		body            []byte
	)
	if err = client.initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/weather"
		if body, err = client.call(apiURL); err == nil {
			err = json.Unmarshal(body, &tadoWeatherInfo)
		}
	}
	return &tadoWeatherInfo, err

}

// GetMobileDevices retrieves the status of all registered mobile devices.
func (client *APIClient) GetMobileDevices() ([]MobileDevice, error) {
	var (
		err               error
		tadoMobileDevices []MobileDevice
		body              []byte
	)
	if err = client.initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/mobileDevices"
		if body, err = client.call(apiURL); err == nil {
			err = json.Unmarshal(body, &tadoMobileDevices)
		}
	}

	return tadoMobileDevices, err
}

// non-exported functions

// Initialize sets up the client to call the various APIs, i.e. authenticates with tado.com,
// retrieving/updating the Access Token required for the API functions, and retrieving the
// user's Home ID.
//
// Each API function calls this before invoking the API, so normally this doesn't need to be
// called by the calling application.
func (client *APIClient) initialize() error {
	var err error
	if err = client.authenticate(); err == nil {
		err = client.getHomeID()
	}
	return err
}

// authenticate logs in to tado.com and gets an Access Token to invoke the API functions.
// Once logged in, authenticate renews the Access Token if it's expired since the last call.
func (client *APIClient) authenticate() error {
	var err error
	if client.ClientSecret == "" {
		client.ClientSecret = "wZaRN7rpjn3FoNyF5IFuxg9uMzYJcvOoQ8QWiIqS3hfk6gLhVlG57j5YNoZL2Rtc"
	}
	// log.WithFields(log.Fields{"refreshTokenLen": len(client.RefreshToken), "expiry": client.Expires}).Debug("checking need to authenticate")
	if client.RefreshToken != "" {
		if time.Now().After(client.Expires) {
			err = client.doAuthentication("refresh_token", client.RefreshToken)
		}
	} else {
		err = client.doAuthentication("password", client.Password)
	}
	return err
}

func (client *APIClient) doAuthentication(grantType, credential string) error {
	var (
		err  error
		resp *http.Response
	)

	log.WithField("grant_type", grantType).Debug("authenticating")

	form := url.Values{}
	form.Add("client_id", "tado-web-app")
	form.Add("client_secret", client.ClientSecret)
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

	if err != nil && grantType == "refresh_token" {
		// failed during refresh. reset refresh_token to force a password login
		client.RefreshToken = ""
	}
	log.WithFields(log.Fields{"err": err, "expires": client.Expires}).Debug("authenticated")

	return err
}

func (client *APIClient) call(apiURL string) ([]byte, error) {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	req, _ = http.NewRequest("GET", apiURL, nil)
	req.Header.Add("Authorization", "Bearer "+client.AccessToken)
	if resp, err = client.HTTPClient.Do(req); err == nil {
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			return ioutil.ReadAll(resp.Body)
		case http.StatusForbidden:
			// we're authenticated, but still got forbidden.
			// force password login to get a new token.
			client.RefreshToken = ""
		}
		err = errors.New(resp.Status)
	}

	log.WithFields(log.Fields{
		"err":               err,
		"url":               apiURL,
		"expiry":            client.Expires,
		"accessTokenLength": len(client.AccessToken)},
	).Debug("call failed")

	return nil, err
}
