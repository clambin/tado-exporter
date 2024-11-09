package zone

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestHomeAwayRule_Evaluate(t *testing.T) {
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
			name: "zone in overlay, but home in HOME mode: no action",
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
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
			},
		},
		{
			name: "zone in overlay and home in AWAY mode: delete overlay",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](5.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "moving to auto mode",
				reason: "home in AWAY mode, manual temp setting detected",
			},
		},
		{
			name: "zone in auto mode: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](5.0)}}},
					},
				},
			},
			want: want{
				err:    assert.NoError,
				action: "no action",
				reason: "home in AWAY mode, no manual temp setting detected",
			},
		},
		{
			name: "invalid zone",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
			},
			want: want{
				err: assert.Error,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := LoadHomeAwayRule(10, "room", tt.update, slog.Default())
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
