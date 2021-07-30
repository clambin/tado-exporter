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
	c.ZoneManager.PostChannel = postChannel
	go c.Run(ctx)

	pollr.Register <- c.ZoneManager.Update

	b.ResetTimer()

	for i := 0; i < 10; i++ {
		_ = server.SetZoneOverlay(ctx, 2, 15.5)
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
	c.ZoneManager.PostChannel = postChannel
	go c.Run(ctx)

	pollr.Register <- c.ZoneManager.Update

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
	c.ZoneManager.PostChannel = postChannel
	go c.Run(ctx)
	pollr.Register <- c.ZoneManager.Update

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

	c.ZoneManager.ReportTasks()
	msg = <-postChannel
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "no rules have been triggered", msg[0].Text)
	}
}

func zoneInOverlay(ctx context.Context, server tado.API, zoneID int) bool {
	info, err := server.GetZoneInfo(ctx, zoneID)
	return err == nil && info.Overlay.Type != ""
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
	// c.ZoneManager.PostChannel = tadoBot.PostChannel
	go c.Run(ctx)
	pollr.Register <- c.ZoneManager.Update

	// log.SetLevel(log.DebugLevel)

	events <- slackClient.ConnectedEvent()

	events <- slackClient.MessageEvent("2", "<@1234> help")
	msg := <-output

	if assert.Len(t, msg.Attachments, 1) {
		assert.Equal(t, "supported commands", msg.Attachments[0].Title)
		assert.Equal(t, "help, rules, version", msg.Attachments[0].Text)
	}

	events <- slackClient.MessageEvent("2", "<@1234> rules")
	msg = <-output

	if assert.Len(t, msg.Attachments, 1) {
		assert.Equal(t, "", msg.Attachments[0].Title)
		assert.Equal(t, "no rules have been triggered", msg.Attachments[0].Text)
	}

}
