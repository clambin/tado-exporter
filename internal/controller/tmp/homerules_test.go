package tmp

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestHomeRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update
		want action
		err  assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			script: `
function Evaluate(state, devices)
	return { Home = state.Home, Manual = state.Manual}, 300, "test"
end
`,
			update: update{nil, devices{{Name: "user", Home: false}}, 1, homeState{true, false}},
			want:   &homeAction{coreAction{homeState{true, false}, "test", 5 * time.Minute}, 1},
			err:    assert.NoError,
		},
		{
			name: "invalid state",
			script: `
			function Evaluate(state, devices)
				return "foo", nil, "test"
			end
			`,
			update: update{nil, devices{{Name: "user", Home: false}}, 1, homeState{true, false}},
			err:    assert.Error,
		},
		{
			name: "invalid delay",
			script: `
			function Evaluate(state, devices)
				return state, nil, "test"
			end
			`,
			update: update{nil, devices{{Name: "user", Home: false}}, 1, homeState{true, false}},
			err:    assert.Error,
		},
		{
			name: "missing Evaluate function",
			script: `
			function NotEvaluate(state, devices)
				return state, 0, "test"
			end
			`,
			update: update{nil, devices{{Name: "user", Home: false}}, 1, homeState{true, false}},
			err:    assert.Error,
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
		name  string
		users []string
		update
		want action
		err  assert.ErrorAssertionFunc
	}{
		{
			name:  "home mode, at least one user home",
			users: []string{"user A", "user B"},
			update: update{
				homeState: homeState{true, false},
				HomeId:    1,
				devices:   devices{{Name: "user A", Home: true}, {Name: "user B", Home: false}},
			},
			want: &homeAction{
				coreAction: coreAction{
					state:  homeState{true, true},
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
			update: update{
				homeState: homeState{true, false},
				HomeId:    1,
				devices:   devices{{Name: "user A", Home: false}, {Name: "user B", Home: false}},
			},
			want: &homeAction{
				coreAction: coreAction{
					state:  homeState{false, true},
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
			update: update{
				homeState: homeState{true, false},
				HomeId:    1,
				devices:   devices{{Name: "user A", Home: false}, {Name: "user B", Home: true}},
			},
			want: &homeAction{
				coreAction: coreAction{
					state:  homeState{false, true},
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
