package controller

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type homeWant struct {
	state  homeState
	delay  time.Duration
	reason string
	err    assert.ErrorAssertionFunc
}

func TestHomeRules_Evaluate(t *testing.T) {
	tests := []struct {
		name string
		update
		homeWant
	}{
		{
			name: "at least one user home",
			update: update{HomeStateAway, 1, nil, devices{
				{Name: "user A", Home: true},
				{Name: "user B", Home: false},
			}},
			homeWant: homeWant{HomeStateHome, 0, "one or more users are home: user A", assert.NoError},
		},
		{
			name: "all users are away",
			update: update{HomeStateHome, 1, nil, devices{
				{Name: "user A", Home: false},
				{Name: "user B", Home: false},
			}},
			homeWant: homeWant{HomeStateAway, 5 * time.Minute, "all users are away", assert.NoError},
		},
		{
			name:     "no devices",
			update:   update{HomeStateAway, 1, nil, devices{}},
			homeWant: homeWant{HomeStateAway, 0, "no devices found", assert.NoError},
		},
	}

	rules, err := loadHomeRules([]RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"user A"}},
	})
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := rules.Evaluate(tt.update)
			tt.homeWant.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.homeWant.state, homeState(a.GetState()))
			assert.Equal(t, tt.homeWant.delay, a.GetDelay())
			assert.Equal(t, tt.homeWant.reason, a.GetReason())
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestHomeRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update
		homeWant
	}{
		{
			name: "success",
			script: `
function Evaluate(state, devices)
	return "away", 300, "test"
end
`,
			update:   update{HomeStateHome, 1, nil, devices{{Name: "user", Home: false}}},
			homeWant: homeWant{HomeStateAway, 5 * time.Minute, "test", assert.NoError},
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(state, devices)
	return "away", nil, "test"
end
`,
			update:   update{HomeStateHome, 1, nil, devices{{Name: "user", Home: false}}},
			homeWant: homeWant{"", 0, "", assert.Error},
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(state, devices)
	return "away", 0, "test"
end
`,
			update:   update{HomeStateHome, 1, nil, devices{{Name: "user", Home: false}}},
			homeWant: homeWant{"", 0, "", assert.Error},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newHomeRule(tt.name, strings.NewReader(tt.script), nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.homeWant.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.homeWant.state, homeState(a.GetState()))
			assert.Equal(t, tt.homeWant.delay, a.GetDelay())
			assert.Equal(t, tt.homeWant.reason, a.GetReason())
		})
	}
}
