package controller

import (
	"context"
	"errors"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

type zoneWant struct {
	ZoneState
	delay  time.Duration
	reason string
	err    assert.ErrorAssertionFunc
}

func TestZoneRules(t *testing.T) {
	type want struct {
		ZoneState
		delay  time.Duration
		reason string
		err    assert.ErrorAssertionFunc
	}
	tests := []struct {
		name string
		ZoneRules
		Update
		want
	}{
		{
			name: "no rules",
			want: want{"", 0, "no rules found", assert.Error},
		},
		{
			name: "single rule",
			ZoneRules: ZoneRules{
				zoneName: "foo",
				rules: []Evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				},
			},
			Update: Update{HomeState: HomeStateAuto, ZoneStates: map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}}, Devices: nil},
			want:   want{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with same desired zone state: pick the first one",
			ZoneRules: ZoneRules{
				zoneName: "foo",
				rules: []Evaluator{
					fakeZoneEvaluator{ZoneStateAuto, time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateAuto, time.Hour, "manual setting detected", nil},
				},
			},
			Update: Update{HomeState: HomeStateAuto, ZoneStates: map[string]ZoneInfo{"foo": {ZoneState: ZoneStateManual}}, Devices: nil},
			want:   want{ZoneStateAuto, time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states: pick the first one",
			ZoneRules: ZoneRules{
				zoneName: "foo",
				rules: []Evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
				},
			},
			Update: Update{HomeState: HomeStateAuto, ZoneStates: map[string]ZoneInfo{"foo": {ZoneState: ZoneStateManual}}, Devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states, including `no change`: pick the first non-matching",
			ZoneRules: ZoneRules{
				zoneName: "foo",
				rules: []Evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
					fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				},
			},
			Update: Update{HomeState: HomeStateAuto, ZoneStates: map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}}, Devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := tt.ZoneRules.Evaluate(tt.Update)
			tt.want.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want.ZoneState, ZoneState(a.GetState()))
			assert.Equal(t, tt.want.delay, a.GetDelay())
			assert.Equal(t, tt.want.reason, a.GetReason())
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestZoneRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		Update
		zoneWant
	}{
		{
			name: "success",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateOff}}, Devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateOff, 5 * time.Minute, "test", assert.NoError},
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(home, zone, devices)
	return zone, nil, "test"
end
`,
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateOff}}, Devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{"", 0, "", assert.Error},
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(home, zone, devices)
	return zone, 0, "test"
end
`,
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateOff}}, Devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{"", 0, "", assert.Error},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewZoneRule("foo", strings.NewReader(tt.script))
			require.NoError(t, err)
			a, err := r.Evaluate(tt.Update)
			assert.Equal(t, tt.zoneWant.ZoneState, ZoneState(a.GetState()))
			assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
			assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			tt.zoneWant.err(t, err)
		})
	}
}

func TestZoneRule_UseCases(t *testing.T) {
	tests := []struct {
		name   string
		script string
		Update
		zoneWant
	}{
		{
			name:     "limitOverlay - auto",
			script:   "limitoverlay.lua",
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}}, Devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name:     "limitOverlay - manual",
			script:   "limitoverlay.lua",
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateOff}}, Devices{}},
			zoneWant: zoneWant{ZoneStateAuto, time.Hour, "manual setting detected", assert.NoError},
		},
		{
			name:     "autoAway - home",
			script:   "autoaway.lua",
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}}, Devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "at least one user is home", assert.NoError},
		},
		{
			name:     "autoAway - away",
			script:   "autoaway.lua",
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}}, Devices{{Name: "user", Home: false}}},
			zoneWant: zoneWant{ZoneStateOff, 15 * time.Minute, "all users are away", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := zonerules.FS.Open(tt.script)
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })
			r, err := NewZoneRule("foo", f)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.Update)
			assert.Equal(t, tt.zoneWant.ZoneState, ZoneState(a.GetState()))
			assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
			assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			assert.NoError(t, err)
		})
	}
}

func TestZoneRule_UseCases_Nighttime(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		Update
		zoneWant
	}{
		{
			name:     "no manual setting",
			now:      time.Date(2024, time.November, 26, 12, 0, 0, 0, time.Local),
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}}, Devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name:     "nightTime",
			now:      time.Date(2024, time.November, 26, 1, 0, 0, 0, time.Local),
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateManual}}, Devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "manual setting detected", assert.NoError},
		},
		{
			name:     "daytime",
			now:      time.Date(2024, time.November, 26, 12, 0, 0, 0, time.Local),
			Update:   Update{HomeStateAuto, 1, map[string]ZoneInfo{"foo": {ZoneState: ZoneStateManual}}, Devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 12 * time.Hour, "manual setting detected", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := zonerules.FS.Open("nighttime.lua")
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })
			r, err := NewZoneRule("foo", f)
			require.NoError(t, err)

			// re-register functions with custom "now" function
			luart.Register(r.State, func() time.Time { return tt.now })

			a, err := r.Evaluate(tt.Update)
			assert.Equal(t, tt.zoneWant.ZoneState, ZoneState(a.GetState()))
			assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
			assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			assert.NoError(t, err)
		})
	}
}

func BenchmarkZoneEvaluator(b *testing.B) {
	f, err := zonerules.FS.Open("nighttime.lua")
	require.NoError(b, err)
	b.Cleanup(func() { _ = f.Close() })
	r, err := NewZoneRule("foo", f)
	require.NoError(b, err)
	u := Update{
		HomeState:  HomeStateAuto,
		ZoneStates: map[string]ZoneInfo{"foo": {ZoneState: ZoneStateAuto}},
		Devices:    Devices{},
	}
	b.ResetTimer()
	for range b.N {
		if _, err := r.Evaluate(u); err != nil {
			b.Fatal(err)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_zoneAction(t *testing.T) {
	a := zoneAction{
		zoneState: ZoneStateOff,
		delay:     5 * time.Minute,
		reason:    "reasons",
		zoneName:  "foo",
	}

	assert.Equal(t, "foo: setting heating to off mode", a.Description(false))
	assert.Equal(t, "foo: setting heating to off mode in 5m0s", a.Description(true))
	assert.Equal(t, "[zone=foo mode=off delay=5m0s reason=reasons]", a.LogValue().String())
}

func Test_zoneAction_Do(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		action zoneAction
		setup  func(*mocks.TadoClient)
		err    assert.ErrorAssertionFunc
	}{
		{
			name: "auto mode - pass",
			action: zoneAction{
				zoneState: ZoneStateAuto,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().
					DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
					Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
					Once()
			},
			err: assert.NoError,
		},
		{
			name: "auto mode - fail",
			action: zoneAction{
				zoneState: ZoneStateAuto,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().
					DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
					Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized}}, nil).
					Once()
			},
			err: assert.Error,
		},
		{
			name: "off mode - pass",
			action: zoneAction{
				zoneState: ZoneStateOff,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					RunAndReturn(func(ctx context.Context, i int64, i2 int, overlay tado.ZoneOverlay, fn ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if *overlay.Setting.Power != tado.PowerOFF {
							return nil, errors.New("invalid power setting")
						}
						if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
							return nil, errors.New("invalid termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
			},
			err: assert.NoError,
		},
		{
			name: "off mode - fail",
			action: zoneAction{
				zoneState: ZoneStateOff,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					Return(&tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized}}, nil).
					Once()
			},
			err: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := mocks.NewTadoClient(t)
			if tt.setup != nil {
				tt.setup(client)
			}
			tt.err(t, tt.action.Do(ctx, client))
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Evaluator = fakeZoneEvaluator{}

type fakeZoneEvaluator struct {
	ZoneState
	delay  time.Duration
	reason string
	err    error
}

func (f fakeZoneEvaluator) Evaluate(_ Update) (Action, error) {
	return zoneAction{zoneState: f.ZoneState, delay: f.delay, reason: f.reason}, f.err
}
