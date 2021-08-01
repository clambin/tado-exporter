package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/slackbot/mock"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func BenchmarkController_Run(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 1*time.Millisecond)

	postChannel := make(slackbot.PostChannel)
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   1 * time.Millisecond,
		},
	}}

	c, _ := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	c.PostChannel = postChannel
	go c.Run(ctx)

	pollr.Register <- c.Update

	b.ResetTimer()

	for i := 0; i < 100; i++ {
		_ = server.SetZoneOverlay(ctx, 2, 15.5)
		_ = c.ReportRules()
		_ = <-postChannel
		_ = <-postChannel
	}
}

func TestController_Run(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, _ := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	c.PostChannel = postChannel
	go c.Run(ctx)

	pollr.Register <- c.Update

	log.SetLevel(log.DebugLevel)

	_ = server.SetZoneOverlay(ctx, 2, 15.5)
	assert.True(t, zoneInOverlay(ctx, server, 2))

	_ = <-postChannel
	_ = <-postChannel

	assert.False(t, zoneInOverlay(ctx, server, 2))

}

func TestController_RevertedOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   1 * time.Hour,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, _ := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	c.PostChannel = postChannel
	go c.Run(ctx)
	pollr.Register <- c.Update

	log.SetLevel(log.DebugLevel)

	err := server.SetZoneOverlay(ctx, 2, 15.5)
	assert.Nil(t, err)

	_ = <-postChannel

	assert.True(t, zoneInOverlay(ctx, server, 2))

	err = server.DeleteZoneOverlay(ctx, 2)
	assert.Nil(t, err)

	msg := <-postChannel

	if assert.Len(t, msg, 1) {
		assert.Equal(t, "resetting rule for bar", msg[0].Title)
	}

	msg = c.ReportRules()
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "no rules have been triggered", msg[0].Text)
	}
}

func TestController_TadoBot(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   1 * time.Hour,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	tadoBot := slackbot.Create("", "", nil)

	events := make(chan slack.RTMEvent)
	output := make(chan slackbot.SlackMessage)
	slackClient := &mock.Client{
		UserID:    "1234",
		Channels:  []string{"1", "2", "3"},
		EventsIn:  events,
		EventsOut: tadoBot.Events,
		Output:    output,
	}
	tadoBot.SlackClient = slackClient
	go func(ctx context.Context) {
		err := tadoBot.Run(ctx)
		assert.NoError(t, err)
	}(ctx)

	c, _ := controller.New(
		server,
		&configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig},
		tadoBot,
	)
	go c.Run(ctx)
	pollr.Register <- c.Update

	// log.SetLevel(log.DebugLevel)

	events <- slackClient.ConnectedEvent()

	events <- slackClient.MessageEvent("2", "<@1234> help")
	msg := <-output

	if assert.Len(t, msg.Attachments, 1) {
		assert.Equal(t, "supported commands", msg.Attachments[0].Title)
		assert.Equal(t, "help, rooms, rules, version", msg.Attachments[0].Text)
	}

	events <- slackClient.MessageEvent("2", "<@1234> rules")
	msg = <-output

	if assert.Len(t, msg.Attachments, 1) {
		assert.Equal(t, "", msg.Attachments[0].Title)
		assert.Equal(t, "no rules have been triggered", msg.Attachments[0].Text)
	}
}

//
var fakeUpdates = []poller.Update{
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, err := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	assert.NoError(t, err)
	c.PostChannel = postChannel

	go c.Run(ctx)

	// manual mode
	_ = c.API.SetZoneOverlay(ctx, 2, 15.0)
	c.Update <- &fakeUpdates[2]

	_ = <-postChannel
	assert.True(t, zoneInOverlay(ctx, c.API, 2))

	// back to auto mode
	c.Update <- &fakeUpdates[3]
	resp := <-postChannel
	assert.Len(t, resp, 1)
	assert.Equal(t, "resetting rule for bar", resp[0].Title)

	// back to manual mode
	c.Update <- &fakeUpdates[2]

	_ = <-postChannel
	_ = <-postChannel
	assert.False(t, zoneInOverlay(ctx, c.API, 2))
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, err := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	assert.NoError(t, err)
	c.PostChannel = postChannel

	go c.Run(ctx)

	c.Update <- &fakeUpdates[2]

	msgs := <-postChannel
	if assert.Len(t, msgs, 1) {
		assert.Equal(t, "manual temperature setting detected in bar", msgs[0].Title)
		assert.Contains(t, msgs[0].Text, "moving to auto mode in ")
	}

	assert.False(t, zoneInOverlay(ctx, c.API, 2))

	msgs = c.ReportRules()
	if assert.Len(t, msgs, 1) {
		assert.Contains(t, msgs[0].Text, "bar: moving to auto mode in ")
	}
}

func TestZoneManager_AutoAway(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   20 * time.Millisecond,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, err := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	assert.NoError(t, err)
	c.PostChannel = postChannel

	go c.Run(ctx)

	// user is away
	c.Update <- &fakeUpdates[0]

	msgs := <-postChannel
	if assert.Len(t, msgs, 1) {
		assert.Equal(t, "bar: bar is away", msgs[0].Title)
		assert.Contains(t, msgs[0].Text, "switching off heating in ")
	}

	// validate that the zone is switched off
	assert.Eventually(t, func() bool { return zoneInOverlay(ctx, c.API, 2) }, 500*time.Millisecond, 10*time.Millisecond)

	msgs = <-postChannel
	if assert.Len(t, msgs, 1) {
		assert.Equal(t, "bar: bar is away", msgs[0].Title)
		assert.Equal(t, "switching off heating", msgs[0].Text)
	}

	// user is away & room in overlay
	c.Update <- &fakeUpdates[4]

	// user comes home
	c.Update <- &fakeUpdates[1]

	// validate that the zone is switched back to auto
	assert.Eventually(t, func() bool { return !zoneInOverlay(ctx, c.API, 2) }, 500*time.Millisecond, 10*time.Millisecond)

	msgs = <-postChannel
	if assert.Len(t, msgs, 1) {
		assert.Equal(t, "bar: bar is home", msgs[0].Title)
		assert.Contains(t, msgs[0].Text, "moving to auto mode")
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, err := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: zoneConfig,
		},
		nil)
	assert.NoError(t, err)
	c.PostChannel = postChannel

	go c.Run(ctx)

	// user is away
	c.Update <- &fakeUpdates[0]

	// notification that zone will be switched off
	msg := <-postChannel
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
	assert.True(t, zoneInOverlay(ctx, c.API, 2))

	// user comes home
	c.Update <- &fakeUpdates[1]

	// notification that zone will be switched on again
	msg = <-postChannel
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "bar: bar is home", msg[0].Title)
		assert.Equal(t, "moving to auto mode", msg[0].Text)
	}

	assert.False(t, zoneInOverlay(ctx, c.API, 2))

	log.SetLevel(log.DebugLevel)
	// user is home & room set to manual
	c.Update <- &fakeUpdates[2]

	// notification that zone will be switched back to auto
	msg = <-postChannel
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "manual temperature setting detected in bar", msg[0].Title)
		assert.Contains(t, msg[0].Text, "moving to auto mode in ")
	}

	// report should say that a rule is triggered
	msg = c.ReportRules()
	if assert.Len(t, msg, 1) {
		assert.Contains(t, msg[0].Text, "bar: moving to auto mode in ")
	}
}

func zoneInOverlay(ctx context.Context, server tado.API, zoneID int) bool {
	info, err := server.GetZoneInfo(ctx, zoneID)
	return err == nil && info.Overlay.Type != ""
}
