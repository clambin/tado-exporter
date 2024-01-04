package bot

import (
	"context"
	"github.com/clambin/tado"
	mocks2 "github.com/clambin/tado-exporter/internal/controller/bot/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"log/slog"
	"strconv"
	"testing"
	"time"
)

func TestBot_Run(t *testing.T) {
	api := mocks2.NewTadoSetter(t)
	b := mocks2.NewSlackBot(t)
	b.EXPECT().Register(mock.AnythingOfType("string"), mock.Anything)

	ch := make(chan poller.Update)
	p := mockPoller.NewPoller(t)
	p.EXPECT().Subscribe().Return(ch).Once()
	p.EXPECT().Unsubscribe(ch).Return().Once()

	c := New(api, b, p, nil, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- c.Run(ctx) }()

	ch <- poller.Update{}

	assert.Eventually(t, func() bool {
		c.lock.RLock()
		defer c.lock.RUnlock()
		return c.updated
	}, time.Second, 10*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestExecutor_ReportRules(t *testing.T) {
	api := mocks2.NewTadoSetter(t)
	bot := mocks2.NewSlackBot(t)
	bot.EXPECT().Register(mock.AnythingOfType("string"), mock.Anything)

	controller := mocks2.NewController(t)
	controller.EXPECT().ReportTasks().Return(nil).Once()

	c := New(api, bot, nil, controller, slog.Default())

	ctx := context.Background()
	attachments := c.ReportRules(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "no rules have been triggered", attachments[0].Text)

	controller.EXPECT().ReportTasks().Return([]string{"foo"}).Once()
	attachments = c.ReportRules(ctx)
	require.Len(t, attachments, 1)
	assert.Equal(t, "foo", attachments[0].Text)
}

func TestExecutor_SetRoom(t *testing.T) {
	api := mocks2.NewTadoSetter(t)
	bot := mocks2.NewSlackBot(t)
	bot.EXPECT().Register(mock.AnythingOfType("string"), mock.Anything)

	p := mockPoller.NewPoller(t)

	executor := New(api, bot, p, nil, slog.Default())
	executor.update = poller.Update{
		Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 22), testutil.ZoneInfoPermanentOverlay())},
	}
	executor.updated = true

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
			Args:  []string{"not-a-room", "auto"},
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
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			if testCase.Action {
				if testCase.Delete {
					api.EXPECT().DeleteZoneOverlay(mock.Anything, 1).Return(nil).Once()
				} else {
					api.EXPECT().SetZoneTemporaryOverlay(mock.Anything, 1, 25.0, testCase.Duration).Return(nil).Once()
				}
				p.EXPECT().Refresh().Once()
			}

			attachments := executor.SetRoom(context.Background(), testCase.Args...)

			require.Len(t, attachments, 1, index)
			assert.Equal(t, testCase.Color, attachments[0].Color, index)
			assert.Empty(t, attachments[0].Title, index)
			assert.Equal(t, testCase.Text, attachments[0].Text, index)
		})
	}
}

func TestExecutor_DoRefresh(t *testing.T) {
	api := mocks2.NewTadoSetter(t)
	bot := mocks2.NewSlackBot(t)
	bot.EXPECT().Register(mock.AnythingOfType("string"), mock.Anything)

	p := mockPoller.NewPoller(t)
	p.EXPECT().Refresh()

	c := New(api, bot, p, nil, slog.Default())
	c.DoRefresh(context.Background())
}

func TestExecutor_ReportRooms(t *testing.T) {
	api := mocks2.NewTadoSetter(t)
	bot := mocks2.NewSlackBot(t)
	bot.EXPECT().Register(mock.AnythingOfType("string"), mock.Anything)

	c := New(api, bot, nil, nil, slog.Default())

	attachments := c.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Empty(t, attachments[0].Title)
	assert.Equal(t, "no updates yet. please check back later", attachments[0].Text)

	c.update = poller.Update{
		Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(22.0, 18.0), testutil.ZoneInfoPermanentOverlay())},
	}
	c.updated = true

	attachments = c.ReportRooms(context.Background())
	require.Len(t, attachments, 1)
	assert.Equal(t, "rooms:", attachments[0].Title)
	assert.Equal(t, "foo: 22.0ºC (target: 18.0, MANUAL)", attachments[0].Text)

}

func TestExecutor_ReportUsers(t *testing.T) {
	api := mocks2.NewTadoSetter(t)
	bot := mocks2.NewSlackBot(t)
	bot.EXPECT().Register(mock.AnythingOfType("string"), mock.Anything)

	c := New(api, bot, nil, nil, slog.Default())

	testCases := []struct {
		name    string
		update  poller.Update
		updated bool
		want    slack.Attachment
	}{
		{
			name: "no update yet",
			//update: nil,
			want: slack.Attachment{Color: "bad", Text: "no update yet. please check back later"},
		},
		{
			name: "home",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(true))},
			},
			updated: true,
			want:    slack.Attachment{Title: "users:", Text: "foo: home"},
		},
		{
			name: "away",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo", testutil.Home(false))},
			},
			updated: true,
			want:    slack.Attachment{Title: "users:", Text: "foo: away"},
		},
		{
			name: "unknown",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{10: testutil.MakeMobileDevice(10, "foo")},
			},
			updated: true,
			want:    slack.Attachment{Title: "users:", Text: "foo: unknown"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			c.update = tt.update
			c.updated = tt.updated

			attachments := c.ReportUsers(context.Background())
			require.Len(t, attachments, 1)
			assert.Equal(t, tt.want, attachments[0])
			//assert.Equal(t, "users:", attachments[0].Title)
			//assert.Equal(t, tt.expected, attachments[0].Text)
		})
	}
}
