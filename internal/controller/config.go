package controller

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

type Rules struct {
	OverlayLimit *[]OverlayLimit `yaml:"overlayLimit"`
	AutoAway     *[]AutoAway     `yaml:"autoAway"`
}

type OverlayLimit struct {
	ZoneID   int           `yaml:"zoneID"`
	ZoneName string        `yaml:"zoneName"`
	MaxTime  time.Duration `yaml:"maxTime"`
}

type AutoAway struct {
	MobileDeviceID    int           `yaml:"mobileDeviceID"`
	MobileDeviceName  string        `yaml:"mobileDeviceName"`
	WaitTime          time.Duration `yaml:"waitTime"`
	ZoneID            int           `yaml:"zoneID"`
	ZoneName          string        `yaml:"zoneName"`
	TargetTemperature float64       `yaml:"targetTemperature"`
}

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

func ParseRules(content []byte) (*Rules, error) {
	var err error

	rules := Rules{}
	err = yaml.Unmarshal(content, &rules)

	log.WithField("err", err).Debug("ParseConfig")

	return &rules, err
}
