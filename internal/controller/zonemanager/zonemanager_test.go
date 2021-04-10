package zonemanager_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler/mockscheduler"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TODO: timing-based testing can be unreliable

func TestZoneManager_Load(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{
		{
			ZoneName: "bar",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
				Delay:   1 * time.Hour,
			},
		},
		{
			ZoneName: "invalid",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Users:   []configuration.ZoneUser{{MobileDeviceName: "invalid"}},
				Delay:   1 * time.Hour,
			},
		},
	}

	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, nil, nil)

	assert.Nil(t, err)

	if assert.Len(t, mgr.ZoneConfig, 1) {
		if zone, ok := mgr.ZoneConfig[2]; assert.True(t, ok) {
			if assert.Len(t, zone.AutoAway.Users, 1) {
				assert.Equal(t, 2, zone.AutoAway.Users[0])
			}
		}
	}
}

var fakeUpdates = []poller.Update{
	{
		ZoneStates: map[int]model.ZoneState{2: {State: model.Auto, Temperature: tado.Temperature{Celsius: 25.0}}},
		UserStates: map[int]model.UserState{2: model.UserAway},
	},
	{
		ZoneStates: map[int]model.ZoneState{2: {State: model.Off, Temperature: tado.Temperature{Celsius: 15.0}}},
		UserStates: map[int]model.UserState{2: model.UserHome},
	},
	{
		ZoneStates: map[int]model.ZoneState{2: {State: model.Manual, Temperature: tado.Temperature{Celsius: 20.0}}},
		UserStates: map[int]model.UserState{2: model.UserHome},
	},
	{
		ZoneStates: map[int]model.ZoneState{2: {State: model.Auto, Temperature: tado.Temperature{Celsius: 25.0}}},
		UserStates: map[int]model.UserState{2: model.UserHome},
	},
}

func TestZoneManager_AutoAway(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   1 * time.Hour,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
	}}

	schedulr := mockscheduler.New()
	updates := make(chan poller.Update)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, updates, schedulr)

	if assert.Nil(t, err) {
		go mgr.Run()

		// user is away
		updates <- fakeUpdates[0]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Off
		}, 500*time.Millisecond, 10*time.Millisecond)

		// user comes home
		updates <- fakeUpdates[1]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Unknown
		}, 500*time.Millisecond, 10*time.Millisecond)

		mgr.Cancel <- struct{}{}
	}
}
func TestZoneManager_LimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
	}}

	updates := make(chan poller.Update)
	schedulr := mockscheduler.New()
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, updates, schedulr)

	if assert.Nil(t, err) {
		go mgr.Run()

		// manual mode
		updates <- fakeUpdates[2]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Auto
		}, 500*time.Millisecond, 10*time.Millisecond)

		// back to auto mode
		updates <- fakeUpdates[3]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Auto
		}, 500*time.Millisecond, 10*time.Millisecond)

		// back to manual mode
		updates <- fakeUpdates[2]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Auto
		}, 500*time.Millisecond, 10*time.Millisecond)

		mgr.Cancel <- struct{}{}
	}
}

func TestZoneManager_NightTime(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    23,
				Minutes: 30,
			},
		},
	}}

	updates := make(chan poller.Update)
	schedulr := mockscheduler.New()
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, updates, schedulr)

	if assert.Nil(t, err) {
		go mgr.Run()

		updates <- fakeUpdates[2]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Auto
		}, 500*time.Millisecond, 10*time.Millisecond)

		mgr.Cancel <- struct{}{}
	}
}

func TestZoneManager_Combined(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   1 * time.Hour,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
	}}

	updates := make(chan poller.Update)
	schedulr := mockscheduler.New()
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, updates, schedulr)

	if assert.Nil(t, err) {
		go mgr.Run()

		// user comes home
		updates <- fakeUpdates[0]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Off
		}, 500*time.Millisecond, 10*time.Millisecond)

		// user comes home
		updates <- fakeUpdates[1]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Unknown
		}, 500*time.Millisecond, 10*time.Millisecond)

		// user is home & room set to manual
		updates <- fakeUpdates[2]

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Auto
		}, 500*time.Millisecond, 10*time.Millisecond)

		mgr.Cancel <- struct{}{}
	}
}

func BenchmarkZoneManager_LimitOverlay(b *testing.B) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
	}}

	server := &mockapi.MockAPI{}
	updates := make(chan poller.Update)
	schedulr := mockscheduler.New()
	mgr, err := zonemanager.New(server, zoneConfig, updates, schedulr)

	if assert.Nil(b, err) {
		go mgr.Run()
		b.ResetTimer()

		for i := 0; i < 100; i++ {
			updates <- fakeUpdates[2]
			if i == 0 {
				assert.Eventually(b, func() bool {
					return schedulr.ScheduledState(2).State == model.Auto
				}, 500*time.Millisecond, 10*time.Millisecond)
			} else {
				assert.Equal(b, model.Auto, schedulr.ScheduledState(2).State)
			}
		}

		mgr.Cancel <- struct{}{}
	}
}
