package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
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
		{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"A"}},
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

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_homeAction(t *testing.T) {
	h := homeAction{
		state:  HomeStateAway,
		delay:  time.Hour,
		reason: "reasons",
	}

	assert.Equal(t, "Setting home to away mode", h.Description(false))
	assert.Equal(t, "Setting home to away mode in 1h0m0s", h.Description(true))
	assert.Equal(t, "[action=away delay=1h0m0s reason=reasons]", h.LogValue().String())
}

func Test_homeAction_Do(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		action homeAction
		setup  func(tadoClient *mocks.TadoClient)
		err    assert.ErrorAssertionFunc
	}{
		{
			name:   "auto mode - pass",
			action: homeAction{state: HomeStateAuto, homeId: 1},
			setup: func(tadoClient *mocks.TadoClient) {
				tadoClient.EXPECT().
					DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
					Return(&tado.DeletePresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
					Once()
			},
			err: assert.NoError,
		},
		{
			name:   "auto mode - fail",
			action: homeAction{state: HomeStateAuto, homeId: 1},
			setup: func(tadoClient *mocks.TadoClient) {
				tadoClient.EXPECT().
					DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
					Return(&tado.DeletePresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized}}, nil).
					Once()
			},
			err: assert.Error,
		},
		{
			name:   "away mode - pass",
			action: homeAction{state: HomeStateAway, homeId: 1},
			setup: func(tadoClient *mocks.TadoClient) {
				tadoClient.EXPECT().
					SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
					RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, fn ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.AWAY {
							return nil, fmt.Errorf("unexpected home presence")
						}
						return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil
					}).
					Once()
			},
			err: assert.NoError,
		},
		{
			name:   "away mode - fail",
			action: homeAction{state: HomeStateAway, homeId: 1},
			setup: func(tadoClient *mocks.TadoClient) {
				tadoClient.EXPECT().
					SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
					Return(&tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized}}, nil).
					Once()
			},
			err: assert.Error,
		},
		{
			name:   "home mode - pass",
			action: homeAction{state: HomeStateHome, homeId: 1},
			setup: func(tadoClient *mocks.TadoClient) {
				tadoClient.EXPECT().
					SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
					RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, fn ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
						if *lock.HomePresence != tado.HOME {
							return nil, fmt.Errorf("unexpected home presence")
						}
						return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil
					}).
					Once()
			},
			err: assert.NoError,
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
