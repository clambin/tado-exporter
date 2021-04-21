package zonemanager_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
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

	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, nil)

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
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneAuto, Temperature: tado.Temperature{Celsius: 25.0}}},
		UserStates: map[int]models.UserState{2: models.UserAway},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneOff, Temperature: tado.Temperature{Celsius: 15.0}}},
		UserStates: map[int]models.UserState{2: models.UserHome},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 20.0}}},
		UserStates: map[int]models.UserState{2: models.UserHome},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneAuto, Temperature: tado.Temperature{Celsius: 25.0}}},
		UserStates: map[int]models.UserState{2: models.UserHome},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneOff, Temperature: tado.Temperature{Celsius: 15.0}}},
		UserStates: map[int]models.UserState{2: models.UserAway},
	},
}

func TestZoneManager_LimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   200 * time.Millisecond,
		},
	}}
	log.SetLevel(log.DebugLevel)

	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, postChannel)

	if assert.Nil(t, err) {
		go mgr.Run()

		// manual mode
		_ = mgr.API.SetZoneOverlay(2, 15.0)
		mgr.Update <- fakeUpdates[2]

		_ = <-postChannel
		assert.True(t, zoneInOverlay(mgr.API, 2))

		// back to auto mode
		mgr.Update <- fakeUpdates[3]
		resp := <-postChannel
		assert.Len(t, resp, 1)
		assert.Equal(t, "resetting rule for bar", resp[0].Title)

		// back to manual mode
		mgr.Update <- fakeUpdates[2]

		_ = <-postChannel
		_ = <-postChannel
		assert.False(t, zoneInOverlay(mgr.API, 2))

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

	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, postChannel)

	if assert.Nil(t, err) {
		go mgr.Run()

		mgr.Update <- fakeUpdates[2]

		_ = <-postChannel

		_ = mgr.ReportTasks()

		assert.False(t, zoneInOverlay(mgr.API, 2))
		msgs := <-postChannel
		if assert.Len(t, msgs, 1) {
			assert.Contains(t, msgs[0].Text, "bar: moving to auto mode in ")
		}

		mgr.Cancel <- struct{}{}
	}
}

func TestZoneManager_AutoAway(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   10 * time.Millisecond,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
	}}

	var msg []slack.Attachment
	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, postChannel)

	if assert.Nil(t, err) {
		go mgr.Run()

		// user is away
		mgr.Update <- fakeUpdates[0]

		// notification that zone will be switched off
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "bar: bar is away", msg[0].Title)
			assert.Contains(t, msg[0].Text, "switching off heating in ")
		}

		// notification that zone gets switched off
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "bar: bar is away", msg[0].Title)
			assert.Contains(t, msg[0].Text, "switching off heating")
		}

		// validate that the zone is switched off
		assert.True(t, zoneInOverlay(mgr.API, 2))

		// user is still away
		mgr.Update <- fakeUpdates[4]

		// shouldn't trigger any actions and no rules should be outstanding
		// (if it did trigger an action, the next message on postChannel wouldn't be "no rules have been triggered")
		mgr.ReportTasks()
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "no rules have been triggered", msg[0].Text)
		}

		mgr.Cancel <- struct{}{}
	}
}

func TestZoneManager_Combined(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   10 * time.Millisecond,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    01,
				Minutes: 30,
				Seconds: 30,
			},
		},
	}}

	var msg []slack.Attachment
	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, postChannel)

	if assert.Nil(t, err) {
		go mgr.Run()

		// user is away
		mgr.Update <- fakeUpdates[0]

		// notification that zone will be switched off
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "bar: bar is away", msg[0].Title)
			assert.Contains(t, msg[0].Text, "switching off heating in ")
		}

		// notification that zone gets switched off
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "bar: bar is away", msg[0].Title)
			assert.Contains(t, msg[0].Text, "switching off heating")
		}

		// validate that the zone is switched off
		assert.True(t, zoneInOverlay(mgr.API, 2))

		// user comes home
		mgr.Update <- fakeUpdates[1]

		// notification that zone will be switched on again
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "bar: bar is home", msg[0].Title)
			assert.Equal(t, "moving to auto mode", msg[0].Text)
		}

		assert.False(t, zoneInOverlay(mgr.API, 2))

		log.SetLevel(log.DebugLevel)
		// user is home & room set to manual
		mgr.Update <- fakeUpdates[2]

		// notification that zone will be switched back to auto
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "manual temperature setting detected in bar", msg[0].Title)
			assert.Contains(t, msg[0].Text, "moving to auto mode in ")
		}

		// report should say that a rule is triggered
		mgr.ReportTasks()
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Contains(t, msg[0].Text, "bar: moving to auto mode in ")
		}

		mgr.Cancel <- struct{}{}
	}
}

func TestManager_ReportTasks(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   1 * time.Hour,
			Users: []configuration.ZoneUser{{
				MobileDeviceID: 2,
			}},
		},
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   1 * time.Hour,
		},
	}}

	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, postChannel)

	if assert.Nil(t, err) {
		go mgr.Run()

		log.SetLevel(log.DebugLevel)

		_ = mgr.ReportTasks()
		msg := <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Equal(t, "no rules have been triggered", msg[0].Text)
		}

		// user is away
		mgr.Update <- fakeUpdates[0]
		_ = <-postChannel

		_ = mgr.ReportTasks()

		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Contains(t, msg[0].Text, "bar: switching off heating in ")
		}

		// user is home & room set to manual
		mgr.Update <- fakeUpdates[2]
		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Contains(t, msg[0].Text, "moving to auto mode in ")
		}

		_ = mgr.ReportTasks()

		msg = <-postChannel
		if assert.Len(t, msg, 1) {
			assert.Contains(t, msg[0].Text, "bar: moving to auto mode in ")
		}
	}
}

func zoneInOverlay(server tado.API, zoneID int) bool {
	info, err := server.GetZoneInfo(zoneID)
	return err == nil && info.Overlay.Type != ""

}

func BenchmarkZoneManager_LimitOverlay(b *testing.B) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   10 * time.Millisecond,
		},
	}}

	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(&mockapi.MockAPI{}, zoneConfig, postChannel)

	if assert.Nil(b, err) {
		go mgr.Run()
		b.ResetTimer()

		_ = mgr.API.SetZoneOverlay(2, 5.0)
		mgr.Update <- fakeUpdates[2]

		_ = <-postChannel
		_ = <-postChannel

		assert.False(b, zoneInOverlay(mgr.API, 2))

		mgr.Cancel <- struct{}{}
	}
}
