package controller

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
				return nil, fmt.Errorf("wrong home presence: wanted %v, got %v", tado.HOME, *lock.HomePresence)
			}
			return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil
		}).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient, discardLogger))

	a = homeAction{coreAction{homeState{true, false}, "test", 15 * time.Minute}, 1}
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

	a = homeAction{coreAction{homeState{false, true}, "test", 15 * time.Minute}, 1}
	tadoClient.EXPECT().
		DeletePresenceLockWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.DeletePresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	assert.NoError(t, a.Do(ctx, tadoClient, discardLogger))
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
			action: coreAction{state: zoneState{false, true}},
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
			action: coreAction{state: zoneState{true, false}},
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
			action: coreAction{state: zoneState{true, true}},
			err:    assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := zoneAction{tt.action, "foo", 1, 10}
			tt.err(t, a.Do(context.Background(), tt.tadoClient(t), discardLogger))
		})
	}
}

func TestHomeAction_LogValue(t *testing.T) {
	h := homeAction{
		coreAction: coreAction{zoneState{true, true}, "foo", 5 * time.Minute},
		homeId:     1,
	}
	assert.Equal(t, `[action=[state=[overlay=true heating=true] delay=5m0s reason=foo]]`, h.LogValue().String())
}

func TestZoneAction_LogValue(t *testing.T) {
	z := zoneAction{
		coreAction: coreAction{zoneState{true, true}, "foo", 5 * time.Minute},
		zoneName:   "zone",
		homeId:     1,
		zoneId:     10,
	}
	assert.Equal(t, `[zone=zone action=[state=[overlay=true heating=true] delay=5m0s reason=foo]]`, z.LogValue().String())
}

func TestCoreAction(t *testing.T) {
	type want struct {
		description    string
		descriptionDue string
		logValue       string
	}
	tests := []struct {
		name string
		coreAction
		want
	}{
		{
			name:       "home: auto",
			coreAction: coreAction{homeState{false, false}, "test", 15 * time.Minute},
			want:       want{"AUTO mode", "AUTO mode in 15m0s", "[state=[overlay=false home=false] delay=15m0s reason=test]"},
		},
		{
			name:       "home: auto",
			coreAction: coreAction{homeState{false, true}, "test", 15 * time.Minute},
			want:       want{"AUTO mode", "AUTO mode in 15m0s", "[state=[overlay=false home=true] delay=15m0s reason=test]"},
		},
		{
			name:       "home: manual away",
			coreAction: coreAction{homeState{true, false}, "test", 15 * time.Minute},
			want:       want{"AWAY mode", "AWAY mode in 15m0s", "[state=[overlay=true home=false] delay=15m0s reason=test]"},
		},
		{
			name:       "home: manual home",
			coreAction: coreAction{homeState{true, true}, "test", 15 * time.Minute},
			want:       want{"HOME mode", "HOME mode in 15m0s", "[state=[overlay=true home=true] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: auto mode",
			coreAction: coreAction{zoneState{false, false}, "test", 15 * time.Minute},
			want:       want{"to auto mode", "to auto mode in 15m0s", "[state=[overlay=false heating=false] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: auto mode",
			coreAction: coreAction{zoneState{false, true}, "test", 15 * time.Minute},
			want:       want{"to auto mode", "to auto mode in 15m0s", "[state=[overlay=false heating=true] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: heating off",
			coreAction: coreAction{zoneState{true, false}, "test", 15 * time.Minute},
			want:       want{"off", "off in 15m0s", "[state=[overlay=true heating=false] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: heating on",
			coreAction: coreAction{zoneState{true, true}, "test", 15 * time.Minute},
			want:       want{"on", "on in 15m0s", "[state=[overlay=true heating=true] delay=15m0s reason=test]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.description, tt.coreAction.Description(false))
			assert.Equal(t, tt.want.descriptionDue, tt.coreAction.Description(true))
			assert.Equal(t, tt.want.logValue, tt.coreAction.LogValue().String())
		})
	}
}
