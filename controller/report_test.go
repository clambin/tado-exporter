package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
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

	log.SetLevel(log.DebugLevel)

	msg := c.ReportRules()
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "no rules have been triggered", msg[0].Text)
	}

	// user is away
	c.Update <- &fakeUpdates[0]
	_ = <-postChannel

	msg = c.ReportRules()
	if assert.Len(t, msg, 1) {
		assert.Contains(t, msg[0].Text, "bar: switching off heating in ")
	}

	// user is home & room set to manual
	c.Update <- &fakeUpdates[2]
	msg = <-postChannel
	if assert.Len(t, msg, 1) {
		assert.Contains(t, msg[0].Text, "moving to auto mode in ")
	}

	msg = c.ReportRules()
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
	c, err := controller.New(
		server,
		&configuration.ControllerConfiguration{
			Enabled:    true,
			ZoneConfig: nil,
		},
		nil)

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
							Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 22.0}},
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
							Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 5.0}},
							Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
						},
					},
					2: {
						Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
						SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 21.0}},
						Overlay: tado.ZoneInfoOverlay{
							Type:        "MANUAL",
							Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 22.0}},
							Termination: tado.ZoneInfoOverlayTermination{Type: "AUTO"},
						},
					},
				},
			},
			output: []string{"foo: 17.0ºC (off)", "bar: 21.0ºC (target: 22.0, MANUAL)"},
		},
	}

	for _, testCase := range testCases {
		c.Update <- testCase.update

		assert.Eventually(t, func() bool {
			msg := c.ReportRooms()

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
		}, 1*time.Second, 10*time.Millisecond)
	}
}
