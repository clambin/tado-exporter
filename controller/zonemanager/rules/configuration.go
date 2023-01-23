package rules

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"time"
)

type ZoneConfig struct {
	Zone  string       `yaml:"zone"`
	Rules []RuleConfig `yaml:"rules"`
}

type RuleConfig struct {
	Kind      Kind          `yaml:"kind"`
	Users     []string      `yaml:"users"`
	Delay     time.Duration `yaml:"delay"`
	Timestamp Timestamp     `yaml:"time"`
}

type Kind int

const (
	AutoAway Kind = iota
	LimitOverlay
	NightTime
)

func (k Kind) String() string {
	var result string
	switch k {
	case AutoAway:
		result = "autoAway"
	case LimitOverlay:
		result = "limitOverlay"
	case NightTime:
		result = "nightTime"
	}
	return result
}

func (k *Kind) UnmarshalYAML(node *yaml.Node) error {
	var err error
	switch node.Value {
	case "autoAway":
		*k = AutoAway
	case "limitOverlay":
		*k = LimitOverlay
	case "nightTime":
		*k = NightTime
	default:
		err = fmt.Errorf("invalid Kind: %s", node.Value)
	}
	return err
}

func (k Kind) MarshalYAML() (interface{}, error) {
	v := k.String()
	if v == "" {
		return "", fmt.Errorf("invalid Kind: %d", k)
	}
	return v, nil
}
