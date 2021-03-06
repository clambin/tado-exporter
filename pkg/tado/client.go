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
//   GetZones:         get the different zones (rooms) defined in your home
//   GetZoneInfo:      get metrics for a specified zone in your home
//   GetWeatherInfo:   get overall weather information
//   GetMobileDevices: get status of each registered mobile device
package tado

import (
	"bytes"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Common Tado data structures

// Temperature contains a temperature in degrees Celsius
type Temperature struct {
	Celsius float64 `json:"celsius"`
}

// Percentage contains a percentage (0-100%)
type Percentage struct {
	Percentage float64 `json:"percentage"`
}

// Value contains a string value
type Value struct {
	Value string `json:"value"`
}

// API for the Tado APIClient.
// Used to mock the API during unit testing
type API interface {
	GetZones() ([]*Zone, error)
	GetZoneInfo(zoneID int) (*ZoneInfo, error)
	GetWeatherInfo() (*WeatherInfo, error)
	GetMobileDevices() ([]*MobileDevice, error)
	SetZoneOverlay(int, float64) error
	DeleteZoneOverlay(int) error
}

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
	lock         sync.RWMutex
}

const baseAPIURL = "https://my.tado.com"
const authURL = "https://auth.tado.com/oauth/token"

// apiURL returns a API v2 URL
func (client *APIClient) apiURL(endpoint string) string {
	return baseAPIURL + "/api/v2/homes/" + strconv.Itoa(client.HomeID) + endpoint
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

	if body, err = client.call("GET", baseAPIURL+"/api/v1/me", ""); err == nil {
		var resp interface{}
		if err = json.Unmarshal(body, &resp); err == nil {
			m := resp.(map[string]interface{})
			client.HomeID = int(m["homeId"].(float64))
		}
	}
	return err
}

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
func (client *APIClient) authenticate() (err error) {
	client.lock.Lock()
	defer client.lock.Unlock()

	if client.ClientSecret == "" {
		client.ClientSecret = "wZaRN7rpjn3FoNyF5IFuxg9uMzYJcvOoQ8QWiIqS3hfk6gLhVlG57j5YNoZL2Rtc"
	}
	if client.RefreshToken != "" {
		if time.Now().After(client.Expires) {
			err = client.doAuthentication("refresh_token", client.RefreshToken)
		}
	} else {
		err = client.doAuthentication("password", client.Password)
	}
	return err
}

func (client *APIClient) getToken() string {
	client.lock.RLock()
	defer client.lock.RUnlock()
	return client.AccessToken
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

	req, _ := http.NewRequest("POST", authURL, strings.NewReader(form.Encode()))
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

func (client *APIClient) call(method string, apiURL string, payload string) ([]byte, error) {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	req, _ = http.NewRequest(method, apiURL, bytes.NewBufferString(payload))
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	req.Header.Add("Authorization", "Bearer "+client.getToken())
	if resp, err = client.HTTPClient.Do(req); err == nil {
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			return ioutil.ReadAll(resp.Body)
		case http.StatusNoContent:
			return []byte{}, nil
		case http.StatusForbidden:
			// we're authenticated, but still got forbidden.
			// force password login to get a new token.
			client.RefreshToken = ""
			err = errors.New(resp.Status)
		case http.StatusUnprocessableEntity:
			errBody, _ := ioutil.ReadAll(resp.Body)
			err = errors.New(string(errBody))
		default:
			err = errors.New(resp.Status)
		}
	}

	log.WithFields(log.Fields{
		"err":               err,
		"url":               apiURL,
		"expiry":            client.Expires,
		"accessTokenLength": len(client.AccessToken)},
	).Debug("call failed")

	return nil, err
}
