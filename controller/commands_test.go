package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	pollerMock "github.com/clambin/tado-exporter/poller/mocks"
	slackMock "github.com/clambin/tado-exporter/slackbot/mocks"
	tadoMock "github.com/clambin/tado/mocks"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestController_Rules(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{
		{
			ZoneName: "foo",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   1 * time.Hour,
				Users:   []configuration.ZoneUser{{MobileDeviceID: 1}},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &tadoMock.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	pollr := &pollerMock.Poller{}

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, pollr)
	assert.NoError(t, err)

	log.SetLevel(log.DebugLevel)
	go c.Run(ctx, 10*time.Millisecond)
	c.Updates <- &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
		},
		Zones: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
		},
	}

	time.Sleep(100 * time.Millisecond)
	msg := c.ReportRules(ctx)
	require.Len(t, msg, 1)
	assert.Equal(t, "no rules have been triggered", msg[0].Text)

	// foo: user is away. bar: room is manual
	bot.
		On("Send", "", "good", "foo: foo is away", "switching off heating in 1h0m0s").
		Return(nil).
		Once()

	c.Updates <- &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
		},
		Zones: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}, Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: 22.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{
					Type: "MANUAL",
				},
			}},
		},
	}

	time.Sleep(100 * time.Millisecond)
	msg = c.ReportRules(ctx)
	require.Len(t, msg, 1)
	assert.Contains(t, msg[0].Text, "foo: switching off heating in ")

	mock.AssertExpectationsForObjects(t, server, bot, pollr)
}

func TestController_Rooms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &tadoMock.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	pollr := &pollerMock.Poller{}

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, bot, pollr)
	require.NoError(t, err)

	c.Update(&poller.Update{
		UserInfo: updateMobileDeviceUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoManual,
	})
	go c.Run(ctx, 10*time.Millisecond)

	attachments := c.ReportRooms(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Equal(t, "foo: 22.0ºC (target: 18.0, MANUAL)", attachments[0].Text)

	mock.AssertExpectationsForObjects(t, server, bot, pollr)

}

func TestController_SetRoom(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &tadoMock.API{}

	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)

	pollr := &pollerMock.Poller{}

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, bot, pollr)
	require.NoError(t, err)

	go c.Run(ctx, 10*time.Millisecond)
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoAuto,
	}

	time.Sleep(100 * time.Millisecond)

	var attachments []slack.Attachment

	type TestCaseStruct struct {
		Args     []string
		Color    string
		Text     string
		Action   bool
		Delete   bool
		Duration time.Duration
	}

	var testCases = []TestCaseStruct{
		{
			Args:  []string{},
			Color: "bad",
			Text:  "invalid command: missing parameters\nUsage: set <room> [auto|<temperature> [<duration>]",
		},
		{
			Args:  []string{"foo"},
			Color: "bad",
			Text:  "invalid command: missing parameters\nUsage: set <room> [auto|<temperature> [<duration>]",
		},
		{
			Args:  []string{"notaroom", "auto"},
			Color: "bad",
			Text:  "invalid command: invalid room name",
		},
		{
			Args:  []string{"foo", "25,0"},
			Color: "bad",
			Text:  "invalid command: invalid target temperature: \"25,0\"",
		},
		{
			Args:  []string{"foo", "25.0", "invalid"},
			Color: "bad",
			Text:  "invalid command: invalid duration: \"invalid\"",
		},
		{
			Args:   []string{"foo", "25.0"},
			Color:  "good",
			Text:   "Setting target temperature for foo to 25.0ºC",
			Action: true,
		},
		{
			Args:     []string{"foo", "25.0", "5m"},
			Color:    "good",
			Text:     "Setting target temperature for foo to 25.0ºC for 5m0s",
			Action:   true,
			Duration: 5 * time.Minute,
		},
		{
			Args:   []string{"foo", "auto"},
			Color:  "good",
			Text:   "Setting foo to automatic mode",
			Action: true,
			Delete: true,
		},
	}

	for index, testCase := range testCases {
		if testCase.Action {
			if testCase.Delete {
				server.On("DeleteZoneOverlay", mock.Anything, 1).Return(nil).Once()
			} else {
				server.On("SetZoneOverlayWithDuration", mock.Anything, 1, 25.0, testCase.Duration).Return(nil).Once()
			}
			pollr.On("Refresh").Return(nil).Once()
		}

		attachments = c.SetRoom(ctx, testCase.Args...)

		require.Len(t, attachments, 1, index)
		assert.Equal(t, testCase.Color, attachments[0].Color, index)
		assert.Empty(t, attachments[0].Title, index)
		assert.Equal(t, testCase.Text, attachments[0].Text, index)
	}

	mock.AssertExpectationsForObjects(t, server, pollr, bot)
}

func TestController_DoRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	api := &tadoMock.API{}
	bot := &slackMock.SlackBot{}
	pollr := &pollerMock.Poller{}

	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)

	c, err := controller.New(api, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, bot, pollr)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		c.Run(ctx, 10*time.Millisecond)
		wg.Done()
	}()

	pollr.On("Refresh").Return(nil).Once()
	attachments := c.DoRefresh(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "refreshing Tado data", attachments[0].Text)

	cancel()
	wg.Wait()

	mock.AssertExpectationsForObjects(t, api, bot, pollr)
}
