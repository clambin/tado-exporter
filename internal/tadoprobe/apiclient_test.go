package tadoprobe_test

import (
	"github.com/clambin/gotools/httpstub"
	"tado-exporter/internal/tadoprobe"
	"tado-exporter/internal/testtools"
	"time"

	"github.com/stretchr/testify/assert"

	"testing"
)

func TestAPIClient_Initialization(t *testing.T) {
	client := tadoprobe.APIClient{
		HTTPClient: httpstub.NewTestClient(testtools.APIServer),
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

func TestAPIClient_Authentication(t *testing.T) {
	client := tadoprobe.APIClient{
		HTTPClient: httpstub.NewTestClient(testtools.APIServer),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	var err error
	err = client.Authenticate()
	assert.Nil(t, err)
	assert.Equal(t, "access_token", client.AccessToken)

	client.Expires = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	err = client.Authenticate()
	assert.Nil(t, err)
	assert.Greater(t, client.Expires.Unix(), time.Now().Unix())

	err = client.GetHomeID()
	assert.Nil(t, err)
	assert.Equal(t, 242, client.HomeID)
}

func TestAPIClient_Get(t *testing.T) {
	client := tadoprobe.APIClient{
		HTTPClient: httpstub.NewTestClient(testtools.APIServer),
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
