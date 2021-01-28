package configuration

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

// Configuration structure for tado-monitor
type Configuration struct {
	Debug      bool
	Exporter   ExporterConfiguration
	Controller ControllerConfiguration
}

// ExporterConfiguration structure for exporter
type ExporterConfiguration struct {
	Enabled  bool
	Interval time.Duration
	Port     int
}

// ControllerConfiguration structure for controller
type ControllerConfiguration struct {
	Enabled           bool
	Interval          time.Duration
	NotifyURL         string               `yaml:"notifyURL"`
	AutoAwayRules     *[]*AutoAwayRule     `yaml:"autoAwayRules"`
	OverlayLimitRules *[]*OverlayLimitRule `yaml:"overlayLimitRules"`
}

// OverlayLimitRule removes an overlay from ZoneID/ZoneName after it's been active for MaxTime
type OverlayLimitRule struct {
	ZoneID   int           `yaml:"zoneID"`
	ZoneName string        `yaml:"zoneName"`
	MaxTime  time.Duration `yaml:"maxTime"`
}

// AutoAwayRule sets a zone (ZoneID/ZoneName) to TargetTemperature when the user (MobileDeviceID/MobileDeviceName)
// has been away for WaitTime
type AutoAwayRule struct {
	MobileDeviceID    int           `yaml:"mobileDeviceID"`
	MobileDeviceName  string        `yaml:"mobileDeviceName"`
	WaitTime          time.Duration `yaml:"waitTime"`
	ZoneID            int           `yaml:"zoneID"`
	ZoneName          string        `yaml:"zoneName"`
	TargetTemperature float64       `yaml:"targetTemperature"`
}

// LoadConfigurationFile loads the tado-monitor configuration file from file
func LoadConfigurationFile(fileName string) (*Configuration, error) {
	var (
		err           error
		content       []byte
		configuration *Configuration
	)
	if content, err = ioutil.ReadFile(fileName); err == nil {
		configuration, err = LoadConfiguration(content)
	}
	return configuration, err
}

// LoadConfiguration loads the tado-monitor configuration file from memory
func LoadConfiguration(content []byte) (*Configuration, error) {
	configuration := Configuration{
		Exporter: ExporterConfiguration{
			Enabled:  true,
			Interval: 1 * time.Minute,
			Port:     8080,
		},
		Controller: ControllerConfiguration{
			Interval: 5 * time.Minute,
		},
	}
	err := yaml.Unmarshal(content, &configuration)

	log.WithField("err", err).Debug("LoadConfiguration")

	return &configuration, err
}
