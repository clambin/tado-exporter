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
	"testing"
)

func Test_commandRunner_listRooms(t *testing.T) {
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
			var r commandRunner
			if tt.update != nil {
				r.setUpdate(*tt.update)
			}

			s := mocks.NewSlackSender(t)
			if tt.expect != nil {
				tt.expect(s)
			}

			err := r.listRooms(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s)
			tt.wantErr(t, err)
		})
	}
}

func Test_commandRunner_listUsers(t *testing.T) {
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
			var r commandRunner
			if tt.update != nil {
				r.setUpdate(*tt.update)
			}

			s := mocks.NewSlackSender(t)
			if tt.expect != nil {
				tt.expect(s)
			}

			err := r.listUsers(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s)
			tt.wantErr(t, err)
		})
	}
}

func Test_commandRunner_listRules(t *testing.T) {
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
			var r commandRunner
			if tt.controller != nil {
				c := mocks.NewController(t)
				tt.controller(c)
				r.controller = c
			}

			s := mocks.NewSlackSender(t)
			if tt.expect != nil {
				tt.expect(s)
			}

			err := r.listRules(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s)
			tt.wantErr(t, err)
		})
	}
}

func Test_commandRunner_refresh(t *testing.T) {
	p := mocks2.NewPoller(t)
	p.EXPECT().Refresh()

	s := mocks.NewSlackSender(t)
	s.EXPECT().PostEphemeral("1", "2", mock.Anything).Return("", nil)

	r := commandRunner{poller: p}
	assert.NoError(t, r.refresh(slack.SlashCommand{ChannelID: "1", UserID: "2"}, s))
}
