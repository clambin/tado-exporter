package bot

import (
	"context"
	"errors"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestBot_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	api := mocks.NewTadoClient(t)
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))
	s.EXPECT().Run(ctx).Return(nil)

	ch := make(chan poller.Update)
	p := mockPoller.NewPoller(t)
	p.EXPECT().Subscribe().Return(ch).Once()
	p.EXPECT().Unsubscribe(ch).Return().Once()

	b := New(api, s, p, nil, slog.Default())

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

func TestBot_ReportRules(t *testing.T) {
	api := mocks.NewTadoClient(t)
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			api := mocks.NewTadoClient(t)
			s := mocks.NewSlackBot(t)
			s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

			p := mockPoller.NewPoller(t)
			bot := New(api, s, p, nil, slog.Default())
			bot.update = poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP(tado.HomeId(1))},
				Zones:    []poller.Zone{{Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("foo")}}},
			}
			bot.updated = true

			if tt.action {
				if tt.del {
					api.EXPECT().
						DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), 10).
						Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK)}}, nil).
						Once()
				} else {
					api.EXPECT().
						SetZoneOverlayWithResponse(ctx, tado.HomeId(1), 10, mock.Anything).
						RunAndReturn(func(ctx context.Context, homeId int64, zoneId int, overlay tado.ZoneOverlay, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
							if *overlay.Setting.Temperature.Celsius != 25 {
								return nil, errors.New("invalid temperature")
							}
							if tt.duration > 0 {
								if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeTIMER || *overlay.Termination.DurationInSeconds != int(tt.duration.Seconds()) {
									return nil, errors.New("invalid termination")
								}
							} else {
								if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
									return nil, errors.New("invalid termination")
								}
							}
							return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK)}}, nil
						}).
						Once()
				}
				p.EXPECT().Refresh().Once()
			}

			attachments := bot.SetRoom(ctx, tt.args...)
			assert.Equal(t, tt.want, attachments)
		})
	}
}

func TestBot_SetHome(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			api := mocks.NewTadoClient(t)
			s := mocks.NewSlackBot(t)
			s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

			p := mockPoller.NewPoller(t)
			if tt.action != actionNone && tt.err == nil {
				p.EXPECT().Refresh()
			}

			bot := New(api, s, p, nil, slog.Default())
			bot.setUpdate(poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
			})

			switch tt.action {
			case actionNone:
			case actionHome:
				api.EXPECT().
					SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.Anything).
					RunAndReturn(func(_ context.Context, zoneId int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.HOME || zoneId != 1 {
							return nil, errors.New("invalid arg")
						}
						return nil, tt.err
					}).
					Once()
			case actionAway:
				api.EXPECT().
					SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.Anything).
					RunAndReturn(func(_ context.Context, zoneId int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.AWAY || zoneId != 1 {
							return nil, errors.New("invalid arg")
						}
						return nil, tt.err
					}).
					Once()
			case actionAuto:
				api.EXPECT().
					DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
					Return(nil, tt.err)
			}

			attachments := bot.SetHome(ctx, tt.args...)
			assert.Equal(t, tt.want, attachments)
		})
	}
}

func TestBot_DoRefresh(t *testing.T) {
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

	p := mockPoller.NewPoller(t)
	p.EXPECT().Refresh()

	b := New(nil, s, p, nil, slog.Default())
	b.DoRefresh(context.Background())
}

func TestBot_ReportRooms(t *testing.T) {
	s := mocks.NewSlackBot(t)
	s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

	b := New(nil, s, nil, nil, slog.Default())

	attachments := b.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Empty(t, attachments[0].Title)
	assert.Equal(t, "no updates yet. please check back later", attachments[0].Text)

	b.update = poller.Update{
		Zones: poller.Zones{
			{
				Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
				ZoneState: tado.ZoneState{
					Setting:          &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](18.0)}},
					Overlay:          &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
					SensorDataPoints: &oapi.SensorDataPoint,
				},
			},
		},
	}
	b.updated = true

	attachments = b.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Equal(t, "room: 21.0ºC (target: 18.0, MANUAL)", attachments[0].Text)

	b.update.Zones[0].Setting.Temperature = nil
	attachments = b.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Equal(t, "room: 21.0ºC (off)", attachments[0].Text)
}

func TestBot_ReportUsers(t *testing.T) {
	tests := []struct {
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
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
				},
			},
			updated: true,
			want:    []slack.Attachment{{Title: "users:", Text: "A: home"}},
		},
		{
			name: "away",
			update: poller.Update{
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			updated: true,
			want:    []slack.Attachment{{Title: "users:", Text: "A: away"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := mocks.NewTadoClient(t)
			s := mocks.NewSlackBot(t)
			s.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))

			b := New(api, s, nil, nil, slog.Default())

			if tt.updated {
				b.setUpdate(tt.update)
			}

			attachments := b.ReportUsers(context.Background())
			assert.Equal(t, tt.want, attachments)
		})
	}
}
