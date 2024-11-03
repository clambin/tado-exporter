package zone

import (
	"bytes"
	"context"
	"errors"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState(t *testing.T) {
	tests := []struct {
		name       string
		state      State
		wantString string
		wantLog    string
	}{
		{
			name: "auto mode",
			state: State{
				zoneID:   10,
				zoneName: "room",
				mode:     action.ZoneInAutoMode,
			},
			wantString: `moving to auto mode`,
			wantLog: `level=INFO msg=state state.type=zone state.name=room state.mode=auto
`,
		},
		{
			name: "overlay mode",
			state: State{
				zoneID:          10,
				zoneName:        "room",
				mode:            action.ZoneInOverlayMode,
				zoneTemperature: 18,
			},
			wantString: `heating to 18.0º`,
			wantLog: `level=INFO msg=state state.type=zone state.name=room state.mode=overlay state.temperature=18
`,
		},
		{
			name: "off",
			state: State{
				zoneID:          10,
				zoneName:        "room",
				mode:            action.ZoneInOverlayMode,
				zoneTemperature: 5,
			},
			wantString: `switching off heating`,
			wantLog: `level=INFO msg=state state.type=zone state.name=room state.mode=overlay state.temperature=5
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logOutput bytes.Buffer
			logger := testutil.NewBufferLogger(&logOutput)
			logger.Info("state", "state", tt.state)

			assert.Equal(t, tt.wantString, tt.state.String())
			assert.Equal(t, tt.wantLog, logOutput.String())
		})
	}
}

func TestState_IsEqual(t *testing.T) {
	t1 := State{
		zoneID:          10,
		zoneName:        "room",
		mode:            action.ZoneInOverlayMode,
		zoneTemperature: 18,
	}
	t2 := State{
		zoneID:   10,
		zoneName: "room",
		mode:     action.ZoneInAutoMode,
	}
	assert.True(t, t1.IsEqual(t1))
	assert.False(t, t1.IsEqual(t2))
}

func TestState_Do(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		state   State
		client  fakeClient
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "invalid mode",
			state: State{
				homeId: 1,
				zoneID: 10,
				mode:   action.NoAction,
			},
			client:  fakeClient{},
			wantErr: assert.Error,
		},
		{
			name: "set overlay",
			state: State{
				homeId:          1,
				zoneID:          10,
				mode:            action.ZoneInOverlayMode,
				zoneTemperature: 15,
			},
			client:  fakeClient{expect: "set"},
			wantErr: assert.NoError,
		},
		{
			name: "delete overlay",
			state: State{
				homeId: 1,
				zoneID: 10,
				mode:   action.ZoneInAutoMode,
			},
			client:  fakeClient{expect: "delete"},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, tt.state.Do(ctx, tt.client))
		})
	}
}

var _ action.TadoClient = fakeClient{}

type fakeClient struct {
	expect string
}

func (f fakeClient) SetPresenceLockWithResponse(_ context.Context, _ tado.HomeId, _ tado.SetPresenceLockJSONRequestBody, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
	// not used in this package
	panic("implement me")
}

func (f fakeClient) DeleteZoneOverlayWithResponse(_ context.Context, homeId tado.HomeId, zoneId tado.ZoneId, _ ...tado.RequestEditorFn) (*tado.DeleteZoneOverlayResponse, error) {
	if homeId != 1 || zoneId != 10 || f.expect != "delete" {
		return nil, errors.New("invalid request")
	}
	return nil, nil
}

func (f fakeClient) SetZoneOverlayWithResponse(_ context.Context, homeId tado.HomeId, zoneId tado.ZoneId, _ tado.SetZoneOverlayJSONRequestBody, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
	if homeId != 1 || zoneId != 10 || f.expect != "set" {
		return nil, errors.New("invalid request")
	}
	return nil, nil
}