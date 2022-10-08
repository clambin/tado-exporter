package commands

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	slackMock "github.com/clambin/tado-exporter/pkg/slackbot/mocks"
	"github.com/clambin/tado-exporter/poller"
	mocks3 "github.com/clambin/tado-exporter/poller/mocks"
	"github.com/clambin/tado/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestManager_Run(t *testing.T) {
	api := &mocks.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	p := &mocks3.Poller{}
	p.On("Register", mock.AnythingOfType("chan *poller.Update")).Return().Once()
	p.On("Unregister", mock.AnythingOfType("chan *poller.Update")).Return().Once()
	c := New(api, bot, p, nil)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()

	c.updates <- &poller.Update{}

	assert.Eventually(t, func() bool {
		c.lock.RLock()
		defer c.lock.RUnlock()
		return c.update != nil
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()

	mock.AssertExpectationsForObjects(t, api, bot, p)
}

func TestController_Rules(t *testing.T) {
	api := &mocks.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	p := &mocks3.Poller{}
	p.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	p.On("Unregister", mock.AnythingOfType("chan *poller.Update")).Return(nil)

	cfg := configuration.ZoneConfig{
		ZoneID:   1,
		ZoneName: "foo",
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time:    configuration.ZoneNightTimeTimestamp{Hour: 23, Minutes: 30, Seconds: 0},
		},
	}

	mgrs := zonemanager.Managers{
		zonemanager.New(api, p, nil, cfg),
	}
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(len(mgrs))
	for _, mgr := range mgrs {
		go func(m *zonemanager.Manager) {
			m.Run(ctx, time.Hour)
			wg.Done()
		}(mgr)
	}

	c := New(api, bot, p, mgrs)

	attachments := c.ReportRules(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "no rules have been triggered", attachments[0].Text)

	mgrs[0].Updates <- &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
	}

	require.Eventually(t, func() bool {
		_, found := mgrs[0].Scheduled()
		return found
	}, time.Second, 10*time.Millisecond)

	attachments = c.ReportRules(ctx)
	require.Len(t, attachments, 1)
	assert.Contains(t, attachments[0].Text, "foo: moving to auto mode in")

	cancel()
	wg.Wait()

	mock.AssertExpectationsForObjects(t, api, bot, p)
}

func TestManager_SetRoom(t *testing.T) {
	api := &mocks.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	p := &mocks3.Poller{}

	c := New(api, bot, p, nil)

	c.update = &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: {
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22}},
			Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}},
	}

	var testCases = []struct {
		Args     []string
		Color    string
		Text     string
		Action   bool
		Delete   bool
		Duration time.Duration
	}{
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
				api.On("DeleteZoneOverlay", mock.Anything, 1).Return(nil).Once()
			} else {
				api.On("SetZoneOverlayWithDuration", mock.Anything, 1, 25.0, testCase.Duration).Return(nil).Once()
			}
			p.On("Refresh").Return(nil).Once()
		}

		attachments := c.SetRoom(context.Background(), testCase.Args...)

		require.Len(t, attachments, 1, index)
		assert.Equal(t, testCase.Color, attachments[0].Color, index)
		assert.Empty(t, attachments[0].Title, index)
		assert.Equal(t, testCase.Text, attachments[0].Text, index)
	}

	mock.AssertExpectationsForObjects(t, api, bot, p)
}

func TestManager_DoRefresh(t *testing.T) {
	api := &mocks.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	p := &mocks3.Poller{}
	p.On("Refresh").Return(nil)
	c := New(api, bot, p, nil)

	c.DoRefresh(context.Background())
}

func TestManager_ReportRooms(t *testing.T) {
	api := &mocks.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	c := New(api, bot, nil, nil)

	c.update = &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: {
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22}},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: 18.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}},
	}

	attachments := c.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Equal(t, "foo: 22.0ºC (target: 18.0, MANUAL)", attachments[0].Text)

}

func TestManager_ReportUsers(t *testing.T) {
	api := &mocks.API{}
	bot := &slackMock.SlackBot{}
	bot.On("RegisterCallback", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	c := New(api, bot, nil, nil)

	testCases := []struct {
		update   *poller.Update
		expected string
	}{
		{
			update: &poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					10: {
						ID:       10,
						Name:     "foo",
						Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
						Location: tado.MobileDeviceLocation{AtHome: true},
					},
				},
			},
			expected: "foo: home",
		},
		{
			update: &poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					10: {
						ID:       10,
						Name:     "foo",
						Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
						Location: tado.MobileDeviceLocation{AtHome: false},
					},
				},
			},
			expected: "foo: away",
		},
		{
			update: &poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					10: {
						ID:       10,
						Name:     "foo",
						Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false},
						Location: tado.MobileDeviceLocation{AtHome: false},
					},
				},
			},
			expected: "foo: unknown",
		},
	}

	for _, tt := range testCases {
		c.update = tt.update

		attachments := c.ReportUsers(context.Background())
		require.Len(t, attachments, 1)
		assert.Equal(t, "users:", attachments[0].Title)
		assert.Equal(t, tt.expected, attachments[0].Text)
	}

	mock.AssertExpectationsForObjects(t, api, bot)
}
