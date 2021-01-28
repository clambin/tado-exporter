package controller_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller"
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

	ctrlr := controller.Controller{
		API:           &mockapi.MockAPI{},
		Configuration: &cfg.Controller,
	}

	err = ctrlr.Run()
	assert.Nil(t, err)
	assert.Len(t, ctrlr.Overlays, 1)
	_, ok := ctrlr.Overlays[2]
	assert.True(t, ok)

	// Overlay's been running more than the expiry time
	ctrlr.Overlays[2] = time.Now().Add(-2 * time.Hour)

	err = ctrlr.Run()
	assert.Nil(t, err)
	assert.Len(t, ctrlr.Overlays, 0)
}
