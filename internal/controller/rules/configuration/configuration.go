package configuration

import (
	"gopkg.in/yaml.v3"
	"io"
	"time"
)

type Configuration struct {
	Home  HomeConfiguration
	Zones []ZoneConfiguration
}

func Load(r io.Reader) (Configuration, error) {
	var c Configuration

	err := yaml.NewDecoder(r).Decode(&c)
	return c, err
}

type HomeConfiguration struct {
	AutoAway AutoAwayConfiguration `yaml:"autoAway"`
}

type ZoneConfiguration struct {
	Name  string                `yaml:"name"`
	Rules ZoneRuleConfiguration `yaml:"rules"`
}

type ZoneRuleConfiguration struct {
	AutoAway     AutoAwayConfiguration     `yaml:"autoAway"`
	LimitOverlay LimitOverlayConfiguration `yaml:"limitOverlay"`
	NightTime    NightTimeConfiguration    `yaml:"nightTime"`
}

type AutoAwayConfiguration struct {
	Users []string      `yaml:"users"`
	Delay time.Duration `yaml:"delay"`
}

func (c AutoAwayConfiguration) IsActive() bool {
	return len(c.Users) > 0
}

type LimitOverlayConfiguration struct {
	Delay time.Duration `yaml:"delay"`
}

func (c LimitOverlayConfiguration) IsActive() bool {
	return c.Delay > 0
}

type NightTimeConfiguration struct {
	Timestamp Timestamp `yaml:"timestamp"`
}

func (c NightTimeConfiguration) IsActive() bool {
	return c.Timestamp.Active
}
