package tadoprobe

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

func (client *APIClient) Initialize() error {
	var err error
	if err = client.Authenticate(); err == nil {
		err = client.GetHomeID()
	}
	return err
}

func (client *APIClient) Authenticate() error {
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
	log.WithFields(log.Fields{"err": err, "expires": client.Expires}).Info("authentication")

	return err
}

func (client *APIClient) GetHomeID() error {
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

func (client *APIClient) GetZones() ([]TadoZone, error) {
	var (
		err  error
		resp *http.Response
	)
	tadoZones := make([]TadoZone, 0)

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

func (client *APIClient) GetZoneInfo(zoneID int) (*TadoZoneInfo, error) {
	var (
		err          error
		resp         *http.Response
		tadoZoneInfo TadoZoneInfo
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

func (client *APIClient) GetWeatherInfo() (*TadoWeatherInfo, error) {
	var (
		err             error
		resp            *http.Response
		tadoWeatherInfo TadoWeatherInfo
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
