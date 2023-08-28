package zone_test

import (
	"context"
	"github.com/clambin/tado"
	slackbot "github.com/clambin/tado-exporter/internal/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestController_Run(t *testing.T) {
	tests := []struct {
		name         string
		update       *poller.Update
		args         []interface{}
		notification string
	}{
		{
			name: "no action",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
			},
		},
		{
			name: "manual temp setting",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
			},
			notification: "moving to auto mode in 1h0m0s",
		},
		{
			name: "manual temp setting #2",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
			},
		},
		{
			name: "no action #2",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
			},
			notification: "canceling moving to auto mode",
		},
		{
			name: "user away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(false))},
			},
			notification: "switching off heating in 2h0m0s",
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
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

	m := zone.New(a, p, b, config)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := make(chan []slack.Attachment)
			if tt.notification != "" {
				b.EXPECT().Send("", mock.AnythingOfType("[]slack.Attachment")).RunAndReturn(func(_ string, attachments []slack.Attachment) error {
					response <- attachments
					return nil
				}).Once()
			}
			ch <- tt.update
			if tt.notification != "" {
				messages := <-response
				require.Len(t, messages, 1)
				assert.Contains(t, messages[0].Title, tt.notification)
			}
		})
	}

	cancel()
	assert.NoError(t, <-errCh)
}

func TestController_Scheduled(t *testing.T) {
	a := mocks.NewTadoSetter(t)
	p := mockPoller.NewPoller(t)
	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()
	m := zone.New(a, p, nil, config)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	ch <- &poller.Update{
		Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(15, 18.5), testutil.ZoneInfoPermanentOverlay())},
		UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
	}

	assert.Eventually(t, func() bool {
		_, scheduled := m.GetScheduled()
		return scheduled
	}, time.Second, 10*time.Millisecond)

	state, scheduled := m.GetScheduled()
	require.True(t, scheduled)
	assert.Equal(t, rules.ZoneState{Overlay: tado.NoOverlay}, state.State)

	controllers := zone.Controllers([]*zone.Controller{m})
	states := controllers.GetScheduled()
	require.Len(t, states, 1)
	assert.Equal(t, rules.ZoneState{Overlay: tado.NoOverlay}, states[0].State)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestController_ReportTasks(t *testing.T) {
	a := mocks.NewTadoSetter(t)
	p := mockPoller.NewPoller(t)
	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()
	m := zone.New(a, p, nil, config)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	mgrs := zone.Controllers{m}
	_, ok := mgrs.ReportTasks()
	assert.False(t, ok)

	ch <- &poller.Update{
		Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(15, 18.5), testutil.ZoneInfoPermanentOverlay())},
		UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
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
	assert.NoError(t, <-errCh)
}