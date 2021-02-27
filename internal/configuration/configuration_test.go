package configuration_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestFullConfigurationFile(t *testing.T) {
	var testRules = []byte(`
debug: true

exporter:
  enabled: true
  port: 8080
  interval: 30s

controller:
  enabled: true
  interval: 5m
  tadoBot:
    enabled: true
    token:
      value: "1234"
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 2h
  - zoneID: 1
    maxTime: 30m
  autoAwayRules:
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

	cfg, err := configuration.LoadConfigurationFile(f.Name())

	if assert.Nil(t, err) {
		assert.True(t, cfg.Debug)

		assert.True(t, cfg.Exporter.Enabled)
		assert.Equal(t, 8080, cfg.Exporter.Port)
		assert.Equal(t, 30*time.Second, cfg.Exporter.Interval)

		assert.True(t, cfg.Controller.Enabled)
		assert.Equal(t, 5*time.Minute, cfg.Controller.Interval)
		assert.True(t, cfg.Controller.TadoBot.Enabled)
		assert.Equal(t, "1234", cfg.Controller.TadoBot.Token.Value)

		if assert.NotNil(t, cfg.Controller.OverlayLimitRules) &&
			assert.Len(t, *cfg.Controller.OverlayLimitRules, 2) {
			assert.Equal(t, "foo", (*cfg.Controller.OverlayLimitRules)[0].ZoneName)
			assert.Equal(t, 0, (*cfg.Controller.OverlayLimitRules)[0].ZoneID)
			assert.Equal(t, 2*time.Hour, (*cfg.Controller.OverlayLimitRules)[0].MaxTime)

			assert.Equal(t, "", (*cfg.Controller.OverlayLimitRules)[1].ZoneName)
			assert.Equal(t, 1, (*cfg.Controller.OverlayLimitRules)[1].ZoneID)
			assert.Equal(t, 30*time.Minute, (*cfg.Controller.OverlayLimitRules)[1].MaxTime)
		}

		if assert.NotNil(t, cfg.Controller.AutoAwayRules) &&
			assert.Len(t, *cfg.Controller.AutoAwayRules, 2) {
			assert.Equal(t, "foo", (*cfg.Controller.AutoAwayRules)[0].ZoneName)
			assert.Equal(t, 0, (*cfg.Controller.AutoAwayRules)[0].ZoneID)
			assert.Equal(t, "bar", (*cfg.Controller.AutoAwayRules)[0].MobileDeviceName)
			assert.Equal(t, 0, (*cfg.Controller.AutoAwayRules)[0].MobileDeviceID)
			assert.Equal(t, 1*time.Hour, (*cfg.Controller.AutoAwayRules)[0].WaitTime)
			assert.Equal(t, 5.0, (*cfg.Controller.AutoAwayRules)[0].TargetTemperature)

			assert.Equal(t, "", (*cfg.Controller.AutoAwayRules)[1].ZoneName)
			assert.Equal(t, 2, (*cfg.Controller.AutoAwayRules)[1].ZoneID)
			assert.Equal(t, "", (*cfg.Controller.AutoAwayRules)[1].MobileDeviceName)
			assert.Equal(t, 20, (*cfg.Controller.AutoAwayRules)[1].MobileDeviceID)
			assert.Equal(t, 2*time.Hour, (*cfg.Controller.AutoAwayRules)[1].WaitTime)
			assert.Equal(t, 18.0, (*cfg.Controller.AutoAwayRules)[1].TargetTemperature)
		}
	}
}

func TestConfigLoader_Defaults(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(``))

	assert.Nil(t, err)

	assert.False(t, cfg.Debug)
	assert.True(t, cfg.Exporter.Enabled)
	assert.Equal(t, 8080, cfg.Exporter.Port)
	assert.Equal(t, 1*time.Minute, cfg.Exporter.Interval)
	assert.False(t, cfg.Controller.Enabled)
	assert.Equal(t, 5*time.Minute, cfg.Controller.Interval)
	assert.False(t, cfg.Controller.TadoBot.Enabled)
	assert.Nil(t, cfg.Controller.AutoAwayRules)
	assert.Nil(t, cfg.Controller.OverlayLimitRules)
}

func TestConfigLoader_Tadobot(t *testing.T) {
	var testConfig = []byte(`
debug: true
controller:
  enabled: true
  interval: 5m
  tadoBot:
    enabled: true
    token:
      envVar: "SLACKBOT_TOKEN"
`)

	var (
		err error
		cfg *configuration.Configuration
	)

	err = os.Setenv("SLACKBOT_TOKEN", "")
	if assert.Nil(t, err) {
		cfg, err = configuration.LoadConfiguration(testConfig)

		if assert.NotNil(t, cfg) {
			if assert.Nil(t, err) {
				assert.False(t, cfg.Controller.TadoBot.Enabled)
			}
		}

		_ = os.Setenv("SLACKBOT_TOKEN", "4321")

		cfg, err = configuration.LoadConfiguration(testConfig)

		if assert.NotNil(t, cfg) {
			if assert.Nil(t, err) {
				assert.True(t, cfg.Controller.TadoBot.Enabled)
				assert.Equal(t, "4321", cfg.Controller.TadoBot.Token.Value)
			}
		}
	}
}
