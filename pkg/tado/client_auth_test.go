package tado_test

import (
	"github.com/clambin/gotools/httpstub"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAPIClient_Initialization(t *testing.T) {
	server := APIServer{}
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(server.serve),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	var err error
	_, err = client.GetZones()
	assert.Nil(t, err)
	assert.Equal(t, "token_1", client.AccessToken)
	assert.Equal(t, 242, client.HomeID)
}

func TestAPIClient_Authentication(t *testing.T) {
	server := APIServer{}
	client := tado.APIClient{
		HTTPClient: httpstub.NewTestClient(server.serve),
		Username:   "user@examle.com",
		Password:   "some-password",
	}

	var err error
	_, err = client.GetZones()
	assert.Nil(t, err)
	assert.Equal(t, "token_1", client.AccessToken)

	// expire token on client side. we should get a new token.
	client.Expires = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.GetZones()
	assert.Nil(t, err)
	assert.Equal(t, "token_2", client.AccessToken)

	// expire token on server side. we should get a 'forbidden' error
	server.expires = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.GetZones()
	assert.NotNil(t, err)
	assert.Equal(t, "Forbidden", err.Error())

	// now retry. we should go back to a reset token
	_, err = client.GetZones()
	assert.Nil(t, err)
	assert.Equal(t, "token_1", client.AccessToken)
	assert.Equal(t, 242, client.HomeID)

	// expire token on client side + set server to failRefresh
	// this should trigger client to do a password-based login
	client.Expires = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	server.failRefresh = true
	_, err = client.GetZones()
	assert.NotNil(t, err)
	_, err = client.GetZones()
	assert.Nil(t, err)
	// token_1 means we logged in w/ password, not refresh_token
	assert.Equal(t, "token_1", client.AccessToken)
	assert.Equal(t, 242, client.HomeID)
}
