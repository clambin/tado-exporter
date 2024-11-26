package controller

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type homeWant struct {
	state  HomeState
	delay  time.Duration
	reason string
	err    assert.ErrorAssertionFunc
}

func TestHomeRules_Evaluate(t *testing.T) {
	tests := []struct {
		name string
		Update
		homeWant
	}{
		{
			name: "at least one user home",
			Update: Update{HomeStateAway, 1, nil, Devices{
				{Name: "user A", Home: true},
				{Name: "user B", Home: false},
			}},
			homeWant: homeWant{HomeStateHome, 0, "one or more users are home: user A", assert.NoError},
		},
		{
			name: "all users are away",
			Update: Update{HomeStateHome, 1, nil, Devices{
				{Name: "user A", Home: false},
				{Name: "user B", Home: false},
			}},
			homeWant: homeWant{HomeStateAway, 5 * time.Minute, "all users are away", assert.NoError},
		},
		{
			name:     "no devices",
			Update:   Update{HomeStateAway, 1, nil, Devices{}},
			homeWant: homeWant{HomeStateAway, 0, "no devices found", assert.NoError},
		},
	}

	rules, err := LoadHomeRules([]RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"A"}},
	})
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := rules.Evaluate(tt.Update)
			tt.homeWant.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.homeWant.state, HomeState(a.GetState()))
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
		Update
		homeWant
	}{
		{
			name: "success",
			script: `
function Evaluate(state, devices)
	return "away", 300, "test"
end
`,
			Update:   Update{HomeStateHome, 1, nil, Devices{{Name: "user", Home: false}}},
			homeWant: homeWant{HomeStateAway, 5 * time.Minute, "test", assert.NoError},
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(state, devices)
	return "away", nil, "test"
end
`,
			Update:   Update{HomeStateHome, 1, nil, Devices{{Name: "user", Home: false}}},
			homeWant: homeWant{"", 0, "", assert.Error},
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(state, devices)
	return "away", 0, "test"
end
`,
			Update:   Update{HomeStateHome, 1, nil, Devices{{Name: "user", Home: false}}},
			homeWant: homeWant{"", 0, "", assert.Error},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewHomeRule(tt.name, strings.NewReader(tt.script))
			require.NoError(t, err)
			a, err := r.Evaluate(tt.Update)
			tt.homeWant.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.homeWant.state, HomeState(a.GetState()))
			assert.Equal(t, tt.homeWant.delay, a.GetDelay())
			assert.Equal(t, tt.homeWant.reason, a.GetReason())
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestHomeAction(t *testing.T) {
	h := homeAction{
		state:  HomeStateAway,
		delay:  time.Hour,
		reason: "reasons",
	}

	assert.Equal(t, "Setting home to away mode", h.Description(false))
	assert.Equal(t, "Setting home to away mode in 1h0m0s", h.Description(true))
	assert.Equal(t, "[action=away delay=1h0m0s reason=reasons]", h.LogValue().String())
}
