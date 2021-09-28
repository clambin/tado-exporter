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
	Port       int
	Interval   time.Duration
	Exporter   ExporterConfiguration
	Controller ControllerConfiguration
}

// ExporterConfiguration structure for exporter
type ExporterConfiguration struct {
	Enabled bool
	// Port is obsolete and will be removed in a future version
	Port int
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
		Port:     8080,
		Interval: 1 * time.Minute,
		Exporter: ExporterConfiguration{
			Enabled: true,
		},
	}
	err := yaml.Unmarshal(content, &configuration)

	if err != nil {
		log.WithError(err).Fatal("unable to parse configuration file")
	}

	if configuration.Controller.TadoBot.Enabled && configuration.Controller.TadoBot.Token.EnvVar != "" {
		configuration.Controller.TadoBot.Token.Value = os.Getenv(configuration.Controller.TadoBot.Token.EnvVar)

		if configuration.Controller.TadoBot.Token.Value == "" {
			log.WithField("envVar", configuration.Controller.TadoBot.Token.EnvVar).
				Warning("tadoBot environment variable for token not set. Disabling tadoBot")
			configuration.Controller.TadoBot.Enabled = false
		}

		if configuration.Exporter.Port > 0 {
			log.Warning("configuration: Exporter.Port is obsolete. move to root level")
			configuration.Port = configuration.Exporter.Port
		}
	}

	if configuration.Exporter.Port > 0 {
		log.Warning("configuration: Exporter.Port is obsolete. move to top level")
		configuration.Port = configuration.Exporter.Port
	}

	return &configuration, err
}
