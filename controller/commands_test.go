package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	mocks3 "github.com/clambin/tado-exporter/poller/mocks"
	mocks2 "github.com/clambin/tado-exporter/slackbot/mocks"
	"github.com/clambin/tado/mocks"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestManager_ReportRules(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{
		{
			ZoneName: "foo",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   1 * time.Hour,
				Users: []configuration.ZoneUser{{
					MobileDeviceID: 1,
				}},
			},
		},
		{
			ZoneName: "bar",
			LimitOverlay: configuration.ZoneLimitOverlay{
				Enabled: true,
				Delay:   1 * time.Hour,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mocks.API{}
	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	pollr := &mocks3.Poller{}

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, pollr)
	assert.NoError(t, err)

	log.SetLevel(log.DebugLevel)
	go c.Run(ctx)
	c.Updates <- &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
			2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
		},
		Zones: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
			2: {ID: 2, Name: "bar"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
			2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
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
	bot.
		On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode in 1h0m0s").
		Return(nil).
		Once()

	c.Updates <- &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
			2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
		},
		Zones: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
			2: {ID: 2, Name: "bar"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
			2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}, Overlay: tado.ZoneInfoOverlay{
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
	assert.Contains(t, msg[0].Text, "bar: moving to auto mode in ")

	mock.AssertExpectationsForObjects(t, server, bot, pollr)
}

func TestController_Rooms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mocks.API{}
	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	pollr := &mocks3.Poller{}

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, bot, pollr)
	require.NoError(t, err)

	c.Update(ctx, &poller.Update{
		UserInfo: updateUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	})
	go c.Run(ctx)

	attachments := c.ReportRooms(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Contains(t, attachments[0].Text, "bar: 22.0ºC (target: 22.0, MANUAL)")
	assert.Contains(t, attachments[0].Text, "foo: 22.0ºC (target: 15.5)")

	mock.AssertExpectationsForObjects(t, server, bot, pollr)

}

func TestController_SetRoom(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mocks.API{}

	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)

	pollr := &mocks3.Poller{}

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, bot, pollr)
	require.NoError(t, err)

	go c.Run(ctx)
	c.Updates <- &poller.Update{
		UserInfo: updateUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneAuto,
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
			Args:  []string{"bar"},
			Color: "bad",
			Text:  "invalid command: missing parameters\nUsage: set <room> [auto|<temperature> [<duration>]",
		},
		{
			Args:  []string{"notaroom", "auto"},
			Color: "bad",
			Text:  "invalid command: invalid room name",
		},
		{
			Args:  []string{"bar", "25,0"},
			Color: "bad",
			Text:  "invalid command: invalid target temperature: \"25,0\"",
		},
		{
			Args:  []string{"bar", "25.0", "invalid"},
			Color: "bad",
			Text:  "invalid command: invalid duration: \"invalid\"",
		},
		{
			Args:   []string{"bar", "25.0"},
			Color:  "good",
			Text:   "Setting target temperature for bar to 25.0ºC",
			Action: true,
		},
		{
			Args:     []string{"bar", "25.0", "5m"},
			Color:    "good",
			Text:     "Setting target temperature for bar to 25.0ºC for 5m0s",
			Action:   true,
			Duration: 5 * time.Minute,
		},
		{
			Args:   []string{"bar", "auto"},
			Color:  "good",
			Text:   "Setting bar to automatic mode",
			Action: true,
			Delete: true,
		},
	}

	for _, testCase := range testCases {
		if testCase.Action {
			if testCase.Delete {
				server.On("DeleteZoneOverlay", mock.Anything, 2).Return(nil).Once()
			} else {
				server.On("SetZoneOverlayWithDuration", mock.Anything, 2, 25.0, testCase.Duration).Return(nil).Once()
			}
			pollr.On("Refresh").Return(nil).Once()
		}

		attachments = c.SetRoom(ctx, testCase.Args...)

		assert.Len(t, attachments, 1)
		assert.Equal(t, testCase.Color, attachments[0].Color)
		assert.Empty(t, attachments[0].Title)
		assert.Equal(t, testCase.Text, attachments[0].Text)
	}

	mock.AssertExpectationsForObjects(t, server, pollr, bot)
}
