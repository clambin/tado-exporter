package controller_test

import (
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/containrrr/shoutrrr"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOverlayLimit(t *testing.T) {
	rules, err := controller.ParseRules([]byte(`
overlayLimit:
  - zoneName: "foo"
    maxTime: 1h
  - zoneName: "bar"
    maxTime: 1h
  - zoneName: "not-a-zone"
    maxTime: 1h
`))

	ctrlr := controller.Controller{
		API:           &mockapi.MockAPI{},
		Configuration: &controller.Configuration{},
		Rules:         rules,
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

func TestSlackNotification(t *testing.T) {
	url := "slack://tado-controller@T01J50B58E8/B01LHL7NHH6/XCJ00nU3Cr5qo2RWJcqKatVb"
	err := shoutrrr.Send(url, "Hello world!")

	assert.Nil(t, err)
}
