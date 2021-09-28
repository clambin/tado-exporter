package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	slackMock "github.com/clambin/tado-exporter/slackbot/mocks"
	tadoMock "github.com/clambin/tado/mocks"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var (
	updateZones = map[int]tado.Zone{
		1: {ID: 1, Name: "foo"},
	}
	updateMobileDeviceUserHome = map[int]tado.MobileDevice{
		1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
	}
	updateMobileDeviceUserAway = map[int]tado.MobileDevice{
		1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
	}
	updateZoneInfoAuto = map[int]tado.ZoneInfo{
		1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
	}
	updateZoneInfoManual = map[int]tado.ZoneInfo{
		1: {
			Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22.0}, Humidity: tado.Percentage{Percentage: 75.0}},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: 18.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{
					Type: "MANUAL",
				},
			},
		},
	}
	updateZoneInfoOff = map[int]tado.ZoneInfo{
		1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}, Overlay: tado.ZoneInfoOverlay{
			Type: "MANUAL",
			Setting: tado.ZoneInfoOverlaySetting{
				Type:        "HEATING",
				Power:       "OFF",
				Temperature: tado.Temperature{Celsius: 5.0},
			},
			Termination: tado.ZoneInfoOverlayTermination{
				Type: "MANUAL",
			},
		}},
	}

	updateZoneInOverlay = &poller.Update{
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoManual,
	}
)

func BenchmarkController_Run(b *testing.B) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "foo",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := &tadoMock.API{}
	api.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(nil)

	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in foo", "moving to auto mode in 0s").
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in foo", "moving to auto mode").
		Return(nil)

	c := controller.New(api, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	go c.Run(ctx, 10*time.Millisecond)

	for i := 0; i < 1000; i++ {
		c.Update(updateZoneInOverlay)
	}

	// time.Sleep(25 * time.Millisecond)
	// mock.AssertExpectationsForObjects(b, api, bot)
}

func TestController_LimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName:     "foo",
		LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: 20 * time.Millisecond},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := &tadoMock.API{}
	api.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(nil).
		Once()

	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in foo", "moving to auto mode in 0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "manual temperature setting detected in foo", "moving to auto mode").
		Return(nil).
		Once()

	c := controller.New(api, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)

	log.SetLevel(log.DebugLevel)

	go c.Run(ctx, 10*time.Millisecond)
	c.Updates <- updateZoneInOverlay

	time.Sleep(100 * time.Millisecond)
	mock.AssertExpectationsForObjects(t, api, bot)
}

func TestController_RevertLimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "foo",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &tadoMock.API{}

	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)

	c := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)

	log.SetLevel(log.DebugLevel)

	go c.Run(ctx, 10*time.Millisecond)

	bot.On("Send", "", "good", "manual temperature setting detected in foo", "moving to auto mode in 20m0s").
		Return(nil).
		Once()

	c.Updates <- updateZoneInOverlay
	c.Updates <- &poller.Update{
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoAuto,
	}

	time.Sleep(100 * time.Millisecond)

	mock.AssertExpectationsForObjects(t, server, bot)
}

func TestController_NightTime(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "foo",
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    23,
				Minutes: 30,
			},
		},
	}}

	server := &tadoMock.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in foo", mock.AnythingOfType("string")).
		Return(nil).
		Once()

	c := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)

	c.Update(updateZoneInOverlay)

	server.AssertExpectations(t)
}

func TestController_AutoAway(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 1,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   20 * time.Millisecond,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}},
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &tadoMock.API{}

	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).Return(nil)

	c := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)

	log.SetLevel(log.DebugLevel)
	go c.Run(ctx, 10*time.Millisecond)

	server.
		On("SetZoneOverlay", mock.Anything, 1, 5.0).
		Return(nil).
		Once()
	bot.On("Send", "", "good", "foo: foo is away", "switching off heating in 0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "foo: foo is away", "switching off heating").
		Return(nil).
		Once()

	// user is away & room in auto mode
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoAuto,
	}

	time.Sleep(100 * time.Millisecond)

	// user is away & room heating is off
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoOff,
	}

	server.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(nil).
		Once()
	bot.On("Send", "", "good", "foo: foo is home", "moving to auto mode").
		Return(nil).
		Once()

	// user comes home
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoOff,
	}

	time.Sleep(100 * time.Millisecond)

	mock.AssertExpectationsForObjects(t, server, bot)
}

func TestController_Combined(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 1,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   10 * time.Millisecond,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}},
		},
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
		NightTime: configuration.ZoneNightTime{
			Enabled: false,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    01,
				Minutes: 30,
				Seconds: 30,
			},
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &tadoMock.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).Return(nil)

	c := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	go c.Run(ctx, 10*time.Millisecond)

	server.On("SetZoneOverlay", mock.Anything, 1, 5.0).Return(nil).Once()
	bot.On("Send", "", "good", "foo: foo is away", "switching off heating in 0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "foo: foo is away", "switching off heating").
		Return(nil).
		Once()

	// user is away
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoAuto,
	}

	time.Sleep(100 * time.Millisecond)

	server.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(nil).
		Once()
	bot.On("Send", "", "good", "foo: foo is home", "moving to auto mode").
		Return(nil).
		Once()

	// user comes home
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoOff,
	}

	time.Sleep(100 * time.Millisecond)

	bot.On("Send", "", "good", "manual temperature setting detected in foo", "moving to auto mode in 20m0s").
		Return(nil).
		Once()

	// user is home & room set to manual
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoManual,
	}

	time.Sleep(100 * time.Millisecond)

	// report should say that a rule is triggered
	msg := c.ReportRules(ctx)
	require.Len(t, msg, 1)
	assert.Equal(t, "foo: moving to auto mode in 20m0s", msg[0].Text)

	mock.AssertExpectationsForObjects(t, server, bot)
}

func TestController_ReplacedTask(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 1,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   20 * time.Minute,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}},
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

	server := &tadoMock.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).Return(nil)

	c := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	go c.Run(ctx, 10*time.Millisecond)

	// user is home. room in manual, with night time configured
	bot.On("Send", "", "good", "manual temperature setting detected in foo", mock.AnythingOfType("string")).
		Return(nil).
		Once()
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoManual,
	}

	time.Sleep(100 * time.Millisecond)

	// user leaves
	bot.On("Send", "", "good", "foo: foo is away", "switching off heating in 20m0s").
		Return(nil).
		Once()
	c.Updates <- &poller.Update{
		UserInfo: updateMobileDeviceUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneInfoManual,
	}

	time.Sleep(100 * time.Millisecond)

	// report should say that a rule is triggered
	msg := c.ReportRules(ctx)
	require.Len(t, msg, 1)
	assert.Equal(t, "foo: switching off heating in 20m0s", msg[0].Text)

	mock.AssertExpectationsForObjects(t, server, bot)
}
