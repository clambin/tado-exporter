package configuration

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

// Configuration structure for tado-monitor
type Configuration struct {
	Debug      bool
	Port       int
	Interval   time.Duration
	Exporter   ExporterConfiguration
	Controller ControllerConfiguration
}

// ExporterConfiguration structure for exporter
type ExporterConfiguration struct {
	Enabled bool
}

// ControllerConfiguration structure for controller
type ControllerConfiguration struct {
	Enabled    bool
	TadoBot    TadoBotConfiguration `yaml:"tadoBot"`
	ZoneConfig []ZoneConfig         `yaml:"zones"`
}

// TadoBotConfiguration structure for TadoBot
type TadoBotConfiguration struct {
	Enabled bool `yaml:"enabled"`
	Token   struct {
		Value  string `yaml:"value"`
		EnvVar string `yaml:"envVar"`
	} `yaml:"token"`
}

// ZoneConfig contains the rules for a zone
type ZoneConfig struct {
	ZoneID       int              `yaml:"id"`
	ZoneName     string           `yaml:"name"`
	AutoAway     ZoneAutoAway     `yaml:"autoAway"`
	LimitOverlay ZoneLimitOverlay `yaml:"limitOverlay"`
	NightTime    ZoneNightTime    `yaml:"nightTime"`
}

// ZoneUser contains a user linked to a zone
type ZoneUser struct {
	MobileDeviceID   int    `yaml:"id"`
	MobileDeviceName string `yaml:"name"`
}

// ZoneAutoAway configures when to switch a zone off if all linked users are away from home
type ZoneAutoAway struct {
	Enabled bool          `yaml:"enabled"`
	Delay   time.Duration `yaml:"delay"`
	Users   []ZoneUser    `yaml:"users"`
}

// ZoneLimitOverlay configures how long a zone will be allowed in manual control
type ZoneLimitOverlay struct {
	Enabled bool          `yaml:"enabled"`
	Delay   time.Duration `yaml:"delay"`
}

// ZoneNightTime configures a timestamp when the zone will be set back to automatic
type ZoneNightTime struct {
	Enabled bool                   `yaml:"enabled"`
	Time    ZoneNightTimeTimestamp `yaml:"time"`
}

type ZoneNightTimeTimestamp struct {
	Hour    int
	Minutes int
	Seconds int
}

func (ts *ZoneNightTimeTimestamp) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var buf string
	if err = unmarshal(&buf); err == nil {
		ts.Hour, ts.Minutes, ts.Seconds, err = parseTimestamp(buf)
	}
	return
}

func parseTimestamp(buf string) (hour, minute, second int, err error) {
	var timestamp time.Time
	timestamp, err = time.Parse("15:04:05", buf)
	if err != nil {
		timestamp, err = time.Parse("15:04", buf)
	}
	if err == nil {
		hour = timestamp.Hour()
		minute = timestamp.Minute()
		second = timestamp.Second()
	}
	return
}

// LoadConfigurationFile loads the tado-monitor configuration file from file
func LoadConfigurationFile(fileName string) (cfg *Configuration, err error) {
	var content []byte
	if content, err = os.ReadFile(fileName); err != nil {
		return
	}
	return LoadConfiguration(content)
}

// LoadConfiguration loads the tado-monitor configuration file from memory
func LoadConfiguration(content []byte) (cfg *Configuration, err error) {
	cfg = &Configuration{
		Port:     8080,
		Interval: 1 * time.Minute,
		Exporter: ExporterConfiguration{
			Enabled: true,
		},
	}
	err = yaml.Unmarshal(content, &cfg)

	if err != nil {
		return nil, fmt.Errorf("invalid ocnfig file: %w", err)
	}

	if cfg.Controller.TadoBot.Token.EnvVar != "" {
		cfg.Controller.TadoBot.Token.Value = os.Getenv(cfg.Controller.TadoBot.Token.EnvVar)
	}

	return
}
