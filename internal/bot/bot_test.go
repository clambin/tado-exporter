package bot

import (
	"context"
	"errors"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestBot_Run(t *testing.T) {
	api := mocks.NewTadoSetter(t)
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

	ch := make(chan poller.Update)
	p := mockPoller.NewPoller(t)
	p.EXPECT().Subscribe().Return(ch).Once()
	p.EXPECT().Unsubscribe(ch).Return().Once()

	b := New(api, s, p, nil, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- b.Run(ctx) }()

	ch <- poller.Update{}

	assert.Eventually(t, func() bool {
		b.lock.RLock()
		defer b.lock.RUnlock()
		return b.updated
	}, time.Second, 10*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestExecutor_ReportRules(t *testing.T) {
	api := mocks.NewTadoSetter(t)
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

	controller := mocks.NewController(t)
	controller.EXPECT().ReportTasks().Return(nil).Once()

	b := New(api, s, nil, controller, slog.Default())

	ctx := context.Background()
	attachments := b.ReportRules(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "no rules have been triggered", attachments[0].Text)

	controller.EXPECT().ReportTasks().Return([]string{"foo"}).Once()
	attachments = b.ReportRules(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "foo", attachments[0].Text)
}

func TestExecutor_SetRoom(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     []slack.Attachment
		action   bool
		del      bool
		duration time.Duration
	}{
		{
			name: "empty",
			args: []string{},
			want: []slack.Attachment{{Color: "bad", Text: "invalid command: missing parameters\nUsage: set room <room> [auto|<temperature> [<duration>]"}},
		},
		{
			name: "missing parameters",
			args: []string{"foo"},
			want: []slack.Attachment{{Color: "bad", Text: "invalid command: missing parameters\nUsage: set room <room> [auto|<temperature> [<duration>]"}},
		},
		{
			name: "invalid parameters",
			args: []string{"foo", "25,0"},
			want: []slack.Attachment{{Color: "bad", Text: "invalid command: invalid target temperature: \"25,0\""}},
		},
		{
			name: "invalid duration",
			args: []string{"foo", "25.0", "invalid"},
			want: []slack.Attachment{{Color: "bad", Text: "invalid command: invalid duration: \"invalid\""}},
		},
		{
			name:   "set permanent",
			args:   []string{"foo", "25.0"},
			want:   []slack.Attachment{{Color: "good", Text: "Setting target temperature for foo to 25.0ºC"}},
			action: true,
		},
		{
			name:     "set temporary",
			args:     []string{"foo", "25.0", "5m"},
			want:     []slack.Attachment{{Color: "good", Text: "Setting target temperature for foo to 25.0ºC for 5m0s"}},
			action:   true,
			duration: 5 * time.Minute,
		},
		{
			name:   "auto mode",
			args:   []string{"foo", "auto"},
			want:   []slack.Attachment{{Color: "good", Text: "Setting foo to automatic mode"}},
			action: true,
			del:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			api := mocks.NewTadoSetter(t)
			s := mocks.NewSlackBot(t)
			s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

			p := mockPoller.NewPoller(t)
			executor := New(api, s, p, nil, slog.Default())
			executor.update = poller.Update{
				Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 22), testutil.ZoneInfoPermanentOverlay())},
			}
			executor.updated = true

			if tt.action {
				if tt.del {
					api.EXPECT().DeleteZoneOverlay(ctx, 1).Return(nil).Once()
				} else {
					api.EXPECT().SetZoneTemporaryOverlay(ctx, 1, 25.0, tt.duration).Return(nil).Once()
				}
				p.EXPECT().Refresh().Once()
			}

			attachments := executor.SetRoom(ctx, tt.args...)
			assert.Equal(t, tt.want, attachments)
		})
	}
}

func TestExecutor_SetHome(t *testing.T) {
	type action int
	const (
		actionNone action = iota
		actionHome
		actionAway
		actionAuto
	)
	tests := []struct {
		name   string
		args   []string
		action action
		err    error
		want   []slack.Attachment
	}{
		{
			name: "empty",
			args: []string{},
			want: []slack.Attachment{{Color: "bad", Text: "missing parameter\nUsage: set home [home|away|auto]"}},
		},
		{
			name: "invalid",
			args: []string{"foo"},
			want: []slack.Attachment{{Color: "bad", Text: "missing parameter\nUsage: set home [home|away|auto]"}},
		},
		{
			name:   "home",
			args:   []string{"home"},
			action: actionHome,
			want:   []slack.Attachment{{Color: "good", Text: "set home to home mode"}},
		},
		{
			name:   "away",
			args:   []string{"away"},
			action: actionAway,
			want:   []slack.Attachment{{Color: "good", Text: "set home to away mode"}},
		},
		{
			name:   "auto",
			args:   []string{"auto"},
			action: actionAuto,
			want:   []slack.Attachment{{Color: "good", Text: "set home to auto mode"}},
		},
		{
			name:   "auto (fail)",
			args:   []string{"auto"},
			action: actionAuto,
			err:    errors.New("fail"),
			want:   []slack.Attachment{{Color: "bad", Text: "failed: fail"}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			api := mocks.NewTadoSetter(t)
			s := mocks.NewSlackBot(t)
			s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

			p := mockPoller.NewPoller(t)
			if tt.action != actionNone && tt.err == nil {
				p.EXPECT().Refresh()
			}

			executor := New(api, s, p, nil, slog.Default())

			switch tt.action {
			case actionNone:
			case actionHome:
				api.EXPECT().SetHomeState(ctx, true).Return(tt.err)
			case actionAway:
				api.EXPECT().SetHomeState(ctx, false).Return(tt.err)
			case actionAuto:
				api.EXPECT().UnsetHomeState(ctx).Return(tt.err)
			}

			attachments := executor.SetHome(ctx, tt.args...)
			assert.Equal(t, tt.want, attachments)
		})
	}
}

func TestExecutor_DoRefresh(t *testing.T) {
	api := mocks.NewTadoSetter(t)
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

	p := mockPoller.NewPoller(t)
	p.EXPECT().Refresh()

	b := New(api, s, p, nil, slog.Default())
	b.DoRefresh(context.Background())
}

func TestExecutor_ReportRooms(t *testing.T) {
	api := mocks.NewTadoSetter(t)
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

	b := New(api, s, nil, nil, slog.Default())

	attachments := b.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Empty(t, attachments[0].Title)
	assert.Equal(t, "no updates yet. please check back later", attachments[0].Text)

	b.update = poller.Update{
		Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(22.0, 18.0), testutil.ZoneInfoPermanentOverlay())},
	}
	b.updated = true

	attachments = b.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Equal(t, "foo: 22.0ºC (target: 18.0, MANUAL)", attachments[0].Text)

}

func TestExecutor_ReportUsers(t *testing.T) {
	testCases := []struct {
		name    string
		update  poller.Update
		updated bool
		want    []slack.Attachment
	}{
		{
			name: "no update yet",
			//update: nil,
			want: []slack.Attachment{{Color: "bad", Text: "no update yet. please check back later"}},
		},
		{
			name: "home",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
			},
			updated: true,
			want:    []slack.Attachment{{Title: "users:", Text: "foo: home"}},
		},
		{
			name: "away",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(false))},
			},
			updated: true,
			want:    []slack.Attachment{{Title: "users:", Text: "foo: away"}},
		},
		{
			name: "unknown",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo")},
			},
			updated: true,
			want:    []slack.Attachment{{Title: "users:", Text: "foo: unknown"}},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := mocks.NewTadoSetter(t)
			s := mocks.NewSlackBot(t)
			s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

			b := New(api, s, nil, nil, slog.Default())

			b.update = tt.update
			b.updated = tt.updated

			attachments := b.ReportUsers(context.Background())
			assert.Equal(t, tt.want, attachments)
		})
	}
}
