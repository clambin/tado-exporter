package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestAutoAwayRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action string
		delay  time.Duration
		reason string
	}

	var testCases = []struct {
		name   string
		update poller.Update
		want
	}{
		{
			name: "zone is heated, all users are home: no action required",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
				reason: "A, B are home",
			},
		},
		{
			name: "zone is heated, one user is home: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
				reason: "A is home",
			},
		},
		{
			name: "zone is heated, all users are away: switch off heating",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "switching off heating",
				delay:  time.Hour,
				reason: "A, B are away",
			},
		},
		{
			name: "zone is not heated, all users are away: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](5.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
				reason: "A, B are away",
			},
		},
		{
			name: "zone is not heated, user comes home: switch to auto mode",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](5.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "moving to auto mode",
				reason: "A is home",
			},
		},
		{
			name: "zone is heated, all users are away, but home in away mode: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
				reason: "home in AWAY mode",
			},
		},
	}

	cfg := configuration.AutoAwayConfiguration{
		Users: []string{"A", "B"},
		Delay: time.Hour,
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			r, err := LoadAutoAwayRule(10, "room", cfg, tt.update, slog.Default())
			require.NoError(t, err)

			e, err := r.Evaluate(tt.update)
			tt.want.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want.action, e.String())
			assert.Equal(t, tt.want.delay, e.Delay)
			assert.Equal(t, tt.want.reason, e.Reason)
		})
	}
}

func TestLoadAutoAwayRule(t *testing.T) {
	tests := []struct {
		name string
		cfg  configuration.AutoAwayConfiguration
		want assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			cfg:  configuration.AutoAwayConfiguration{Users: []string{"A"}, Delay: time.Hour},
			want: assert.NoError,
		},
		{
			name: "invalid user",
			cfg:  configuration.AutoAwayConfiguration{Users: []string{"A", "B"}, Delay: time.Hour},
			want: assert.Error,
		},
		{
			name: "no users",
			cfg:  configuration.AutoAwayConfiguration{Users: nil, Delay: time.Hour},
			want: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := poller.Update{
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}},
				},
			}
			_, err := LoadAutoAwayRule(10, "room", tt.cfg, update, slog.Default())
			tt.want(t, err)
		})
	}
}
