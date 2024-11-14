package bot

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func Test_shortcuts_dispatch(t *testing.T) {
	tests := []struct {
		name    string
		event   slack.InteractionCallback
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "invalid callbackID is rejected",
			event:   slack.InteractionCallback{CallbackID: "bar"},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "initiate shortcut",
			event: slack.InteractionCallback{
				Type:       slack.InteractionTypeShortcut,
				CallbackID: "foo",
			},
			want:    []string{"shortcut"},
			wantErr: assert.NoError,
		},
		{
			name: "update view",
			event: slack.InteractionCallback{
				Type: slack.InteractionTypeBlockActions,
				View: slack.View{CallbackID: "foo"},
			},
			want:    []string{"action"},
			wantErr: assert.NoError,
		},
		{
			name: "update view checks ActionID",
			event: slack.InteractionCallback{
				Type: slack.InteractionTypeBlockActions,
				View: slack.View{CallbackID: "bar"},
			},
			wantErr: assert.Error,
		},
		{
			name: "submit",
			event: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{CallbackID: "foo"},
			},
			want:    []string{"submit"},
			wantErr: assert.NoError,
		},
		{
			name: "submit checks ActionID",
			event: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{CallbackID: "bar"},
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h fakeHandler
			s := shortcuts{"foo": &h}

			err := s.dispatch(tt.event, nil)
			assert.Equal(t, tt.want, h.calls)
			tt.wantErr(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_setRoomShortcut_makeView(t *testing.T) {
	tests := []struct {
		name            string
		mode            string
		update          poller.Update
		wantInputBlocks []string
		wantZones       []string
	}{
		{
			name: "auto mode",
			mode: "auto",
			update: poller.Update{
				Zones: []poller.Zone{
					{Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("foo")}},
					{Zone: tado.Zone{Id: oapi.VarP(20), Name: oapi.VarP("bar")}},
				},
			},
			wantInputBlocks: []string{"zone", "mode", "channel"},
			wantZones:       []string{"foo", "bar"},
		},
		{
			name: "manual mode",
			mode: "manual",
			update: poller.Update{
				Zones: []poller.Zone{
					{Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("foo")}},
					{Zone: tado.Zone{Id: oapi.VarP(20), Name: oapi.VarP("bar")}},
				},
			},
			wantInputBlocks: []string{"zone", "mode", "temperature", "expiration", "channel"},
			wantZones:       []string{"foo", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setRoomShortcut{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

			req := h.makeView(tt.mode, tt.update)

			zones := make([]string, 0, len(tt.update.Zones))
			inputBlocks := make([]string, 0, len(req.Blocks.BlockSet))
			for _, block := range req.Blocks.BlockSet {
				if inputBlock, ok := block.(*slack.InputBlock); ok {
					inputBlocks = append(inputBlocks, inputBlock.BlockID)

					if blockElement, ok := inputBlock.Element.(*slack.SelectBlockElement); ok {
						for _, option := range blockElement.Options {
							zones = append(zones, option.Value)
						}
					}
				}
			}
			assert.Equal(t, tt.wantInputBlocks, inputBlocks)
			assert.Equal(t, tt.wantZones, zones)
		})
	}
}

func Test_setRoomShortcut_setRoom(t *testing.T) {
	type want struct {
		channel string
		action  string
		wantErr assert.ErrorAssertionFunc
	}
	tests := []struct {
		name      string
		data      slack.InteractionCallback
		setupTado func(t *mocks.TadoClient)
		want
	}{
		{
			name: "room to auto",
			data: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					State: &slack.ViewState{
						Values: map[string]map[string]slack.BlockAction{
							"zone":    {"zone": {SelectedOption: slack.OptionBlockObject{Value: "foo"}}},
							"mode":    {"mode": {SelectedOption: slack.OptionBlockObject{Value: "auto"}}},
							"channel": {"channel": {SelectedConversation: "C123456789"}},
						},
					},
				},
			},
			setupTado: func(t *mocks.TadoClient) {
				t.EXPECT().DeleteZoneOverlayWithResponse(context.Background(), int64(1), 10).Return(nil, nil)
			},
			want: want{
				channel: "C123456789",
				action:  "set *foo* to auto mode",
				wantErr: assert.NoError,
			},
		},
		{
			name: "room to permanent manual",
			data: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					State: &slack.ViewState{
						Values: map[string]map[string]slack.BlockAction{
							"zone":        {"zone": {SelectedOption: slack.OptionBlockObject{Value: "foo"}}},
							"mode":        {"mode": {SelectedOption: slack.OptionBlockObject{Value: "manual"}}},
							"temperature": {"temperature": {Value: "21.5"}},
							"channel":     {"channel": {SelectedConversation: "C123456789"}},
						},
					},
				},
			},
			setupTado: func(t *mocks.TadoClient) {
				t.EXPECT().
					SetZoneOverlayWithResponse(context.Background(), int64(1), 10, mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, _ int, overlay tado.ZoneOverlay, fn ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if temperature := *overlay.Setting.Temperature.Celsius; temperature != 21.5 {
							return nil, fmt.Errorf("temperature is wrong: want 21.5, got %.1f", temperature)
						}
						if mode := *overlay.Termination.Type; mode != tado.ZoneOverlayTerminationTypeMANUAL {
							return nil, fmt.Errorf("termination is wrong: want TIMER, got %q", mode)
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: want{
				channel: "C123456789",
				action:  "set *foo* to 21.5ºC",
				wantErr: assert.NoError,
			},
		},
		{
			name: "room to timer manual",
			data: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					State: &slack.ViewState{
						Values: map[string]map[string]slack.BlockAction{
							"zone":        {"zone": {SelectedOption: slack.OptionBlockObject{Value: "foo"}}},
							"mode":        {"mode": {SelectedOption: slack.OptionBlockObject{Value: "manual"}}},
							"temperature": {"temperature": {Value: "21.5"}},
							"expiration":  {"expiration": {SelectedTime: "22:00"}},
							"channel":     {"channel": {SelectedConversation: "C123456789"}},
						},
					},
				},
			},
			setupTado: func(t *mocks.TadoClient) {
				t.EXPECT().
					SetZoneOverlayWithResponse(context.Background(), int64(1), 10, mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, _ int, overlay tado.ZoneOverlay, fn ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if temperature := *overlay.Setting.Temperature.Celsius; temperature != 21.5 {
							return nil, fmt.Errorf("temperature is wrong: want 21.5, got %.1f", temperature)
						}
						if mode := *overlay.Termination.Type; mode != tado.ZoneOverlayTerminationTypeTIMER {
							return nil, fmt.Errorf("termination is wrong: want TIMER, got %q", mode)
						}
						if *overlay.Termination.DurationInSeconds == 0 {
							return nil, fmt.Errorf("expiration is not set")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: want{
				channel: "C123456789",
				action:  "set *foo* to 21.5ºC for ",
				wantErr: assert.NoError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tadoClient := mocks.NewTadoClient(t)
			h := setRoomShortcut{
				TadoClient: tadoClient,
				updateStore: updateStore{update: &poller.Update{
					HomeBase: tado.HomeBase{Id: oapi.VarP(int64(1))},
					Zones:    []poller.Zone{{Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("foo")}}},
				}},
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}
			if tt.setupTado != nil {
				tt.setupTado(tadoClient)
			}

			channel, action, err := h.setRoom(tt.data)
			assert.Equal(t, tt.want.channel, channel)
			assert.True(t, strings.HasPrefix(action, tt.want.action))
			tt.want.wantErr(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_setHomeShortcut_makeView(t *testing.T) {
	h := setHomeShortcut{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	req := h.makeView()

	inputBlocks := make([]string, 0, len(req.Blocks.BlockSet))
	for _, block := range req.Blocks.BlockSet {
		if inputBlock, ok := block.(*slack.InputBlock); ok {
			inputBlocks = append(inputBlocks, inputBlock.BlockID)
		}
	}
	assert.Equal(t, []string{"mode", "channel"}, inputBlocks)
}

func Test_setHomeShortcut_setHome(t *testing.T) {
	type want struct {
		channel string
		action  string
		wantErr assert.ErrorAssertionFunc
	}
	tests := []struct {
		name      string
		data      slack.InteractionCallback
		setupTado func(t *mocks.TadoClient)
		want
	}{
		{
			name: "home to auto",
			data: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					State: &slack.ViewState{
						Values: map[string]map[string]slack.BlockAction{
							"mode":    {"mode": {SelectedOption: slack.OptionBlockObject{Value: "auto"}}},
							"channel": {"channel": {SelectedConversation: "C123456789"}},
						},
					},
				},
			},
			setupTado: func(t *mocks.TadoClient) {
				t.EXPECT().DeletePresenceLockWithResponse(context.Background(), int64(1)).Return(nil, nil)
			},
			want: want{
				channel: "C123456789",
				action:  "set home to auto mode",
				wantErr: assert.NoError,
			},
		},
		{
			name: "home to home mode",
			data: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					State: &slack.ViewState{
						Values: map[string]map[string]slack.BlockAction{
							"mode":    {"mode": {SelectedOption: slack.OptionBlockObject{Value: "home"}}},
							"channel": {"channel": {SelectedConversation: "C123456789"}},
						},
					},
				},
			},
			setupTado: func(t *mocks.TadoClient) {
				t.EXPECT().
					SetPresenceLockWithResponse(context.Background(), int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, fn ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.HOME {
							return nil, fmt.Errorf("home presence is wrong")
						}
						return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: want{
				channel: "C123456789",
				action:  "set home to home mode",
				wantErr: assert.NoError,
			},
		},
		{
			name: "home to away mode",
			data: slack.InteractionCallback{
				Type: slack.InteractionTypeViewSubmission,
				View: slack.View{
					State: &slack.ViewState{
						Values: map[string]map[string]slack.BlockAction{
							"mode":    {"mode": {SelectedOption: slack.OptionBlockObject{Value: "away"}}},
							"channel": {"channel": {SelectedConversation: "C123456789"}},
						},
					},
				},
			},
			setupTado: func(t *mocks.TadoClient) {
				t.EXPECT().
					SetPresenceLockWithResponse(context.Background(), int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, fn ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.AWAY {
							return nil, fmt.Errorf("home presence is wrong")
						}
						return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					})
			},
			want: want{
				channel: "C123456789",
				action:  "set home to away mode",
				wantErr: assert.NoError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tadoClient := mocks.NewTadoClient(t)
			h := setHomeShortcut{
				TadoClient:  tadoClient,
				updateStore: updateStore{update: &poller.Update{HomeBase: tado.HomeBase{Id: oapi.VarP(int64(1))}}},
				logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
			}
			if tt.setupTado != nil {
				tt.setupTado(tadoClient)
			}

			channel, action, err := h.setHome(tt.data)
			assert.Equal(t, tt.want.channel, channel)
			assert.Equal(t, tt.want.action, action)
			tt.want.wantErr(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_timeStampToDuration(t *testing.T) {
	tests := []struct {
		name    string
		now     time.Time
		input   string
		want    time.Duration
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "later timestamp",
			now:     time.Date(2024, 11, 13, 23, 00, 0, 0, time.UTC),
			input:   "23:30",
			want:    30 * time.Minute,
			wantErr: assert.NoError,
		},
		{
			name:    "earlier timestamp means tomorrow",
			now:     time.Date(2024, 11, 13, 23, 00, 0, 0, time.UTC),
			input:   "22:00",
			want:    23 * time.Hour,
			wantErr: assert.NoError,
		},
		{
			name:    "invalid timestamp",
			now:     time.Date(2024, 11, 13, 23, 00, 0, 0, time.UTC),
			input:   "",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nowFunc = func() time.Time { return tt.now }
			got, err := timeStampToDuration(tt.input)
			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ shortcutHandler = &fakeHandler{}

type fakeHandler struct {
	calls []string
}

func (f *fakeHandler) HandleShortcut(_ slack.InteractionCallback, _ SlackSender) error {
	f.calls = append(f.calls, "shortcut")
	return nil
}

func (f *fakeHandler) HandleAction(_ slack.InteractionCallback, _ SlackSender) error {
	f.calls = append(f.calls, "action")
	return nil
}

func (f *fakeHandler) HandleSubmission(_ slack.InteractionCallback, _ SlackSender) error {
	f.calls = append(f.calls, "submit")
	return nil
}

func (f *fakeHandler) setUpdate(_ poller.Update) {}