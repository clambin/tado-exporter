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

func TestZoneManager_Load(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	zoneConfig := []configuration.ZoneConfig{
		{
			ZoneName: "bar",
			Users:    []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
		{
			ZoneName: "invalid",
			Users:    []configuration.ZoneUser{{MobileDeviceName: "invalid"}},
		},
	}

	mgr := zonemanager.New(zoneConfig, proxy)

	if assert.Len(t, mgr.ZoneConfig, 1) {
		if zone, ok := mgr.ZoneConfig[2]; assert.True(t, ok) {
			if assert.Len(t, zone.Users, 1) {
				assert.Equal(t, 2, zone.Users[0])
			}
		}
	}
}

func TestZoneManager_AutoAway(t *testing.T) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		Users:    []configuration.ZoneUser{{MobileDeviceName: "bar"}},
	}}

	mgr := zonemanager.New(zoneConfig, proxy)

	updates := mgr.Update()

	if assert.Len(t, updates, 1) {
		if state, ok := updates[2]; assert.True(t, ok) {
			assert.Equal(t, model.Off, state.State)
		}
	}

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
		if state, ok2 := states[2]; assert.True(t, ok2) {
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

	mgr := zonemanager.New(zoneConfig, proxy)

	assert.Eventually(t, func() bool {
		updates := mgr.Update()
		return len(updates) == 1 && updates[2].State == model.Auto
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
		if state, ok2 := states[2]; assert.True(t, ok2) {
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

	mgr := zonemanager.New(zoneConfig, proxy)

	assert.Eventually(t, func() bool {
		updates := mgr.Update()
		return len(updates) == 1 && updates[2].State == model.Auto
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
		if state, ok2 := states[2]; assert.True(t, ok2) {
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

	mgr := zonemanager.New(zoneConfig, proxy)

	assert.Never(t, func() bool {
		updates := mgr.Update()
		return len(updates) == 1 && updates[2].State == model.Auto
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func BenchmarkZoneManager_LimitOverlay(b *testing.B) {
	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	proxy.SetZones <- map[int]model.ZoneState{2: {State: model.Manual, Temperature: tado.Temperature{Celsius: 18.5}}}

	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Limit:   100 * time.Millisecond,
		},
	}}

	mgr := zonemanager.New(zoneConfig, proxy)

	b.ResetTimer()
	for {
		updates := mgr.Update()
		if len(updates) == 1 && updates[2].State == model.Auto {
			break
		}
	}
}
