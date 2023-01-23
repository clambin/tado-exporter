package zonemanager

import (
	"context"
	"github.com/clambin/tado"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/poller"
	mocks3 "github.com/clambin/tado-exporter/poller/mocks"
	"github.com/clambin/tado/mocks"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

var (
	config = rules.ZoneConfig{
		Zone: "foo",
		Rules: []rules.RuleConfig{
			{
				Kind:  rules.AutoAway,
				Delay: 2 * time.Hour,
				Users: []string{"foo"},
			},
			{
				Kind:  rules.LimitOverlay,
				Delay: time.Hour,
			},
		},
	}

	testCases = []struct {
		name         string
		update       *poller.Update
		current      tado.ZoneState
		next         rules.NextState
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
			next: rules.NextState{
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
			next: rules.NextState{
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
			next: rules.NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateAuto,
				Delay:        time.Hour,
				ActionReason: "manual temperature setting detected",
				CancelReason: "room is now in auto mode",
			},
		},
		{
			name: "no action #2",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			current: tado.ZoneStateAuto,
			next: rules.NextState{
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
			next: rules.NextState{
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
			next: rules.NextState{
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
	a := mocks.NewAPI(t)
	b := slackbot.NewSlackBot(t)
	p := mocks3.NewPoller(t)
	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()
	m := New(a, p, b, config)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Run(ctx)
		wg.Done()
	}()

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.call != "" {
				a.On(tt.call, tt.args...).Return(nil).Once()
			}

			var wg2 sync.WaitGroup
			if tt.notification != "" {
				wg2.Add(1)
				b.On("Send", "", mock.AnythingOfType("[]slack.Attachment")).Run(func(args mock.Arguments) {
					defer wg2.Done()
					require.Len(t, args, 2)
					attachments, ok := args[1].([]slack.Attachment)
					require.True(t, ok)
					require.Len(t, attachments, 1)
					assert.Equal(t, tt.notification, attachments[0].Text)
				}).Return(nil).Once()
			}
			ch <- tt.update
			wg2.Wait()
		})
	}

	cancel()
	wg.Wait()
}

func TestManager_Scheduled(t *testing.T) {
	a := mocks.NewAPI(t)
	p := mocks3.NewPoller(t)
	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()
	m := New(a, p, nil, config)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Run(ctx)
		wg.Done()
	}()

	ch <- &poller.Update{
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
	assert.Equal(t, tado.ZoneStateAuto, state.State)

	var mgrs Managers = []*Manager{m}
	states := mgrs.GetScheduled()
	require.Len(t, states, 1)
	assert.Equal(t, tado.ZoneStateAuto, states[0].State)

	cancel()
	wg.Wait()
}

func TestManagers_ReportTasks(t *testing.T) {
	c := mocks.NewAPI(t)
	p := mocks3.NewPoller(t)
	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()
	m := New(c, p, nil, config)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		m.Run(ctx)
		wg.Done()
	}()

	mgrs := Managers{m}
	_, ok := mgrs.ReportTasks()
	assert.False(t, ok)

	ch <- &poller.Update{
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
