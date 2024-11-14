package bot

import (
	"context"
	"errors"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	mocks2 "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

func TestBot_listRooms(t *testing.T) {
	tests := []struct {
		name    string
		update  *poller.Update
		expect  func(sender *mocks.SlackSender)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no rooms",
			update:  &poller.Update{},
			wantErr: assert.Error,
		},
		{
			name: "room found",
			update: &poller.Update{
				Zones: []poller.Zone{{
					Zone: tado.Zone{Name: oapi.VarP("room")},
					ZoneState: tado.ZoneState{
						Setting: &tado.ZoneSetting{
							Power:       oapi.VarP(tado.PowerON),
							Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(17.5))},
						},
						SensorDataPoints: &tado.SensorDataPoints{
							InsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(21))},
						},
					}},
				},
			},
			expect: func(sender *mocks.SlackSender) {
				sender.EXPECT().
					PostEphemeral("1", "2", mock.Anything).
					Return("", nil)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Bot{}
			if tt.update != nil {
				b.setUpdate(*tt.update)
			}

			s := mocks.NewSlackSender(t)
			if tt.expect != nil {
				tt.expect(s)
			}

			err := b.listRooms(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s)
			tt.wantErr(t, err)
		})
	}
}

func TestBot_listUsers(t *testing.T) {
	tests := []struct {
		name    string
		update  *poller.Update
		expect  func(sender *mocks.SlackSender)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no rooms",
			update:  &poller.Update{},
			wantErr: assert.Error,
		},
		{
			name: "users found",
			update: &poller.Update{
				MobileDevices: []tado.MobileDevice{
					{
						Name:     oapi.VarP("foo"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
						Location: &tado.MobileDeviceLocation{AtHome: oapi.VarP(false)},
					},
					{
						Name:     oapi.VarP("bar"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
						Location: &tado.MobileDeviceLocation{AtHome: oapi.VarP(true)},
					},
					{
						Name:     oapi.VarP("bar"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(false)},
					},
					{
						Name:     oapi.VarP("snafu"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
					},
				},
			},
			expect: func(sender *mocks.SlackSender) {
				sender.EXPECT().
					PostEphemeral("1", "2", mock.Anything).
					Return("", nil)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Bot{}
			if tt.update != nil {
				b.setUpdate(*tt.update)
			}

			s := mocks.NewSlackSender(t)
			if tt.expect != nil {
				tt.expect(s)
			}

			err := b.listUsers(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s)
			tt.wantErr(t, err)
		})
	}
}

func TestBot_listRules(t *testing.T) {
	tests := []struct {
		name       string
		controller func(controller *mocks.Controller)
		expect     func(sender *mocks.SlackSender)
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no rooms",
			wantErr: assert.Error,
		},
		{
			name: "users found",
			controller: func(controller *mocks.Controller) {
				controller.EXPECT().ReportTasks().Return([]string{"foo", "bar"})
			},
			expect: func(sender *mocks.SlackSender) {
				sender.EXPECT().
					PostEphemeral("1", "2", mock.Anything).
					Return("", nil)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Bot{}
			if tt.controller != nil {
				c := mocks.NewController(t)
				tt.controller(c)
				b.controller = c
			}

			s := mocks.NewSlackSender(t)
			if tt.expect != nil {
				tt.expect(s)
			}

			err := b.listRules(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s)
			tt.wantErr(t, err)
		})
	}
}

func TestBot_refresh(t *testing.T) {
	p := mocks2.NewPoller(t)
	p.EXPECT().Refresh()

	s := mocks.NewSlackSender(t)
	s.EXPECT().PostEphemeral("1", "2", mock.Anything).Return("", nil)

	b := Bot{poller: p}
	assert.NoError(t, b.refresh(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s))
}

func TestBot_setRoom(t *testing.T) {
	tests := []struct {
		name        string
		cmdline     string
		update      *poller.Update
		tadoExpect  func(sender *mocks.TadoClient)
		slackExpect func(sender *mocks.SlackSender)
		wantErr     assert.ErrorAssertionFunc
		errMessage  string
	}{
		{
			name:       "invalid command",
			wantErr:    assert.Error,
			errMessage: "missing parameters\nUsage: set room <room> [auto|<temperature> [<duration>]",
		},
		{
			name:       "no update",
			cmdline:    "foo auto",
			wantErr:    assert.Error,
			errMessage: ErrNoUpdates.Error(),
		},
		{
			name:       "invalid zone name",
			cmdline:    "foo auto",
			update:     &poller.Update{},
			wantErr:    assert.Error,
			errMessage: `invalid room name: "foo"`,
		},
		{
			name:    "move zone to auto mode",
			cmdline: "foo auto",
			update: &poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP(tado.HomeId(1))},
				Zones: []poller.Zone{{
					Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("foo")},
				}},
			},
			tadoExpect: func(sender *mocks.TadoClient) {
				sender.EXPECT().DeleteZoneOverlayWithResponse(context.Background(), int64(1), 10).Return(nil, nil)
			},
			slackExpect: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostMessage("1", mock.Anything).Return("", "", nil)
			},
			wantErr: assert.NoError,
		},
		{
			name:    "move zone to manual mode",
			cmdline: "foo 21 20m",
			update: &poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP(tado.HomeId(1))},
				Zones: []poller.Zone{{
					Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("foo")},
				}},
			},
			tadoExpect: func(c *mocks.TadoClient) {
				c.EXPECT().
					SetZoneOverlayWithResponse(context.Background(), int64(1), 10, mock.Anything).
					Return(&tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil)
			},
			slackExpect: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostMessage("1", mock.Anything).Return("", "", nil)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mocks.NewTadoClient(t)
			if tt.tadoExpect != nil {
				tt.tadoExpect(c)
			}
			s := mocks.NewSlackSender(t)
			if tt.slackExpect != nil {
				tt.slackExpect(s)
			}
			p := mocks2.NewPoller(t)
			p.EXPECT().Refresh().Maybe()
			b := Bot{TadoClient: c, poller: p}
			if tt.update != nil {
				b.setUpdate(*tt.update)
			}

			err := b.setRoom(slack.SlashCommand{ChannelID: "1", UserID: "2", Text: tt.cmdline}, s)
			tt.wantErr(t, err)
			if err != nil {
				assert.Equal(t, tt.errMessage, err.Error())
			}
		})
	}
}

func TestBot_setHome(t *testing.T) {
	tests := []struct {
		name        string
		cmdline     string
		update      *poller.Update
		tadoExpect  func(sender *mocks.TadoClient)
		slackExpect func(sender *mocks.SlackSender)
		wantErr     assert.ErrorAssertionFunc
		errMessage  string
	}{
		{
			name:       "invalid command",
			wantErr:    assert.Error,
			errMessage: "missing parameter\nUsage: set home [home|away|auto]",
		},
		{
			name:       "no update",
			cmdline:    "auto",
			wantErr:    assert.Error,
			errMessage: ErrNoUpdates.Error(),
		},
		{
			name:    "move home to auto mode",
			cmdline: "auto",
			update: &poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP(tado.HomeId(1))},
			},
			tadoExpect: func(sender *mocks.TadoClient) {
				sender.EXPECT().DeletePresenceLockWithResponse(context.Background(), int64(1)).Return(nil, nil)
			},
			slackExpect: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostMessage("1", mock.Anything).Return("", "", nil)
			},
			wantErr: assert.NoError,
		},
		{
			name:    "move home to away mode",
			cmdline: "away",
			update: &poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP(tado.HomeId(1))},
			},
			tadoExpect: func(sender *mocks.TadoClient) {
				sender.EXPECT().
					SetPresenceLockWithResponse(context.Background(), int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.AWAY {
							return nil, errors.New("not AWAY")
						}
						return nil, nil
					})
			},
			slackExpect: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostMessage("1", mock.Anything).Return("", "", nil)
			},
			wantErr: assert.NoError,
		},
		{
			name:    "move home to home mode",
			cmdline: "home",
			update: &poller.Update{
				HomeBase: tado.HomeBase{Id: oapi.VarP(tado.HomeId(1))},
			},
			tadoExpect: func(sender *mocks.TadoClient) {
				sender.EXPECT().
					SetPresenceLockWithResponse(context.Background(), int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.HOME {
							return nil, errors.New("not HOME")
						}
						return nil, nil
					})
			},
			slackExpect: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostMessage("1", mock.Anything).Return("", "", nil)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mocks.NewTadoClient(t)
			if tt.tadoExpect != nil {
				tt.tadoExpect(c)
			}
			s := mocks.NewSlackSender(t)
			if tt.slackExpect != nil {
				tt.slackExpect(s)
			}
			p := mocks2.NewPoller(t)
			p.EXPECT().Refresh().Maybe()
			b := Bot{TadoClient: c, poller: p}
			if tt.update != nil {
				b.setUpdate(*tt.update)
			}

			err := b.setHome(slack.SlashCommand{ChannelID: "1", UserID: "2", Text: tt.cmdline}, s)
			tt.wantErr(t, err)
			if err != nil {
				assert.Equal(t, tt.errMessage, err.Error())
			}
		})
	}
}
