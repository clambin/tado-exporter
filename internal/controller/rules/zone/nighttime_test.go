package zone

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

func TestNightTimeRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action string
		delay  time.Duration
		reason string
	}

	tests := []struct {
		name   string
		update poller.Update
		now    time.Time
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
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
			},
			want: want{assert.NoError, "no action", 0, "no manual temp setting detected"},
		},
		{
			name: "zone in non-manual overlay: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationTimer},
						},
					},
				},
			},
			now:  time.Date(2023, time.December, 31, 22, 30, 0, 0, time.Local),
			want: want{assert.NoError, "no action", 0, "no manual temp setting detected"},
		},
		{
			name: "zone in manual mode, but not heating: no action",
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
			now:  time.Date(2023, time.December, 31, 23, 45, 0, 0, time.Local),
			want: want{assert.NoError, "no action", 0, "no manual temp setting detected"},
		},
		{
			name: "zone in manual mode, but home is AWAY: no action",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.AWAY)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](25.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			now:  time.Date(2023, time.December, 31, 23, 45, 0, 0, time.Local),
			want: want{assert.NoError, "no action", 0, "home in AWAY mode"},
		},
		{
			name: "zone in manual mode: switch off manual mode today",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](25.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			now:  time.Date(2023, time.December, 31, 23, 15, 0, 0, time.Local),
			want: want{assert.NoError, "moving to auto mode", 15 * time.Minute, "manual temp setting detected"},
		},
		{
			name: "zone in manual mode: switch off manual mode tomorrow",
			update: poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{
							Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](25.0)}},
							Overlay: &tado.ZoneOverlay{Termination: &oapi.TerminationManual},
						},
					},
				},
			},
			now:  time.Date(2023, time.December, 31, 23, 45, 0, 0, time.Local),
			want: want{assert.NoError, "moving to auto mode", 23*time.Hour + 45*time.Minute, "manual temp setting detected"},
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
			cfg := configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Seconds: 0, Active: true}}
			r, err := LoadNightTime(10, "room", cfg, tt.update, slog.Default())
			require.NoError(t, err)

			r.getCurrentTime = func() time.Time { return tt.now }
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
