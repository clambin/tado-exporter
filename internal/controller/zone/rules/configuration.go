package rules

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log/slog"
	"strings"
	"time"
)

type ZoneConfig struct {
	Zone  string     `yaml:"zone"`
	Rules []ZoneRule `yaml:"rules"`
}

func Load(in io.Reader, l *slog.Logger) ([]ZoneConfig, error) {
	var config struct {
		Zones []ZoneConfig `yaml:"zones"`
	}

	if err := yaml.NewDecoder(in).Decode(&config); err != nil {
		return nil, err
	}
	for _, zone := range config.Zones {
		var kinds []string
		for _, rule := range zone.Rules {
			kinds = append(kinds, rule.Kind.String())
		}
		l.Info("zone rules found",
			slog.String("zone", zone.Zone),
			slog.String("rules", strings.Join(kinds, ",")),
		)
	}
	return config.Zones, nil
}

type ZoneRule struct {
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
