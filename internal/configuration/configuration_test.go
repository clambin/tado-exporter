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
  zones:
  - id: 1
    users:
    - id: 1
    - name: bar
    limitOverlay:
      enabled: true
      limit: 1h
    nightTime:
      enabled: true
      time: "23:30:15"
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

		if assert.NotNil(t, cfg.Controller.ZoneConfig) && assert.Len(t, *cfg.Controller.ZoneConfig, 1) {
			assert.Equal(t, 1, (*cfg.Controller.ZoneConfig)[0].ZoneID)
			assert.Equal(t, "", (*cfg.Controller.ZoneConfig)[0].ZoneName)

			if assert.Len(t, (*cfg.Controller.ZoneConfig)[0].Users, 2) {
				assert.Equal(t, 1, (*cfg.Controller.ZoneConfig)[0].Users[0].MobileDeviceID)
				assert.Equal(t, "", (*cfg.Controller.ZoneConfig)[0].Users[0].MobileDeviceName)

				assert.Equal(t, 0, (*cfg.Controller.ZoneConfig)[0].Users[1].MobileDeviceID)
				assert.Equal(t, "bar", (*cfg.Controller.ZoneConfig)[0].Users[1].MobileDeviceName)

			}

			assert.True(t, (*cfg.Controller.ZoneConfig)[0].LimitOverlay.Enabled)
			assert.Equal(t, 1*time.Hour, (*cfg.Controller.ZoneConfig)[0].LimitOverlay.Limit)

			assert.True(t, (*cfg.Controller.ZoneConfig)[0].NightTime.Enabled)
			assert.Equal(t, 23, (*cfg.Controller.ZoneConfig)[0].NightTime.Time.Hour)
			assert.Equal(t, 30, (*cfg.Controller.ZoneConfig)[0].NightTime.Time.Minutes)
			assert.Equal(t, 15, (*cfg.Controller.ZoneConfig)[0].NightTime.Time.Seconds)
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
	assert.Nil(t, cfg.Controller.ZoneConfig)
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
