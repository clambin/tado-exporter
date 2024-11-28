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
