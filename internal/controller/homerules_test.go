package controller

import (
	"context"
	"fmt"
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

func TestHomeRule_Evaluate(t *testing.T) {
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
function Evaluate(state, devices)
	return { Overlay = state.Overlay, Home = state.Home}, 300, "test"
end
`,
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, true)),
			),
			want: &homeAction{coreAction{homeState{false, true}, "test", 5 * time.Minute}, 1},
			err:  assert.NoError,
		},
		{
			name: "invalid state",
			script: `
			function Evaluate(state, devices)
				return "foo", nil, "test"
			end
			`,
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, true)),
			),
			err: assert.Error,
		},
		{
			name: "invalid delay",
			script: `
			function Evaluate(state, devices)
				return state, nil, "test"
			end
			`,
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, true)),
			),
			err: assert.Error,
		},
		{
			name: "missing Evaluate function",
			script: `
			function NotEvaluate(state, devices)
				return state, 0, "test"
			end
			`,
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, true)),
			),
			err: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newHomeRule(tt.name, strings.NewReader(tt.script), nil, nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, a)
		})
	}
}

func TestHomeRule_Evaluate_AutoAway(t *testing.T) {
	tests := []struct {
		name   string
		users  []string
		update poller.Update
		want   action
		err    assert.ErrorAssertionFunc
	}{
		{
			name:  "home mode, at least one user home",
			users: []string{"user A", "user B"},
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
				testutils.WithMobileDevice(101, "user B", testutils.WithLocation(false, false)),
			),
			want: &homeAction{
				coreAction: coreAction{
					state:  homeState{false, true},
					reason: "one or more users are home: user A",
					delay:  0,
				},
				homeId: 1,
			},
			err: assert.NoError,
		},
		{
			name:  "home mode, all users away",
			users: []string{"user A", "user B"},
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
				testutils.WithMobileDevice(101, "user B", testutils.WithLocation(false, false)),
			),
			want: &homeAction{
				coreAction: coreAction{
					state:  homeState{true, false},
					reason: "all users are away: user A, user B",
					delay:  5 * time.Minute,
				},
				homeId: 1,
			},
			err: assert.NoError,
		},
		{
			name:  "only consider selected users",
			users: []string{"user A"},
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
				testutils.WithMobileDevice(101, "user B", testutils.WithLocation(true, false)),
			),
			want: &homeAction{
				coreAction: coreAction{
					state:  homeState{true, false},
					reason: "all users are away: user A",
					delay:  5 * time.Minute,
				},
				homeId: 1,
			},
			err: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := RuleConfiguration{
				Name:   "test",
				Script: ScriptConfig{Packaged: "homeandaway.lua"},
				Users:  tt.users,
			}
			r, err := loadHomeRule(cfg)
			require.NoError(t, err)

			got, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestHomeAction_Do(t *testing.T) {
	tadoClient := mocks.NewTadoClient(t)
	ctx := context.Background()

	a := homeAction{coreAction{homeState{true, true}, "test", 15 * time.Minute}, 1}
	tadoClient.EXPECT().
		SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
		RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
			if lock.HomePresence == nil {
				return nil, fmt.Errorf("missing home presence")
			}
			if *lock.HomePresence != tado.HOME {
				return nil, fmt.Errorf("wrong home presence: wanted %v, got %v", tado.HOME, *lock.HomePresence)
			}
			return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil
		}).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient, discardLogger))

	a = homeAction{coreAction{homeState{true, false}, "test", 15 * time.Minute}, 1}
	tadoClient.EXPECT().
		SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
		RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
			if lock.HomePresence == nil {
				return nil, fmt.Errorf("missing home presence")
			}
			if *lock.HomePresence != tado.AWAY {
				return nil, fmt.Errorf("wrong home presence: got %v, wanted %v", *lock.HomePresence, tado.AWAY)
			}
			return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil
		}).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient, discardLogger))

	a = homeAction{coreAction{homeState{false, true}, "test", 15 * time.Minute}, 1}
	tadoClient.EXPECT().
		DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.DeletePresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient, discardLogger))
}

func TestHomeAction_LogValue(t *testing.T) {
	h := homeAction{
		coreAction: coreAction{zoneState{true, true}, "foo", 5 * time.Minute},
		homeId:     1,
	}
	assert.Equal(t, `[action=[state=[overlay=true heating=true] delay=5m0s reason=foo]]`, h.LogValue().String())
}
