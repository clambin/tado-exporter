package bot

import (
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	mocks2 "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_commandRunner_listRooms(t *testing.T) {
	tests := []struct {
		name    string
		update  *poller.Update
		wantErr assert.ErrorAssertionFunc
		want    textResponse
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no rooms",
			update:  &poller.Update{},
			wantErr: assert.NoError,
			want:    textResponse{header: "Rooms:", body: []string{"no rooms have been found"}},
		},
		{
			name: "rooms found",
			update: &poller.Update{
				Zones: []poller.Zone{
					{
						Zone: tado.Zone{Id: oapi.VarP(40), Name: oapi.VarP("room D")},
						ZoneState: tado.ZoneState{
							Setting:          &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
							SensorDataPoints: &tado.SensorDataPoints{InsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(20))}},
						},
					},
					{
						Zone: tado.Zone{Id: oapi.VarP(30), Name: oapi.VarP("room C")},
						ZoneState: tado.ZoneState{
							Setting:          &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON), Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(21))}},
							SensorDataPoints: &tado.SensorDataPoints{InsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(20))}},
							Overlay:          &tado.ZoneOverlay{Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeTIMER), RemainingTimeInSeconds: oapi.VarP(300)}},
						},
					},
					{
						Zone: tado.Zone{Id: oapi.VarP(20), Name: oapi.VarP("room B")},
						ZoneState: tado.ZoneState{
							Setting:          &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON), Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(17.5))}},
							SensorDataPoints: &tado.SensorDataPoints{InsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(21))}},
							Overlay:          &tado.ZoneOverlay{Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL)}},
						},
					},
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room A")},
						ZoneState: tado.ZoneState{
							Setting:          &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON), Temperature: &tado.Temperature{Celsius: oapi.VarP(float32(21))}},
							SensorDataPoints: &tado.SensorDataPoints{InsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(20))}},
						},
					},
				},
			},
			wantErr: assert.NoError,
			want: textResponse{header: "Rooms:", body: []string{
				"*room A*: 20.0ºC (target: 21.0)",
				"*room B*: 21.0ºC (target: 17.5, MANUAL)",
				"*room C*: 20.0ºC (target: 21.0, MANUAL for 5m0s)",
				"*room D*: 20.0ºC (off)",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := commandRunner{}
			if tt.update != nil {
				r.setUpdate(*tt.update)
			}

			got, err := r.listRooms()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commandRunner_listUsers(t *testing.T) {
	tests := []struct {
		name    string
		update  *poller.Update
		wantErr assert.ErrorAssertionFunc
		want    textResponse
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no users",
			update:  &poller.Update{},
			wantErr: assert.NoError,
			want:    textResponse{header: "Users:", body: []string{"no users have been found"}},
		},
		{
			name: "users found",
			update: &poller.Update{
				MobileDevices: []tado.MobileDevice{
					{
						Name:     oapi.VarP("user D"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
						Location: &tado.MobileDeviceLocation{AtHome: oapi.VarP(false)},
					},
					{
						Name:     oapi.VarP("user C"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
						Location: &tado.MobileDeviceLocation{AtHome: oapi.VarP(true)},
					},
					{
						Name:     oapi.VarP("user B"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(false)},
					},
					{
						Name:     oapi.VarP("user A"),
						Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
					},
				},
			},
			wantErr: assert.NoError,
			want: textResponse{header: "Users:", body: []string{
				"*user C*: home",
				"*user D*: away",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := commandRunner{}
			if tt.update != nil {
				r.setUpdate(*tt.update)
			}

			got, err := r.listUsers()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commandRunner_listRules(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.Controller)
		wantErr assert.ErrorAssertionFunc
		want    textResponse
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name: "no rules",
			setup: func(c *mocks.Controller) {
				c.EXPECT().ReportTasks().Return(nil).Once()
			},
			wantErr: assert.NoError,
			want:    textResponse{header: "Rules:", body: []string{"no rules have been triggered"}},
		},
		{
			name: "rules found",
			setup: func(c *mocks.Controller) {
				c.EXPECT().ReportTasks().Return([]string{
					"room B: bar",
					"room A: foo",
				}).Once()
			},
			wantErr: assert.NoError,
			want: textResponse{header: "Rules:", body: []string{
				"room A: foo",
				"room B: bar",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := commandRunner{}
			if tt.setup != nil {
				c := mocks.NewController(t)
				tt.setup(c)
				r.Controller = c
			}

			got, err := r.listRules()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commandRunner_refresh(t *testing.T) {
	p := mocks2.NewPoller(t)
	p.EXPECT().Refresh().Once()
	r := commandRunner{Poller: p}
	_, err := r.refresh()
	assert.NoError(t, err)
}

func Test_commandRunner_help(t *testing.T) {
	var r commandRunner
	resp, err := r.help()
	assert.NoError(t, err)
	assert.Equal(t, textResponse{header: "Supported commands:", body: []string{"users, rooms, rules, help"}}, resp)
}
