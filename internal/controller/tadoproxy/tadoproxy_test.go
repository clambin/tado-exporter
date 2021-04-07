package tadoproxy_test

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/tadoproxy"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProxyZones(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}

	go proxy.Run()

	statesResponse := make(chan map[int]model.ZoneState)

	proxy.GetZones <- statesResponse
	states := <-statesResponse

	if state, ok := states[1]; assert.True(t, ok) {
		assert.Equal(t, model.Auto, state.State)
	}

	if state, ok := states[2]; assert.True(t, ok) {
		assert.Equal(t, model.Auto, state.State)
	}

	proxy.SetZones <- map[int]model.ZoneState{2: {State: model.Manual, Temperature: tado.Temperature{Celsius: 25.0}}}

	proxy.GetZones <- statesResponse
	states = <-statesResponse

	if state, ok := states[2]; assert.True(t, ok) {
		assert.Equal(t, model.Manual, state.State)
		assert.Equal(t, 25.0, state.Temperature.Celsius)
	}

	proxy.SetZones <- map[int]model.ZoneState{2: {State: model.Off}}

	proxy.GetZones <- statesResponse
	states = <-statesResponse

	if state, ok := states[2]; assert.True(t, ok) {
		assert.Equal(t, model.Off, state.State)
	}

	proxy.SetZones <- map[int]model.ZoneState{2: {State: model.Auto}}

	proxy.GetZones <- statesResponse
	states = <-statesResponse

	if state, ok := states[2]; assert.True(t, ok) {
		assert.Equal(t, model.Auto, state.State)
	}

	proxy.Stop <- struct{}{}

	assert.Eventually(t, func() bool {
		_, ok := <-proxy.GetZones
		return ok == false
	}, 500*time.Millisecond, 10*time.Millisecond)

}

func TestProxyUsers(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}

	go proxy.Run()

	response := make(chan map[int]model.UserState)

	proxy.GetUsers <- response
	users := <-response

	assert.Len(t, users, 2)
	if state, ok := users[1]; assert.True(t, ok) {
		assert.Equal(t, model.UserHome, state)
	}
	if state, ok := users[2]; assert.True(t, ok) {
		assert.Equal(t, model.UserAway, state)
	}

	proxy.Stop <- struct{}{}

	assert.Eventually(t, func() bool {
		_, ok := <-proxy.GetZones
		return ok == false
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestAllZones(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}

	go proxy.Run()

	response := make(chan map[int]string)

	proxy.GetAllZones <- response
	zones := <-response

	assert.Len(t, zones, 2)
	if name, ok := zones[1]; assert.True(t, ok) {
		assert.Equal(t, "foo", name)
	}
	if name, ok := zones[2]; assert.True(t, ok) {
		assert.Equal(t, "bar", name)
	}

	proxy.Stop <- struct{}{}

	assert.Eventually(t, func() bool {
		_, ok := <-proxy.GetZones
		return ok == false
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestAllUsers(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}

	go proxy.Run()

	response := make(chan map[int]string)

	proxy.GetAllUsers <- response
	users := <-response

	assert.Len(t, users, 2)
	if name, ok := users[1]; assert.True(t, ok) {
		assert.Equal(t, "foo", name)
	}
	if name, ok := users[2]; assert.True(t, ok) {
		assert.Equal(t, "bar", name)
	}

	proxy.Stop <- struct{}{}

	assert.Eventually(t, func() bool {
		_, ok := <-proxy.GetZones
		return ok == false
	}, 500*time.Millisecond, 10*time.Millisecond)
}
