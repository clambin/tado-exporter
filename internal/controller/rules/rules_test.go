package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRules_Evaluate(t *testing.T) {
	type want struct {
		action string
		reason string
		delay  time.Duration
	}
	testCases := []struct {
		name  string
		input Rules
		want
	}{
		{
			name: "no action",
			input: Rules{
				stubbedEvaluator{value: action.Action{Reason: "foo"}},
				stubbedEvaluator{value: action.Action{Reason: "bar"}},
			},
			want: want{
				action: "no action",
				reason: "bar, foo",
			},
		},
		{
			name: "single action",
			input: Rules{
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "foo", State: testutil.FakeState{ModeValue: 1}}},
			},
			want: want{
				action: "home",
				reason: "foo",
				delay:  time.Hour,
			},
		},
		{
			name: "multiple actions",
			input: Rules{
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "foo", State: testutil.FakeState{ModeValue: 1}}},
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "bar", State: testutil.FakeState{ModeValue: 1}}},
			},
			want: want{
				action: "home",
				reason: "bar, foo",
				delay:  time.Hour,
			},
		},
		{
			name: "duplicates",
			input: Rules{
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "foo", State: testutil.FakeState{ModeValue: 1}}},
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "bar", State: testutil.FakeState{ModeValue: 1}}},
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "foo", State: testutil.FakeState{ModeValue: 1}}},
			},
			want: want{
				action: "home",
				reason: "bar, foo",
				delay:  time.Hour,
			},
		},
		{
			name: "first action only",
			input: Rules{
				stubbedEvaluator{value: action.Action{Delay: time.Minute, Reason: "foo", State: testutil.FakeState{ModeValue: 1}}},
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "bar", State: testutil.FakeState{ModeValue: 2}}},
				stubbedEvaluator{value: action.Action{Delay: time.Hour, Reason: "foo", State: testutil.FakeState{ModeValue: 1}}},
			},
			want: want{
				action: "home",
				reason: "foo",
				delay:  time.Minute,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := tt.input.Evaluate(poller.Update{})
			require.NoError(t, err)
			assert.Equal(t, tt.want.action, a.String())
			assert.Equal(t, tt.want.reason, a.Reason)
			assert.Equal(t, tt.want.delay, a.Delay)
		})
	}
}

func BenchmarkRules_Evaluate(b *testing.B) {
	var r Rules
	for i := 0; i < 10; i++ {
		r = append(r, stubbedEvaluator{})
	}
	b.ResetTimer()
	for range b.N {
		_, err := r.Evaluate(poller.Update{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_getCombinedReasons(b *testing.B) {
	actions := make([]action.Action, 5)

	for range b.N {
		_ = getCombinedReason(actions)
	}
}

var _ Evaluator = stubbedEvaluator{}

type stubbedEvaluator struct {
	value action.Action
}

func (s stubbedEvaluator) Evaluate(_ poller.Update) (action.Action, error) {
	return s.value, nil
}
