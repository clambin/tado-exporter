package tmp

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
	"time"
)

func TestHomeAction_Do(t *testing.T) {
	tadoClient := mocks.NewTadoClient(t)
	ctx := context.Background()

	a := homeAction{coreAction{homeState{true, true}, "test", 15 * time.Minute}, 1}
	tadoClient.EXPECT().
		SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
		RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
			if lock.HomePresence == nil {
				return nil, fmt.Errorf("missing home presence")
			}
			if *lock.HomePresence != tado.HOME {
				return nil, fmt.Errorf("wrong home presence: got %v, wanted %v", *lock.HomePresence, tado.HOME)
			}
			return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil
		}).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient))

	a = homeAction{coreAction{homeState{false, true}, "test", 15 * time.Minute}, 1}
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
	assert.NoError(t, a.Do(ctx, tadoClient))

	a = homeAction{coreAction{homeState{true, false}, "test", 15 * time.Minute}, 1}
	tadoClient.EXPECT().
		DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.DeletePresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient))

}

func TestZoneAction_Do(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		tadoClient func(t *testing.T) *mocks.TadoClient
		action     coreAction
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
			action: coreAction{
				state: zoneState{true, false},
			},
			err: assert.NoError,
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
						if *overlay.Termination.TypeSkillBasedApp != tado.ZoneOverlayTerminationTypeSkillBasedAppNEXTTIMEBLOCK {
							return nil, fmt.Errorf("wrong termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
				return c
			},
			action: coreAction{
				state: zoneState{false, true},
			},
			err: assert.NoError,
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
						if *overlay.Termination.TypeSkillBasedApp != tado.ZoneOverlayTerminationTypeSkillBasedAppNEXTTIMEBLOCK {
							return nil, fmt.Errorf("wrong termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
				return c
			},
			action: coreAction{
				state: zoneState{true, true},
			},
			err: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := zoneAction{tt.action, "foo", 1, 10}
			tt.err(t, a.Do(context.Background(), tt.tadoClient(t)))
		})
	}

}

func TestCoreAction(t *testing.T) {
	type want struct {
		description    string
		descriptionDue string
		logValue       string
	}
	a := coreAction{homeState{true, false}, "test", 15 * time.Minute}
	w := want{
		description:    "setting home to HOME mode",
		descriptionDue: "setting home to HOME mode in 15m0s",
		logValue:       "[state={true false} delay=15m0s reason=test]",
	}
	assert.Equal(t, w.description, a.Description(false))
	assert.Equal(t, w.descriptionDue, a.Description(true))
	assert.Equal(t, w.logValue, a.LogValue().String())
}
