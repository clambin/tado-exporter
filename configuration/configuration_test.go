package configuration_test

import (
	"github.com/clambin/tado-exporter/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestFullConfigurationFile(t *testing.T) {
	var testRules = []byte(`
debug: true
interval: 5m

exporter:
  enabled: true
  port: 8080

controller:
  enabled: true
  tadoBot:
    enabled: true
    token:
      value: "1234"
  zones:
  - id: 1
    autoAway:
      enabled: true
      delay: 1h
      users:
       - id: 1
       - name: bar
    limitOverlay:
      enabled: true
      delay: 1h
    nightTime:
      enabled: true
      time: "23:30"
`)

	f, err := ioutil.TempFile("", "tmp")
	if err != nil {
		panic(err)
	}

	defer func(name string) {
		_ = os.Remove(name)
	}(f.Name())
	_, _ = f.Write(testRules)
	_ = f.Close()

	cfg, err := configuration.LoadConfigurationFile(f.Name())

	if assert.Nil(t, err) {
		assert.True(t, cfg.Debug)

		assert.True(t, cfg.Exporter.Enabled)
		assert.Equal(t, 5*time.Minute, cfg.Interval)
		assert.Equal(t, 8080, cfg.Exporter.Port)

		assert.True(t, cfg.Controller.Enabled)
		assert.True(t, cfg.Controller.TadoBot.Enabled)
		assert.Equal(t, "1234", cfg.Controller.TadoBot.Token.Value)

		if assert.NotNil(t, cfg.Controller.ZoneConfig) && assert.Len(t, cfg.Controller.ZoneConfig, 1) {
			assert.Equal(t, 1, cfg.Controller.ZoneConfig[0].ZoneID)
			assert.Equal(t, "", cfg.Controller.ZoneConfig[0].ZoneName)

			assert.True(t, cfg.Controller.ZoneConfig[0].AutoAway.Enabled)
			if assert.Len(t, cfg.Controller.ZoneConfig[0].AutoAway.Users, 2) {
				assert.Equal(t, 1, cfg.Controller.ZoneConfig[0].AutoAway.Users[0].MobileDeviceID)
				assert.Equal(t, "", cfg.Controller.ZoneConfig[0].AutoAway.Users[0].MobileDeviceName)

				assert.Equal(t, 0, cfg.Controller.ZoneConfig[0].AutoAway.Users[1].MobileDeviceID)
				assert.Equal(t, "bar", cfg.Controller.ZoneConfig[0].AutoAway.Users[1].MobileDeviceName)
			}

			assert.True(t, cfg.Controller.ZoneConfig[0].LimitOverlay.Enabled)
			assert.Equal(t, 1*time.Hour, cfg.Controller.ZoneConfig[0].LimitOverlay.Delay)

			assert.True(t, cfg.Controller.ZoneConfig[0].NightTime.Enabled)
			assert.Equal(t, 23, cfg.Controller.ZoneConfig[0].NightTime.Time.Hour)
			assert.Equal(t, 30, cfg.Controller.ZoneConfig[0].NightTime.Time.Minutes)
			assert.Equal(t, 0, cfg.Controller.ZoneConfig[0].NightTime.Time.Seconds)
		}
	}
}

func TestConfigLoader_Defaults(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(``))

	assert.Nil(t, err)

	assert.False(t, cfg.Debug)
	assert.Equal(t, 1*time.Minute, cfg.Interval)
	assert.True(t, cfg.Exporter.Enabled)
	assert.Equal(t, 8080, cfg.Port)
	assert.False(t, cfg.Controller.Enabled)
	assert.False(t, cfg.Controller.TadoBot.Enabled)
	assert.Nil(t, cfg.Controller.ZoneConfig)
}

func TestConfigLoader_Port_Backward_Compatible(t *testing.T) {
	var testConfig = []byte(`
debug: true
exporter:
  enabled: true
  port: 8765
`)

	var (
		err error
		cfg *configuration.Configuration
	)

	cfg, err = configuration.LoadConfiguration(testConfig)
	require.NoError(t, err)
	assert.Equal(t, 8765, cfg.Port)
}

func TestConfigLoader_Tadobot(t *testing.T) {
	var testConfig = []byte(`
debug: true
controller:
  enabled: true
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
