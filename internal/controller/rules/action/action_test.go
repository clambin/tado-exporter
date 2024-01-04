package action_test

import (
	"bytes"
	"github.com/clambin/tado-exporter/internal/controller/internal/testutil"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAction(t *testing.T) {
	type want struct {
		isAction assert.BoolAssertionFunc
		logValue string
		asString string
	}
	testCases := []struct {
		name   string
		action action.Action
		want
	}{
		{
			name:   "no action",
			action: action.Action{Reason: "test"},
			want: want{
				isAction: assert.False,
				logValue: `level=INFO msg=action action.action=false action.reason=test
`,
				asString: "no action",
			},
		},
		{
			name:   "action",
			action: action.Action{State: testutil.FakeState{ModeValue: action.HomeInAwayMode}, Reason: "test", Delay: time.Hour},
			want: want{
				isAction: assert.True,
				logValue: `level=INFO msg=action action.action=true action.reason=test action.delay=1h0m0s action.state.mode=away
`,
				asString: "away",
			},
		},
		{
			name:   "action (label)",
			action: action.Action{State: testutil.FakeState{ModeValue: action.HomeInAwayMode}, Reason: "test", Delay: time.Hour, Label: "room"},
			want: want{
				isAction: assert.True,
				logValue: `level=INFO msg=action action.action=true action.reason=test action.label=room action.delay=1h0m0s action.state.mode=away
`,
				asString: "away",
			},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logOutput bytes.Buffer
			logger := testutil.NewBufferLogger(&logOutput)
			logger.Info("action", "action", tt.action)

			tt.want.isAction(t, tt.action.IsAction())
			assert.Equal(t, tt.want.logValue, logOutput.String())
			assert.Equal(t, tt.want.asString, tt.action.String())
		})
	}
}
