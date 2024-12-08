package rules

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
	"time"
)

func TestAction(t *testing.T) {
	type want struct {
		due              time.Duration
		reason           string
		shortDescription string
		longDescription  string
	}
	tests := []struct {
		name   string
		action Action
		want
	}{
		{
			name: "home",
			action: &homeAction{
				reason:    "manual setting detected",
				delay:     time.Hour,
				HomeId:    1,
				HomeState: HomeState{Overlay: false, Home: true},
			},
			want: want{
				due:              time.Hour,
				reason:           "manual setting detected",
				shortDescription: "setting home to HOME mode",
				longDescription:  "setting home to HOME mode in 1h0m0s",
			},
		},
		{
			name: "zone",
			action: &zoneAction{
				reason:    "manual setting detected",
				zoneName:  "zone",
				delay:     time.Hour,
				HomeId:    1,
				ZoneState: ZoneState{Overlay: false, Heating: true},
			},
			want: want{
				due:              time.Hour,
				reason:           "manual setting detected",
				shortDescription: "*zone*: switching heating to auto mode",
				longDescription:  "*zone*: switching heating to auto mode in 1h0m0s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.due, tt.action.Delay())
			assert.Equal(t, tt.want.reason, tt.action.Reason())
			assert.Equal(t, tt.want.shortDescription, tt.action.Description(false))
			assert.Equal(t, tt.want.longDescription, tt.action.Description(true))
		})
	}
}

func TestAction_IsState(t *testing.T) {
	var tests = []struct {
		name    string
		action  Action
		state   State
		isEqual assert.BoolAssertionFunc
	}{
		{
			name:    "home equal",
			action:  &homeAction{HomeState: HomeState{Overlay: false, Home: true}},
			state:   State{HomeState: HomeState{Overlay: false, Home: true}},
			isEqual: assert.True,
		},
		{
			name:    "home not equal",
			action:  &homeAction{HomeState: HomeState{Overlay: false, Home: true}},
			state:   State{},
			isEqual: assert.False,
		},
		{
			name:    "zone equal",
			action:  &zoneAction{ZoneState: ZoneState{Overlay: false, Heating: true}},
			state:   State{ZoneState: ZoneState{Overlay: false, Heating: true}},
			isEqual: assert.True,
		},
		{
			name:    "zone not equal",
			action:  &zoneAction{ZoneState: ZoneState{Overlay: false, Heating: true}},
			state:   State{},
			isEqual: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.isEqual(t, tt.action.IsState(tt.state))
		})
	}
}

func TestAction_IsActionState(t *testing.T) {
	var tests = []struct {
		name    string
		action  Action
		other   Action
		isEqual assert.BoolAssertionFunc
	}{
		{
			name:    "home equal",
			action:  &homeAction{HomeState: HomeState{Overlay: false, Home: true}},
			other:   &homeAction{HomeState: HomeState{Overlay: false, Home: true}},
			isEqual: assert.True,
		},
		{
			name:    "home not equal",
			action:  &homeAction{HomeState: HomeState{Overlay: false, Home: true}},
			other:   &homeAction{HomeState: HomeState{Overlay: true, Home: true}},
			isEqual: assert.False,
		},
		{
			name:    "zone equal",
			action:  &zoneAction{ZoneState: ZoneState{Overlay: false, Heating: true}},
			other:   &zoneAction{ZoneState: ZoneState{Overlay: false, Heating: true}},
			isEqual: assert.True,
		},
		{
			name:    "zone not equal",
			action:  &zoneAction{ZoneState: ZoneState{Overlay: false, Heating: true}},
			other:   &zoneAction{ZoneState: ZoneState{Overlay: true, Heating: true}},
			isEqual: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.isEqual(t, tt.action.IsActionState(tt.other))
		})
	}
}

func TestAction_Do_HomeAction(t *testing.T) {
	tadoClient := mocks.NewTadoClient(t)
	ctx := context.Background()

	a := homeAction{"test", 15 * time.Minute, 1, HomeState{true, true}}
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

	a = homeAction{"test", 15 * time.Minute, 1, HomeState{true, false}}
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

	a = homeAction{"test", 15 * time.Minute, 1, HomeState{false, true}}
	tadoClient.EXPECT().
		DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.DeletePresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient, discardLogger))
}

func TestAction_Do_ZoneAction(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		tadoClient func(t *testing.T) *mocks.TadoClient
		action     zoneAction
		err        assert.ErrorAssertionFunc
	}{
		{
			name: "move to auto mode",
			tadoClient: func(t *testing.T) *mocks.TadoClient {
				c := mocks.NewTadoClient(t)
				c.EXPECT().
					DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
					Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
					Once()
				return c
			},
			action: zoneAction{"", "zone", 0, 1, 10, ZoneState{false, true}},
			err:    assert.NoError,
		},
		{
			name: "switch heating off",
			tadoClient: func(t *testing.T) *mocks.TadoClient {
				c := mocks.NewTadoClient(t)
				c.EXPECT().
					SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					RunAndReturn(func(_ context.Context, _ int64, _ int, overlay tado.ZoneOverlay, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if *overlay.Setting.Type != tado.HEATING || *overlay.Setting.Power != tado.PowerOFF {
							return nil, fmt.Errorf("wrong settings")
						}
						if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
							return nil, fmt.Errorf("wrong termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
				return c
			},
			action: zoneAction{"", "zone", 0, 1, 10, ZoneState{true, false}},
			err:    assert.NoError,
		},
		{
			name: "switch heating on",
			tadoClient: func(t *testing.T) *mocks.TadoClient {
				c := mocks.NewTadoClient(t)
				c.EXPECT().
					SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					RunAndReturn(func(_ context.Context, _ int64, _ int, overlay tado.ZoneOverlay, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if *overlay.Setting.Type != tado.HEATING || *overlay.Setting.Power != tado.PowerON {
							return nil, fmt.Errorf("wrong settings")
						}
						if *overlay.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
							return nil, fmt.Errorf("wrong termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
				return c
			},
			action: zoneAction{"", "zone", 0, 1, 10, ZoneState{true, true}},
			err:    assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.err(t, tt.action.Do(context.Background(), tt.tadoClient(t), discardLogger))
		})
	}
}
