package rules

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLoadHomeRules(t *testing.T) {
	r, err := LoadHomeRules([]RuleConfiguration{
		{Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"user A"}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, r.Count())
	_, ok := r.rules[0].(homeRule)
	assert.True(t, ok)
}

func TestHomeRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update poller.Update
		want   Action
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
			want: &homeAction{"test", 5 * time.Minute, tado.HomeId(1), HomeState{false, true}},
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
			r, err := LoadHomeRule(RuleConfiguration{Name: tt.name, Script: ScriptConfig{Text: tt.script}})
			require.NoError(t, err)

			s, err := GetHomeState(tt.update)
			require.NoError(t, err)

			a, err := r.Evaluate(s)
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
		want   Action
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
				HomeState: HomeState{false, true},
				reason:    "one or more users are home: user A",
				delay:     0,
				HomeId:    1,
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
				HomeState: HomeState{true, false},
				reason:    "all users are away: user A, user B",
				delay:     5 * time.Minute,
				HomeId:    1,
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
				HomeState: HomeState{true, false},
				reason:    "all users are away: user A",
				delay:     5 * time.Minute,
				HomeId:    1,
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
			r, err := LoadHomeRule(cfg)
			require.NoError(t, err)

			state, err := GetHomeState(tt.update)
			require.NoError(t, err)

			got, err := r.Evaluate(state)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
