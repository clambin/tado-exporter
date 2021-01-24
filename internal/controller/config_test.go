package controller

import (
	// "github.com/clambin/tado-exporter/internal/controller"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestRulesLoader(t *testing.T) {
	var testRules = []byte(`
overlayLimit:
  - zoneName: "foo"
    maxTime: 2h
  - zoneID: 1
    maxTime: 30m

autoAway:
  - zoneName: "foo"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 5.0
  - zoneID: 2
    mobileDeviceID: 20
    waitTime: 2h
    targetTemperature: 18.0
`)

	f, err := ioutil.TempFile("", "tmp")
	if err != nil {
		panic(err)
	}

	defer os.Remove(f.Name())
	_, _ = f.Write(testRules)
	_ = f.Close()

	configuration, err := ParseRulesFile(f.Name())

	if assert.Nil(t, err) {
		if assert.NotNil(t, configuration.OverlayLimit) &&
			assert.Len(t, *configuration.OverlayLimit, 2) {
			assert.Equal(t, "foo", (*configuration.OverlayLimit)[0].ZoneName)
			assert.Equal(t, 0, (*configuration.OverlayLimit)[0].ZoneID)
			assert.Equal(t, 2*time.Hour, (*configuration.OverlayLimit)[0].MaxTime)

			assert.Equal(t, "", (*configuration.OverlayLimit)[1].ZoneName)
			assert.Equal(t, 1, (*configuration.OverlayLimit)[1].ZoneID)
			assert.Equal(t, 30*time.Minute, (*configuration.OverlayLimit)[1].MaxTime)
		}

		if assert.NotNil(t, configuration.AutoAway) &&
			assert.Len(t, *configuration.AutoAway, 2) {
			assert.Equal(t, "foo", (*configuration.AutoAway)[0].ZoneName)
			assert.Equal(t, 0, (*configuration.AutoAway)[0].ZoneID)
			assert.Equal(t, "bar", (*configuration.AutoAway)[0].MobileDeviceName)
			assert.Equal(t, 0, (*configuration.AutoAway)[0].MobileDeviceID)
			assert.Equal(t, 1*time.Hour, (*configuration.AutoAway)[0].WaitTime)
			assert.Equal(t, 5.0, (*configuration.AutoAway)[0].TargetTemperature)

			assert.Equal(t, "", (*configuration.AutoAway)[1].ZoneName)
			assert.Equal(t, 2, (*configuration.AutoAway)[1].ZoneID)
			assert.Equal(t, "", (*configuration.AutoAway)[1].MobileDeviceName)
			assert.Equal(t, 20, (*configuration.AutoAway)[1].MobileDeviceID)
			assert.Equal(t, 2*time.Hour, (*configuration.AutoAway)[1].WaitTime)
			assert.Equal(t, 18.0, (*configuration.AutoAway)[1].TargetTemperature)
		}
	}
}

func TestConfigLoader_Empty(t *testing.T) {
	rules, err := ParseRules([]byte(``))

	assert.Nil(t, err)
	assert.Nil(t, rules.AutoAway)
	assert.Nil(t, rules.OverlayLimit)
}
