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
	testCases := []struct {
		name         string
		action       action.Action
		wantIsAction assert.BoolAssertionFunc
		wantLogValue string
		wantString   string
	}{
		{
			name:         "no action",
			action:       action.Action{Reason: "test"},
			wantIsAction: assert.False,
			wantLogValue: `level=INFO msg=action action.action=false action.reason=test
`,
			wantString: "no action",
		},
		{
			name:         "action",
			action:       action.Action{State: testutil.FakeState{ModeValue: action.HomeInAwayMode}, Reason: "test", Delay: time.Hour},
			wantIsAction: assert.True,
			wantLogValue: `level=INFO msg=action action.action=true action.reason=test action.delay=1h0m0s action.state.mode=away
`,
			wantString: "away",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logOutput bytes.Buffer
			logger := testutil.NewBufferLogger(&logOutput)
			logger.Info("action", "action", tt.action)

			tt.wantIsAction(t, tt.action.IsAction())
			assert.Equal(t, tt.wantLogValue, logOutput.String())
			assert.Equal(t, tt.wantString, tt.action.String())
		})
	}
}
