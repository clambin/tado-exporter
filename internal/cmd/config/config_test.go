package config_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/cmd/config"
	"github.com/clambin/tado-exporter/internal/cmd/config/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestShowConfig(t *testing.T) {
	ctx := context.Background()
	c := mocks.NewTadoGetter(t)
	c.EXPECT().GetZones(ctx).Return([]tado.Zone{{ID: 1, Name: "room"}}, nil)
	c.EXPECT().GetMobileDevices(ctx).Return([]tado.MobileDevice{{ID: 1000, Name: "user"}}, nil)

	var out bytes.Buffer
	e1 := yaml.NewEncoder(&out)
	err := config.ShowConfig(ctx, c, e1)
	require.NoError(t, err)
	assert.Equal(t, `zones:
    - id: 1
      name: room
devices:
    - id: 1000
      name: user
`, out.String())

	out.Reset()
	e2 := json.NewEncoder(&out)
	err = config.ShowConfig(ctx, c, e2)
	require.NoError(t, err)
	assert.Equal(t, `{"Zones":[{"ID":1,"Name":"room"}],"Devices":[{"ID":1000,"Name":"user"}]}
`, out.String())

}
