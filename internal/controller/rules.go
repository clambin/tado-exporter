package controller

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

// Rules for tado-controller. Currently supports OverlayLimit and AutoAway
type Rules struct {
	OverlayLimit *[]OverlayLimit `yaml:"overlayLimit"`
	AutoAway     *[]AutoAway     `yaml:"autoAway"`
}

// OverlayLimit rule removes an overlay from ZoneID/ZoneName for MaxTime duration
type OverlayLimit struct {
	ZoneID   int           `yaml:"zoneID"`
	ZoneName string        `yaml:"zoneName"`
	MaxTime  time.Duration `yaml:"maxTime"`
}

// AutoAway sets a zone (ZoneID/ZoneName) to TargetTemperature when the user (MobileDeviceID/MobileDeviceName)
// has been away for a WaitTime duration
type AutoAway struct {
	MobileDeviceID    int           `yaml:"mobileDeviceID"`
	MobileDeviceName  string        `yaml:"mobileDeviceName"`
	WaitTime          time.Duration `yaml:"waitTime"`
	ZoneID            int           `yaml:"zoneID"`
	ZoneName          string        `yaml:"zoneName"`
	TargetTemperature float64       `yaml:"targetTemperature"`
}

// ParseRulesFile returns the list of tado-controller rules in fileName
func ParseRulesFile(fileName string) (*Rules, error) {
	var (
		err     error
		content []byte
		rules   *Rules
	)
	if content, err = ioutil.ReadFile(fileName); err == nil {
		rules, err = ParseRules(content)
	}
	return rules, err
}

// ParseRules returns a list of tado-controller rules in content
func ParseRules(content []byte) (*Rules, error) {
	var err error

	rules := Rules{}
	err = yaml.Unmarshal(content, &rules)

	log.WithField("err", err).Debug("ParseConfig")

	return &rules, err
}
