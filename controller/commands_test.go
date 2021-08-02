package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	pollMock "github.com/clambin/tado-exporter/poller/mock"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestManager_ReportRules(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	go pollr.Run(ctx, 20*time.Millisecond)

	postChannel := make(slackbot.PostChannel)

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, nil, pollr)
	assert.NoError(t, err)
	c.PostChannel = postChannel

	go c.Run(ctx)

	log.SetLevel(log.DebugLevel)

	msg := c.ReportRules(ctx)
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "no rules have been triggered", msg[0].Text)
	}

	// user is away
	c.Update <- &pollMock.FakeUpdates[0]
	_ = <-postChannel

	msg = c.ReportRules(ctx)
	if assert.Len(t, msg, 1) {
		assert.Contains(t, msg[0].Text, "bar: switching off heating in ")
	}

	// user is home & room set to manual
	c.Update <- &pollMock.FakeUpdates[2]
	msg = <-postChannel
	if assert.Len(t, msg, 1) {
		assert.Contains(t, msg[0].Text, "moving to auto mode in ")
	}

	msg = c.ReportRules(ctx)
	if assert.Len(t, msg, 1) {
		assert.Contains(t, msg[0].Text, "bar: moving to auto mode in ")
	}
}

func TestManager_ReportRooms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	_ = server.SetZoneOverlay(context.Background(), 2, 22.0)

	postChannel := make(slackbot.PostChannel)
	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, nil, nil)

	assert.NoError(t, err)
	c.PostChannel = postChannel
	go c.Run(ctx)

	// log.SetLevel(log.DebugLevel)

	type testCaseStruct struct {
		update *poller.Update
		output []string
	}
	var testCases = []testCaseStruct{
		{
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}, 2: {ID: 2, Name: "bar"}},
				ZoneInfo: map[int]tado.ZoneInfo{
					1: {
						Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
						SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 19.9}},
					},
					2: {
						Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
						SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 21.0}},
						Overlay: tado.ZoneInfoOverlay{
							Type:        "MANUAL",
							Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 22.0}},
							Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
						},
					},
				},
			},
			output: []string{"foo: 19.9ºC (target: 20.0)", "bar: 21.0ºC (target: 22.0, MANUAL)"},
		},
		{
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}, 2: {ID: 2, Name: "bar"}},
				ZoneInfo: map[int]tado.ZoneInfo{
					1: {
						// Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 21.0}},
						SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 17.0}},
						Overlay: tado.ZoneInfoOverlay{
							Type:        "MANUAL",
							Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
							Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
						},
					},
					2: {
						Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
						SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 21.0}},
						Overlay: tado.ZoneInfoOverlay{
							Type:        "MANUAL",
							Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 22.0}},
							Termination: tado.ZoneInfoOverlayTermination{Type: "TIMER", DurationInSeconds: 300},
						},
					},
				},
			},
			output: []string{"foo: 17.0ºC (off)", "bar: 21.0ºC (target: 22.0, MANUAL for 5m0s)"},
		},
	}

	for index, testCase := range testCases {
		c.Update <- testCase.update

		assert.Eventually(t, func() bool {
			msg := c.ReportRooms(ctx)

			if len(msg) != 1 {
				return false
			}

			lines := strings.Split(msg[0].Text, "\n")

			if len(lines) != len(testCase.output) {
				return false
			}

			for index, line := range lines {
				if line != testCase.output[index] {
					return false
				}
			}

			return true
		}, 1*time.Second, 10*time.Millisecond, index)
	}
}

func TestController_SetRoom(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mockapi.MockAPI{}
	p := poller.New(server)
	go p.Run(ctx, 10*time.Millisecond)
	postChannel := make(slackbot.PostChannel)
	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: nil}, nil, p)

	assert.NoError(t, err)
	c.PostChannel = postChannel
	p.Register <- c.Update
	go c.Run(ctx)

	var attachments []slack.Attachment

	// wait for poller to send an update to controller, so it knows about the rooms
	assert.Eventually(t, func() bool {
		attachments = c.ReportRooms(ctx)
		return len(attachments) == 1 && attachments[0].Text != "no rooms found"

	}, 500*time.Millisecond, 10*time.Millisecond)

	type TestCaseStruct struct {
		Args  []string
		Color string
		Text  string
	}

	var testCases = []TestCaseStruct{
		{
			Args:  []string{},
			Color: "bad",
			Text:  "invalid command: missing room name",
		},
		{
			Args:  []string{"notaroom"},
			Color: "bad",
			Text:  "invalid command: invalid room name",
		},
		{
			Args:  []string{"bar"},
			Color: "bad",
			Text:  "invalid command: missing target temperature",
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
			Args:  []string{"bar", "25.0"},
			Color: "good",
			Text:  "Setting target temperature for bar to 25.0ºC",
		},
		{
			Args:  []string{"bar", "25.0", "5m"},
			Color: "good",
			Text:  "Setting target temperature for bar to 25.0ºC for 5m0s",
		},
	}

	for _, testCase := range testCases {
		attachments = c.SetRoom(ctx, testCase.Args...)

		assert.Len(t, attachments, 1)
		assert.Equal(t, testCase.Color, attachments[0].Color)
		assert.Empty(t, attachments[0].Title)
		assert.Equal(t, testCase.Text, attachments[0].Text)
	}

	assert.Eventually(t, func() bool {
		zoneInfo, _ := c.API.GetZoneInfo(ctx, 2)
		return zoneInfo.GetState() == tado.ZoneStateTemporaryManual
	}, 500*time.Millisecond, 10*time.Millisecond)
}
