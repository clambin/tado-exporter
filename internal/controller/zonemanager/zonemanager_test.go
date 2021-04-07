package zonemanager_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/tadoproxy"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TODO: timing-based testing can be unreliable
// Could refactor to stub the Proxy and monitor the commands it receives?

func TestZoneManager_AutoAway(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		Users:    []configuration.ZoneUser{{MobileDeviceName: "bar"}},
	}}

	mgr := zonemanager.New(zoneConfig, 10*time.Millisecond, proxy)
	go mgr.Run()

	assert.Eventually(t, func() bool {
		response := make(chan map[int]model.ZoneState)
		proxy.GetZones <- response
		if states, ok := <-response; ok == true {
			if state, ok := states[2]; ok == true {
				if state.State == model.Off {
					return true
				}
			}
		}
		return false
	}, 500*time.Millisecond, 10*time.Millisecond)

	// TODO: test when a user comes home
}

func TestZoneManager_LimitOverlay(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	proxy.SetZones <- map[int]model.ZoneState{
		2: {
			State:       model.Manual,
			Temperature: tado.Temperature{Celsius: 18.5},
		},
	}

	response := make(chan map[int]model.ZoneState)
	proxy.GetZones <- response
	if states, ok := <-response; assert.True(t, ok) {
		if state, ok := states[2]; assert.True(t, ok) {
			assert.Equal(t, model.Manual, state.State)
		}
	}

	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Limit:   100 * time.Millisecond,
		},
	}}

	mgr := zonemanager.New(zoneConfig, 10*time.Millisecond, proxy)
	go mgr.Run()

	assert.Eventually(t, func() bool {
		response = make(chan map[int]model.ZoneState)
		proxy.GetZones <- response
		if states, ok := <-response; ok == true {
			if state, ok := states[2]; ok == true {
				if state.State == model.Auto {
					return true
				}
			}
		}
		return false
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestZoneManager_NightTime(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	proxy.SetZones <- map[int]model.ZoneState{
		2: {
			State:       model.Manual,
			Temperature: tado.Temperature{Celsius: 18.5},
		},
	}

	response := make(chan map[int]model.ZoneState)
	proxy.GetZones <- response
	if states, ok := <-response; assert.True(t, ok) {
		if state, ok := states[2]; assert.True(t, ok) {
			assert.Equal(t, model.Manual, state.State)
		}
	}

	now := time.Now()
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    now.Hour(),
				Minutes: now.Minute(),
				Seconds: now.Second() + 1,
			},
		},
	}}

	mgr := zonemanager.New(zoneConfig, 10*time.Millisecond, proxy)
	go mgr.Run()

	assert.Eventually(t, func() bool {
		response = make(chan map[int]model.ZoneState)
		proxy.GetZones <- response
		if states, ok := <-response; ok == true {
			if state, ok := states[2]; ok == true {
				if state.State == model.Auto {
					return true
				}
			}
		}
		return false
	}, 2000*time.Millisecond, 10*time.Millisecond)
}

func TestZoneManager_NightTime_Fail(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	proxy.SetZones <- map[int]model.ZoneState{
		2: {
			State:       model.Manual,
			Temperature: tado.Temperature{Celsius: 18.5},
		},
	}

	response := make(chan map[int]model.ZoneState)
	proxy.GetZones <- response
	if states, ok := <-response; assert.True(t, ok) {
		if state, ok := states[2]; assert.True(t, ok) {
			assert.Equal(t, model.Manual, state.State)
		}
	}

	now := time.Now()
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    now.Hour() + 1,
				Minutes: now.Minute(),
				Seconds: now.Second(),
			},
		},
	}}

	mgr := zonemanager.New(zoneConfig, 10*time.Millisecond, proxy)
	go mgr.Run()

	assert.Never(t, func() bool {
		response = make(chan map[int]model.ZoneState)
		proxy.GetZones <- response
		if states, ok := <-response; ok == true {
			if state, ok := states[2]; ok == true {
				if state.State == model.Auto {
					return true
				}
			}
		}
		return false
	}, 500*time.Millisecond, 10*time.Millisecond)
}
