package bot

import (
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	mocks2 "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"log/slog"
	"testing"
)

func Test_commandRunner_dispatch(t *testing.T) {
	tests := []struct {
		name              string
		update            *poller.Update
		expectPoller      func(*mocks2.Poller)
		expectSlackSender func(*mocks.SlackSender)
		expectController  func(*mocks.Controller)
		command           string
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name:    "invalid command",
			wantErr: assert.Error,
		},
		{
			name:    "rooms: no update",
			command: "rooms",
			wantErr: assert.Error,
		},
		{
			name:    "rooms: no rooms",
			update:  &poller.Update{},
			command: "rooms",
			wantErr: assert.Error,
		},
		{
			name: "rooms: room found",
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
			expectSlackSender: func(sender *mocks.SlackSender) {
				sender.EXPECT().
					PostEphemeral("1", "2", mock.Anything).
					Return("", nil)
			},
			command: "rooms",
			wantErr: assert.NoError,
		},
		{
			name:    "users: no update",
			command: "users",
			wantErr: assert.Error,
		},
		{
			name:    "users: no users found",
			update:  &poller.Update{},
			command: "users",
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
			expectSlackSender: func(sender *mocks.SlackSender) {
				sender.EXPECT().
					PostEphemeral("1", "2", mock.Anything).
					Return("", nil)
			},
			command: "users",
			wantErr: assert.NoError,
		},
		{
			name:    "rules: no controller",
			command: "rules",
			wantErr: assert.Error,
		},
		{
			name: "rules: rules found",
			expectController: func(controller *mocks.Controller) {
				controller.EXPECT().ReportTasks().Return([]string{"foo", "bar"})
			},
			expectSlackSender: func(sender *mocks.SlackSender) {
				sender.EXPECT().
					PostEphemeral("1", "2", mock.Anything).
					Return("", nil)
			},
			command: "rules",
			wantErr: assert.NoError,
		},
		{
			name: "refresh",
			expectPoller: func(p *mocks2.Poller) {
				p.EXPECT().Refresh()
			},
			expectSlackSender: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostEphemeral("1", "2", mock.Anything).Return("", nil)
			},
			command: "refresh",
			wantErr: assert.NoError,
		},
		{
			name: "help",
			expectSlackSender: func(sender *mocks.SlackSender) {
				sender.EXPECT().PostEphemeral("1", "2", mock.Anything).Return("", nil)
			},
			command: "help",
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p *mocks2.Poller
			if tt.expectPoller != nil {
				p = mocks2.NewPoller(t)
				tt.expectPoller(p)
			}
			var s *mocks.SlackSender
			if tt.expectSlackSender != nil {
				s = mocks.NewSlackSender(t)
				tt.expectSlackSender(s)
			}
			var c *mocks.Controller
			if tt.expectController != nil {
				c = mocks.NewController(t)
				tt.expectController(c)
			}

			r := commandRunner{
				poller:     p,
				controller: c,
				logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			if tt.update != nil {
				r.setUpdate(*tt.update)
			}

			err := r.dispatch(slack.SlashCommand{ChannelID: "1", UserID: "2", Text: tt.command}, s)
			tt.wantErr(t, err)

		})
	}
}

func Test_zoneState(t *testing.T) {
	tests := []struct {
		name string
		zone poller.Zone
		want string
	}{
		{
			name: "auto",
			zone: poller.Zone{
				Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
				ZoneState: tado.ZoneState{
					Setting: &tado.ZoneSetting{
						Power:       oapi.VarP(tado.PowerON),
						Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(17.5))},
					},
				},
			},
			want: "target: 17.5",
		},
		{
			name: "off",
			zone: poller.Zone{
				Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
				ZoneState: tado.ZoneState{
					Setting: &tado.ZoneSetting{
						Power: oapi.VarP(tado.PowerOFF),
					},
				},
			},
			want: "off",
		},
		{
			name: "manual - permanent",
			zone: poller.Zone{
				Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
				ZoneState: tado.ZoneState{
					Setting: &tado.ZoneSetting{
						Power:       oapi.VarP(tado.PowerON),
						Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(17.5))},
					},
					Overlay: &tado.ZoneOverlay{
						Termination: &tado.ZoneOverlayTermination{
							Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL),
						},
					},
				},
			},
			want: "target: 17.5, MANUAL",
		},
		{
			name: "manual - timer",
			zone: poller.Zone{
				Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
				ZoneState: tado.ZoneState{
					Setting: &tado.ZoneSetting{
						Power:       oapi.VarP(tado.PowerON),
						Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(17.5))},
					},
					Overlay: &tado.ZoneOverlay{
						Termination: &tado.ZoneOverlayTermination{
							Type:                   oapi.VarP(tado.ZoneOverlayTerminationTypeTIMER),
							RemainingTimeInSeconds: oapi.VarP(300),
						},
					},
				},
			},
			want: "target: 17.5, MANUAL for 5m0s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, zoneState(tt.zone))
		})
	}
}
