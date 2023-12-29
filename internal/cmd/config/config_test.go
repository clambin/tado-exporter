package config_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/cmd/config"
	"github.com/clambin/tado-exporter/internal/cmd/config/mocks"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestShowConfig(t *testing.T) {
	type zonesResult struct {
		zones []tado.Zone
		err   error
	}
	type mobileDevicesResult struct {
		mobileDevices []tado.MobileDevice
		err           error
	}

	testCases := []struct {
		name string
		zonesResult
		mobileDevicesResult
		wantErr  assert.ErrorAssertionFunc
		wantJSON string
		wantYAML string
	}{
		{
			name:                "pass",
			zonesResult:         zonesResult{[]tado.Zone{{ID: 1, Name: "room"}}, nil},
			mobileDevicesResult: mobileDevicesResult{[]tado.MobileDevice{{ID: 1000, Name: "user"}}, nil},
			wantErr:             assert.NoError,
			wantJSON: `{"Zones":[{"ID":1,"Name":"room"}],"Devices":[{"ID":1000,"Name":"user"}]}
`,
			wantYAML: `zones:
    - id: 1
      name: room
devices:
    - id: 1000
      name: user
`,
		},
		{
			name:        "zones fail",
			zonesResult: zonesResult{nil, errors.New("fail")},
			wantErr:     assert.Error,
		},
		{
			name:                "mobileDevices fail",
			zonesResult:         zonesResult{[]tado.Zone{{ID: 1, Name: "room"}}, nil},
			mobileDevicesResult: mobileDevicesResult{nil, errors.New("fail")},
			wantErr:             assert.Error,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			c := mocks.NewTadoGetter(t)
			c.EXPECT().GetZones(ctx).Return(tt.zonesResult.zones, tt.zonesResult.err).Maybe()
			c.EXPECT().GetMobileDevices(ctx).Return(tt.mobileDevicesResult.mobileDevices, tt.mobileDevicesResult.err).Maybe()

			var out bytes.Buffer
			e1 := yaml.NewEncoder(&out)
			err := config.ShowConfig(ctx, c, e1)
			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.wantYAML, out.String())
			}

			out.Reset()

			e2 := json.NewEncoder(&out)
			err = config.ShowConfig(ctx, c, e2)
			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.wantJSON, out.String())
			}
		})
	}

}
