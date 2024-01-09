package configuration

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"time"
)

type Timestamp struct {
	Hour    int
	Minutes int
	Seconds int
	Active  bool
}

func (t *Timestamp) UnmarshalYAML(value *yaml.Node) error {
	timestamp, err := time.Parse("15:04:05", value.Value)
	if err != nil {
		timestamp, err = time.Parse("15:04", value.Value)
	}
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	*t = Timestamp{
		Hour:    timestamp.Hour(),
		Minutes: timestamp.Minute(),
		Seconds: timestamp.Second(),
		Active:  true,
	}
	return nil
}

func (t Timestamp) MarshalYAML() (interface{}, error) {
	ts := time.Date(0, 0, 0, t.Hour, t.Minutes, t.Seconds, 0, time.UTC)
	return ts.Format("15:04:05"), nil
}
