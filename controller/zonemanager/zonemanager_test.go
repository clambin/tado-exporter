package zonemanager

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	mocks2 "github.com/clambin/tado-exporter/pkg/slackbot/mocks"
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

var (
	config = configuration.ZoneConfig{
		ZoneID:   1,
		ZoneName: "foo",
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   2 * time.Hour,
			Users: []configuration.ZoneUser{
				{MobileDeviceID: 10, MobileDeviceName: "foo"},
			},
		},
		LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
		//NightTime:    configuration.ZoneNightTime{Enabled: true, Time: configuration.ZoneNightTimeTimestamp{Hour: 23, Minutes: 30}},
	}

	testCases = []struct {
		name         string
		update       *poller.Update
		current      tado.ZoneState
		next         NextState
		call         string
		args         []interface{}
		notification string
	}{
		{
			name: "no action",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			current: tado.ZoneStateAuto,
			next: NextState{
				ZoneID:   1,
				ZoneName: "foo",
				State:    tado.ZoneStateAuto,
			},
		},
		{
			name: "manual temp setting",
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			current: tado.ZoneStateManual,
			next: NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateAuto,
				Delay:        time.Hour,
				ActionReason: "manual temperature setting detected",
				CancelReason: "room is now in auto mode",
			},
			notification: "moving to auto mode in 1h0m0s",
		},
		{
			name: "manual temp setting #2",
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			current: tado.ZoneStateManual,
			next: NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateAuto,
				Delay:        time.Hour,
				ActionReason: "manual temperature setting detected",
				CancelReason: "room is now in auto mode",
			},
		},
		{
			name: "no action",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			current: tado.ZoneStateAuto,
			next: NextState{
				ZoneID:   1,
				ZoneName: "foo",
				State:    tado.ZoneStateAuto,
			},
			notification: "cancel moving to auto mode",
		},
		{
			name: "user away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			current: tado.ZoneStateAuto,
			next: NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateOff,
				Delay:        2 * time.Hour,
				ActionReason: "foo is away",
				CancelReason: "foo is home",
			},
			notification: "switching off heating in 2h0m0s",
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			current: tado.ZoneStateOff,
			next: NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateAuto,
				Delay:        0,
				ActionReason: "foo is home",
				CancelReason: "foo is away",
			},
			call:         "DeleteZoneOverlay",
			args:         []interface{}{mock.AnythingOfType("*context.cancelCtx"), 1},
			notification: "moving to auto mode",
		},
	}
)

func TestManager_Run(t *testing.T) {
	c := &mocks.API{}
	postChannel := make(slackbot.PostChannel, 10)
	b := &mocks2.SlackBot{}
	b.On("GetPostChannel").Return(postChannel)
	p := &mocks3.Poller{}
	p.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	p.On("Unregister", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	m := New(c, p, b, config)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Run(ctx, 10*time.Millisecond)
		wg.Done()
	}()

	time.Sleep(20 * time.Millisecond)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.call != "" {
				c.On(tt.call, tt.args...).Return(nil).Once()
			}
			wg2 := sync.WaitGroup{}
			wg2.Add(1)
			go func() {
				m.Updates <- tt.update
				wg2.Done()
			}()

			if tt.notification != "" {
				msg := <-postChannel
				require.Len(t, msg, 1, tt.name)
				assert.Equal(t, tt.notification, msg[0].Text, tt.name)
			}

			wg2.Wait()
		})
	}

	cancel()
	wg.Wait()

	mock.AssertExpectationsForObjects(t, c)
}

func TestManager_Scheduled(t *testing.T) {
	c := &mocks.API{}
	p := &mocks3.Poller{}
	p.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	p.On("Unregister", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	m := New(c, p, nil, config)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Run(ctx, 10*time.Millisecond)
		wg.Done()
	}()

	m.Updates <- &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	assert.Eventually(t, func() bool {
		_, scheduled := m.Scheduled()
		return scheduled
	}, time.Second, 10*time.Millisecond)

	state, scheduled := m.Scheduled()
	require.True(t, scheduled)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), state.State)

	var mgrs Managers = []*Manager{m}
	states := mgrs.GetScheduled()
	require.Len(t, states, 1)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), states[0].State)

	cancel()
	wg.Wait()
}

func TestManagers_ReportTasks(t *testing.T) {
	c := &mocks.API{}
	p := &mocks3.Poller{}
	p.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	p.On("Unregister", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	m := New(c, p, nil, config)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Run(ctx, 10*time.Millisecond)
		wg.Done()
	}()

	mgrs := Managers{m}
	_, ok := mgrs.ReportTasks()
	assert.False(t, ok)

	m.Updates <- &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	assert.Eventually(t, func() bool {
		_, scheduled := m.Scheduled()
		return scheduled
	}, time.Second, 10*time.Millisecond)

	tasks, ok := mgrs.ReportTasks()
	assert.True(t, ok)
	require.NotEmpty(t, tasks)
	assert.Contains(t, tasks[0], "foo: moving to auto mode in")

	cancel()
	wg.Wait()
}
