package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestAutoAwayRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action assert.BoolAssertionFunc
		delay  time.Duration
		reason string
		state  action.State
		home   bool
	}

	tests := []struct {
		name   string
		update poller.Update
		want
	}{
		{
			name: "home in home mode, all users are home: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
				},
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "A, B are home",
			},
		},
		{
			name: "home in home mode, one user is home: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "A is home",
			},
		},
		{
			name: "home in home mode, all users go away: move home to away mode",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				delay:  time.Hour,
				reason: "A, B are away",
				state:  State{mode: action.HomeInAwayMode},
				home:   false,
			},
		},
		{
			name: "home in away mode, all users are away: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "A, B are away",
			},
		},
		{
			name: "home in away mode, one user comes home: move to home mode",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationAway},
				},
			},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				reason: "A is home",
				state:  State{mode: action.HomeInHomeMode},
				home:   true,
			},
		},
		{
			name: "home in away mode, all users are home: move to home mode",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
					{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
				},
			},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				reason: "A, B are home",
				state:  State{mode: action.HomeInHomeMode},
				home:   true,
			},
		},
	}

	cfg := configuration.AutoAwayConfiguration{
		Users: []string{"A", "B"},
		Delay: time.Hour,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := LoadAutoAwayRule(cfg, tt.update, slog.Default())
			tt.err(t, err)
			if err != nil {
				return
			}
			e, err := r.Evaluate(tt.update)
			tt.want.err(t, err)
			if err != nil {
				return
			}
			tt.action(t, e.IsAction())
			assert.Equal(t, tt.want.delay, e.Delay)
			assert.Equal(t, tt.want.reason, e.Reason)
		})
	}
}
