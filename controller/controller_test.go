package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	mocks2 "github.com/clambin/tado-exporter/slackbot/mocks"
	"github.com/clambin/tado/mocks"
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
		2: {ID: 2, Name: "bar"},
	}
	updateUserHome = map[int]tado.MobileDevice{
		1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
		2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
	}
	updateUserAway = map[int]tado.MobileDevice{
		1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
		2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
	}
	updateZoneAuto = map[int]tado.ZoneInfo{
		1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
		2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
	}
	updateZoneManual = map[int]tado.ZoneInfo{
		1: {
			Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22.0}, Humidity: tado.Percentage{Percentage: 75.0}}},
		2: {
			Setting:          tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22.0}, Humidity: tado.Percentage{Percentage: 75.0}},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: 22.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{
					Type: "MANUAL",
				},
			},
		},
	}
	updateZoneOff = map[int]tado.ZoneInfo{
		1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}},
		2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.5}}, Overlay: tado.ZoneInfoOverlay{
			Type: "MANUAL",
			Setting: tado.ZoneInfoOverlaySetting{
				Type:        "HEATING",
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 5.0},
			},
			Termination: tado.ZoneInfoOverlayTermination{
				Type: "MANUAL",
			},
		}},
	}

	updateZoneInOverlay = &poller.Update{
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	}
)

func BenchmarkController_Run(b *testing.B) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mocks.API{}
	server.On("DeleteZoneOverlay", mock.Anything, 2).Return(nil)

	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode in 0s").
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode").
		Return(nil)

	c, _ := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	go c.Run(ctx)

	for i := 0; i < 1000; i++ {
		c.Update(ctx, &poller.Update{
			Zones:    updateZones,
			ZoneInfo: updateZoneManual,
		})
	}

	time.Sleep(25 * time.Millisecond)
	mock.AssertExpectationsForObjects(b, server, bot)
}

func TestController_LimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mocks.API{}
	server.On("DeleteZoneOverlay", mock.Anything, 2).Return(nil)

	bot := mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode in 0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode").
		Return(nil).
		Once()

	c, _ := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, &bot, nil)

	log.SetLevel(log.DebugLevel)

	c.Update(ctx, &poller.Update{
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	})
	go c.Run(ctx)

	time.Sleep(50 * time.Millisecond)
	server.AssertExpectations(t)
}

func TestController_RevertLimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &mocks.API{}

	bot := mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)

	c, _ := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, &bot, nil)

	log.SetLevel(log.DebugLevel)

	go c.Run(ctx)

	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode in 20m0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode").
		Return(nil).
		Once()

	c.Updates <- &poller.Update{
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	}

	time.Sleep(100 * time.Millisecond)

	bot.
		On("Send", "", "good", "resetting rule for bar", "").
		Return(nil).
		Once()

	c.Updates <- &poller.Update{
		Zones:    updateZones,
		ZoneInfo: updateZoneAuto,
	}

	time.Sleep(100 * time.Millisecond)

	server.AssertExpectations(t)
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

	server := &mocks.API{}
	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).
		Return(nil)
	bot.On("Send", "", "good", "manual temperature setting detected in bar", mock.AnythingOfType("string")).
		Return(nil).
		Once()

	c, _ := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)

	c.Update(ctx, updateZoneInOverlay)

	server.AssertExpectations(t)
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

	server := &mocks.API{}

	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).Return(nil)

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	assert.NoError(t, err)

	log.SetLevel(log.DebugLevel)
	go c.Run(ctx)

	server.On("SetZoneOverlay", mock.Anything, 2, 5.0).Return(nil).Once()
	bot.On("Send", "", "good", "bar: bar is away", "switching off heating in 0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "bar: bar is away", "switching off heating").
		Return(nil).
		Once()

	// user is away & room in auto mode
	c.Updates <- &poller.Update{
		UserInfo: updateUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneAuto,
	}

	time.Sleep(100 * time.Millisecond)

	// user is away & room heating is off
	c.Updates <- &poller.Update{
		UserInfo: updateUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneOff,
	}

	server.
		On("DeleteZoneOverlay", mock.Anything, 2).
		Return(nil).
		Once()
	bot.On("Send", "", "good", "bar: bar is home", "moving to auto mode").
		Return(nil).
		Once()

	// user comes home
	c.Updates <- &poller.Update{
		UserInfo: updateUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneOff,
	}

	time.Sleep(100 * time.Millisecond)

	mock.AssertExpectationsForObjects(t, server, bot)
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

	server := &mocks.API{}
	// server.On("DeleteZoneOverlay", mock.Anything, 2).Return(nil).Once()

	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).Return(nil)

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	assert.NoError(t, err)

	go c.Run(ctx)

	server.On("SetZoneOverlay", mock.Anything, 2, 5.0).Return(nil).Once()
	bot.On("Send", "", "good", "bar: bar is away", "switching off heating in 0s").
		Return(nil).
		Once()
	bot.On("Send", "", "good", "bar: bar is away", "switching off heating").
		Return(nil).
		Once()

	// user is away
	c.Updates <- &poller.Update{
		UserInfo: updateUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneAuto,
	}

	time.Sleep(100 * time.Millisecond)

	server.
		On("DeleteZoneOverlay", mock.Anything, 2).
		Return(nil).
		Once()
	bot.On("Send", "", "good", "bar: bar is home", "moving to auto mode").
		Return(nil).
		Once()

	// user comes home
	c.Updates <- &poller.Update{
		UserInfo: updateUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneOff,
	}

	time.Sleep(100 * time.Millisecond)

	bot.On("Send", "", "good", "manual temperature setting detected in bar", "moving to auto mode in 20m0s").
		Return(nil).
		Once()

	// user is home & room set to manual
	c.Updates <- &poller.Update{
		UserInfo: updateUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	}

	time.Sleep(100 * time.Millisecond)

	// report should say that a rule is triggered
	msg := c.ReportRules(ctx)
	require.Len(t, msg, 1)
	assert.Equal(t, "bar: moving to auto mode in 20m0s", msg[0].Text)

	mock.AssertExpectationsForObjects(t, server, bot)
}

func TestZoneManager_ReplacedTask(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   20 * time.Minute,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
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

	server := &mocks.API{}
	bot := &mocks2.SlackBot{}
	bot.On("RegisterCallback", mock.Anything, mock.Anything).Return(nil)

	c, err := controller.New(server, &configuration.ControllerConfiguration{Enabled: true, ZoneConfig: zoneConfig}, bot, nil)
	assert.NoError(t, err)

	go c.Run(ctx)

	// user is home. room in manual, with night time configured
	bot.On("Send", "", "good", "manual temperature setting detected in bar", mock.AnythingOfType("string")).
		Return(nil).
		Once()
	c.Updates <- &poller.Update{
		UserInfo: updateUserHome,
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	}

	time.Sleep(100 * time.Millisecond)

	// user leaves
	bot.On("Send", "", "good", "bar: bar is away", "switching off heating in 20m0s").
		Return(nil).
		Once()
	c.Updates <- &poller.Update{
		UserInfo: updateUserAway,
		Zones:    updateZones,
		ZoneInfo: updateZoneManual,
	}

	time.Sleep(100 * time.Millisecond)

	// report should say that a rule is triggered
	msg := c.ReportRules(ctx)
	require.Len(t, msg, 1)
	assert.Equal(t, "bar: switching off heating in 20m0s", msg[0].Text)

	mock.AssertExpectationsForObjects(t, server, bot)
}
