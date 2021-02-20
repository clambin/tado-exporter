package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/tadoproxy"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOverlayLimit(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 1h
  - zoneName: "bar"
    maxTime: 1h
  - zoneName: "not-a-zone"
    maxTime: 1h
`))

	if err != nil {
		panic(err)
	}

	controller := Controller{
		Configuration: &cfg.Controller,
		proxy: tadoproxy.Proxy{
			API: &mockapi.MockAPI{},
		},
	}

	err = controller.Run()
	assert.Nil(t, err)
	assert.Len(t, controller.Overlays, 1)
	_, ok := controller.Overlays[2]
	assert.True(t, ok)

	// Overlay's been running more than the expiry time
	controller.Overlays[2] = time.Now().Add(-2 * time.Hour)

	err = controller.Run()
	assert.Nil(t, err)
	assert.Len(t, controller.Overlays, 0)
}
