package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
	"time"
)

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

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_zoneAction(t *testing.T) {
	a := zoneAction{
		zoneState: ZoneStateOff,
		delay:     5 * time.Minute,
		reason:    "reasons",
		zoneName:  "foo",
	}

	assert.Equal(t, "*foo*: switching off heating", a.Description(false))
	assert.Equal(t, "*foo*: switching off heating in 5m0s", a.Description(true))
	assert.Equal(t, "[zone=foo mode=off delay=5m0s reason=reasons]", a.LogValue().String())
}

func Test_zoneAction_Do(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		action zoneAction
		setup  func(*mocks.TadoClient)
		err    assert.ErrorAssertionFunc
	}{
		{
			name: "auto mode - pass",
			action: zoneAction{
				zoneState: ZoneStateAuto,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().
					DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
					Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
					Once()
			},
			err: assert.NoError,
		},
		{
			name: "auto mode - fail",
			action: zoneAction{
				zoneState: ZoneStateAuto,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().
					DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
					Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized}}, nil).
					Once()
			},
			err: assert.Error,
		},
		{
			name: "off mode - pass",
			action: zoneAction{
				zoneState: ZoneStateOff,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					RunAndReturn(func(ctx context.Context, i int64, i2 int, overlay tado.ZoneOverlay, fn ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
						if *overlay.Setting.Type != tado.HEATING {
							return nil, errors.New("invalid type setting")
						}
						if *overlay.Setting.Power != tado.PowerOFF {
							return nil, errors.New("invalid power setting")
						}
						if *overlay.Termination.TypeSkillBasedApp != tado.ZoneOverlayTerminationTypeSkillBasedAppNEXTTIMEBLOCK {
							return nil, errors.New("invalid termination type")
						}
						return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
					}).
					Once()
			},
			err: assert.NoError,
		},
		{
			name: "off mode - fail",
			action: zoneAction{
				zoneState: ZoneStateOff,
				homeId:    1,
				zoneId:    10,
			},
			setup: func(client *mocks.TadoClient) {
				client.EXPECT().SetZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10), mock.AnythingOfType("tado.ZoneOverlay")).
					Return(&tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized}}, nil).
					Once()
			},
			err: assert.Error,
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
