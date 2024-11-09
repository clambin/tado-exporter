package zone

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
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

func TestLimitOverlayRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action string
		delay  time.Duration
		reason string
	}

	tests := []struct {
		name   string
		update poller.Update
		want
	}{
		{
			name: "zone in auto mode: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON), Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
				reason: "no manual temp setting detected",
			},
		},
		{
			name: "zone had timer overlay: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON), Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationTimer},
						},
					},
				},
			},
			want: want{assert.NoError, "no action", 0, "no manual temp setting detected"},
		},
		{
			name: "zone has manual 'off' mode, home is HOME: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			want: want{assert.NoError, "no action", 0, "no manual temp setting detected"},
		},
		{
			name: "zone has manual 'off' overlay, but home is AWAY: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			want: want{assert.NoError, "no action", 0, "home in AWAY mode"},
		},
		{
			name: "zone has manual 'on' overlay and home is HOME: remove overlay",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON), Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			want: want{assert.NoError, "moving to auto mode", time.Hour, "manual temp setting detected"},
		},
		{
			name: "invalid zone",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
			},
			want: want{
				err: assert.Error,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := configuration.LimitOverlayConfiguration{Delay: time.Hour}
			r, err := LoadLimitOverlay(10, "room", cfg, tt.update, slog.Default())
			require.NoError(t, err)

			e, err := r.Evaluate(tt.update)
			tt.want.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want.action, e.String())
			assert.Equal(t, tt.want.delay, e.Delay)
			assert.Equal(t, tt.want.reason, e.Reason)

			switch e.State.Mode() {
			case action.ZoneInAutoMode:
				assert.NoError(t, e.State.Do(context.Background(), fakeClient{expect: "delete"}))
			case action.NoAction:
			default:
				t.Errorf("unknown state: %s", e.State.Mode().String())
			}

		})
	}
}
