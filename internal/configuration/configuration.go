package configuration

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
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
	Enabled    bool
	Interval   time.Duration
	TadoBot    TadoBotConfiguration `yaml:"tadoBot"`
	ZoneConfig *[]ZoneConfig        `yaml:"zones"`
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
	Users        []ZoneUser       `yaml:"users"`
	LimitOverlay ZoneLimitOverlay `yaml:"limitOverlay"`
	NightTime    ZoneNightTime    `yaml:"nightTime"`
}

// ZoneUser contains a user linked to a zone
type ZoneUser struct {
	MobileDeviceID   int    `yaml:"id"`
	MobileDeviceName string `yaml:"name"`
}

// ZoneLimitOverlay configures how long a zone will be allowed in manual control
type ZoneLimitOverlay struct {
	Enabled bool          `yaml:"enabled"`
	Limit   time.Duration `yaml:"limit"`
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

	if err == nil {
		if configuration.Controller.TadoBot.Enabled && configuration.Controller.TadoBot.Token.EnvVar != "" {
			configuration.Controller.TadoBot.Token.Value = os.Getenv(configuration.Controller.TadoBot.Token.EnvVar)

			if configuration.Controller.TadoBot.Token.Value == "" {
				log.WithField("envVar", configuration.Controller.TadoBot.Token.EnvVar).
					Warning("tadoBot environment variable for token not set. Disabling tadoBot")
				configuration.Controller.TadoBot.Enabled = false
			}
		}
	}

	log.WithField("err", err).Debug("LoadConfiguration")

	return &configuration, err
}
