package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestZoneRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update poller.Update
		want   action
		err    assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: true, heating: true},
					reason: "test",
					delay:  5 * time.Minute,
				},
				homeId:   1,
				zoneName: "foo",
				zoneId:   10,
			},
			err: assert.NoError,
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(home, zone, devices)
	return zone, nil, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.Error,
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(home, zone, devices)
	return zone, 0, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.Error,
		},
		{
			name: "missing zone in update",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
			),
			err: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newZoneRule("foo", strings.NewReader(tt.script), nil, nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

func TestZoneRule_Evaluate_LimitOverlay(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		err    assert.ErrorAssertionFunc
		want   action
	}{
		{
			name: "zone auto -> no action",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: false, heating: true}, reason: "no manual setting detected", delay: 0},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
		{
			name: "zone manual -> delete overlay",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: false, heating: true}, reason: "manual setting detected", delay: time.Hour},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
		{
			name: "zone off -> no action",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: true, heating: false}, reason: "heating is off", delay: 0},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := loadZoneRule(
				"zone",
				RuleConfiguration{Script: ScriptConfig{Packaged: "limitoverlay.lua"}},
			)
			require.NoError(t, err)

			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

func TestZoneRule_Evaluate_AutoAway(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		err    assert.ErrorAssertionFunc
		want   action
	}{
		{
			name: "zone auto, user home -> heating in auto",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: false, heating: true},
					reason: "one or more users are home: user A",
					delay:  0,
				},
				homeId:   1,
				zoneId:   10,
				zoneName: "zone",
			},
		},
		{
			name: "zone auto, user away -> heating off",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: true, heating: false}, reason: "all users are away: user A", delay: 15 * time.Minute},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
		{
			name: "zone off, user away -> heating off",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: true, heating: false}, reason: "all users are away: user A", delay: 0},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
		{
			name: "zone off, user home -> zone to auto",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: false, heating: true}, reason: "one or more users are home: user A", delay: 0},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := loadZoneRule(
				"zone",
				RuleConfiguration{Script: ScriptConfig{Packaged: "autoaway.lua"}, Users: []string{"user A"}},
			)
			require.NoError(t, err)

			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

func TestZoneAction_Evaluate_NightTime(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		now    time.Time
		err    assert.ErrorAssertionFunc
		want   action
	}{
		{
			name: "zone auto -> no action",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: false, heating: true},
					reason: "no manual setting detected",
					delay:  0,
				},
				homeId:   1,
				zoneId:   10,
				zoneName: "zone",
			},
		},
		{
			name: "zone in manual mode, before range -> schedule moving to auto mode",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			now: time.Date(2024, time.December, 3, 18, 0, 0, 0, time.Local),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: false, heating: true},
					reason: "manual setting detected",
					delay:  5 * time.Hour,
				},
				homeId:   1,
				zoneId:   10,
				zoneName: "zone",
			},
		},
		{
			name: "zone in manual mode, during range, before midnight -> immediately move to auto mode",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			now: time.Date(2024, time.December, 3, 23, 30, 0, 0, time.Local),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: false, heating: true},
					reason: "manual setting detected",
					delay:  0,
				},
				homeId:   1,
				zoneId:   10,
				zoneName: "zone",
			},
		},
		/*
			{
				// TODO: fix bug in IsInRange where it doesn't detect that a timestamp after midnight is still in range
				name: "zone in manual mode, during range, after midnight -> immediately move to auto mode",
				update: testutils.Update(
					testutils.WithHome(1, "my home", tado.HOME),
					testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				),
				now: time.Date(2024, time.December, 4, 1, 0, 0, 0, time.Local),
				err: assert.NoError,
				want: &zoneAction{
					coreAction: coreAction{
						state:  zoneState{overlay: false, heating: true},
						reason: "manual setting detected",
						delay:  0,
					},
					homeId:   1,
					zoneId:   10,
					zoneName: "zone",
				},
			},
		*/
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := loadZoneRule(
				"zone",
				RuleConfiguration{
					Script: ScriptConfig{Packaged: "nighttime.lua"},
					Args: Args{
						"StartHour": 23, "StartMin": 0,
						"EndHour": 7, "EndMin": 0,
					},
				},
			)
			require.NoError(t, err)

			luart.Register(r.(*zoneRule).luaScript.State, func() time.Time { return tt.now })

			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestZoneAction_Do(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		tadoClient func(t *testing.T) *mocks.TadoClient
		action     coreAction
		err        assert.ErrorAssertionFunc
	}{
		{
			name: "move to auto mode",
			tadoClient: func(t *testing.T) *mocks.TadoClient {
				c := mocks.NewTadoClient(t)
				c.EXPECT().
					DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
					Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
					Once()
				return c
			},
			action: coreAction{state: zoneState{false, true}},
			err:    assert.NoError,
		},
		{
			name: "switch heating off",
			tadoClient: func(t *testing.T) *mocks.TadoClient {
				c := mocks.NewTadoClient(t)
				c.EXPECT().
					SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					RunAndReturn(func(_ context.Context, _ int64, _ int, overlay tado.ZoneOverlay, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if *overlay.Setting.Type != tado.HEATING || *overlay.Setting.Power != tado.PowerOFF {
							return nil, fmt.Errorf("wrong settings")
						}
						if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
							return nil, fmt.Errorf("wrong termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
				return c
			},
			action: coreAction{state: zoneState{true, false}},
			err:    assert.NoError,
		},
		{
			name: "switch heating on",
			tadoClient: func(t *testing.T) *mocks.TadoClient {
				c := mocks.NewTadoClient(t)
				c.EXPECT().
					SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					RunAndReturn(func(_ context.Context, _ int64, _ int, overlay tado.ZoneOverlay, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if *overlay.Setting.Type != tado.HEATING || *overlay.Setting.Power != tado.PowerON {
							return nil, fmt.Errorf("wrong settings")
						}
						if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
							return nil, fmt.Errorf("wrong termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
				return c
			},
			action: coreAction{state: zoneState{true, true}},
			err:    assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := zoneAction{tt.action, "foo", 1, 10}
			tt.err(t, a.Do(context.Background(), tt.tadoClient(t), discardLogger))
		})
	}
}

func TestZoneAction_LogValue(t *testing.T) {
	z := zoneAction{
		coreAction: coreAction{zoneState{true, true}, "foo", 5 * time.Minute},
		zoneName:   "zone",
		homeId:     1,
		zoneId:     10,
	}
	assert.Equal(t, `[zone=zone action=[state=[overlay=true heating=true] delay=5m0s reason=foo]]`, z.LogValue().String())
}

func BenchmarkZoneAction_Evaluate(b *testing.B) {
	r, err := loadZoneRule("foo", RuleConfiguration{Script: ScriptConfig{Packaged: "nighttime.lua"}})
	require.NoError(b, err)
	u := testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(10, "foo", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
	)
	b.ResetTimer()
	for range b.N {
		if _, err := r.Evaluate(u); err != nil {
			b.Fatal(err)
		}
	}
}
