package zonemanager

import (
	"bytes"
	"context"
	"github.com/clambin/tado"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules/mocks"
	"github.com/clambin/tado-exporter/poller"
	mockPoller "github.com/clambin/tado-exporter/poller/mocks"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
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
)

func TestManager_Run(t *testing.T) {
	tests := []struct {
		name         string
		update       *poller.Update
		call         string
		args         []interface{}
		notification string
	}{
		{
			name: "no action",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
		},
		{
			name: "manual temp setting",
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {
					Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}},
					Overlay: tado.ZoneInfoOverlay{
						Type:        "MANUAL",
						Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
					}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			notification: "moving to auto mode in 1h0m0s",
		},
		{
			name: "manual temp setting #2",
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {
					Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}},
					Overlay: tado.ZoneInfoOverlay{
						Type:        "MANUAL",
						Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
					}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
		},
		{
			name: "no action #2",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			notification: "canceling moving to auto mode",
		},
		{
			name: "user away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			notification: "switching off heating in 2h0m0s",
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: {
					Setting: tado.ZonePowerSetting{Power: "OFF"},
					Overlay: tado.ZoneInfoOverlay{Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"}}}},
				UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			notification: "canceling switching off heating",
		},
	}

	a := mocks.NewTadoSetter(t)
	b := slackbot.NewSlackBot(t)
	p := mockPoller.NewPoller(t)

	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()

	m := New(a, p, b, config)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() { defer wg.Done(); m.Run(ctx) }()

	for _, tt := range tests {
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
					assert.Contains(t, attachments[0].Title, tt.notification)
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
	a := mocks.NewTadoSetter(t)
	p := mockPoller.NewPoller(t)
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
		ZoneInfo: map[int]tado.ZoneInfo{1: {
			Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}},
			Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZonePowerSetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}},
		UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	assert.Eventually(t, func() bool {
		_, scheduled := m.GetScheduled()
		return scheduled
	}, time.Second, 10*time.Millisecond)

	state, scheduled := m.GetScheduled()
	require.True(t, scheduled)
	assert.Equal(t, rules.ZoneState{Overlay: tado.NoOverlay}, state.State)

	var mgrs Managers = []*Manager{m}
	states := mgrs.GetScheduled()
	require.Len(t, states, 1)
	assert.Equal(t, rules.ZoneState{Overlay: tado.NoOverlay}, states[0].State)

	cancel()
	wg.Wait()
}

func TestManagers_ReportTasks(t *testing.T) {
	a := mocks.NewTadoSetter(t)
	p := mockPoller.NewPoller(t)
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

	mgrs := Managers{m}
	_, ok := mgrs.ReportTasks()
	assert.False(t, ok)

	ch <- &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: {
			Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}},
			Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZonePowerSetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}},
		UserInfo: map[int]tado.MobileDevice{10: {ID: 10, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	assert.Eventually(t, func() bool {
		_, scheduled := m.GetScheduled()
		return scheduled
	}, time.Second, 10*time.Millisecond)

	tasks, ok := mgrs.ReportTasks()
	assert.True(t, ok)
	require.NotEmpty(t, tasks)
	assert.Contains(t, tasks[0], "foo: moving to auto mode in")

	cancel()
	wg.Wait()
}

func Test_zoneLogger_LogValue(t *testing.T) {
	tests := []struct {
		name     string
		zoneInfo tado.ZoneInfo
		want     string
	}{
		{
			name:     "auto mode (on)",
			zoneInfo: tado.ZoneInfo{Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}}},
			want:     `level=INFO msg=zone z.settings.power=ON z.settings.temperature=20`,
		},
		{
			name:     "auto mode (off)",
			zoneInfo: tado.ZoneInfo{Setting: tado.ZonePowerSetting{Power: "OFF"}},
			want:     `level=INFO msg=zone z.settings.power=OFF`,
		},
		{
			name: "manual (on)",
			zoneInfo: tado.ZoneInfo{
				Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 20.0}},
				Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL", TypeSkillBasedApp: "MANUAL"},
				},
			},
			want: `level=INFO msg=zone z.settings.power=ON z.settings.temperature=20 z.overlay.termination.type=MANUAL z.overlay.termination.subtype=MANUAL`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := zoneLogger(tt.zoneInfo)

			out := bytes.NewBufferString("")
			opt := slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				// Remove time from the output for predictable test output.
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			}}
			l := slog.New(opt.NewTextHandler(out))

			l.Log(context.Background(), slog.LevelInfo, "zone", "z", z)
			assert.Equal(t, tt.want+"\n", out.String())
		})
	}
}
