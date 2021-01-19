package tado

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ZoneInfo contains the response to /api/v2/homes/<HomeID>/zones/<zoneID>/state
//
// This structure provides the following key information:
//   Setting.Power:                              power state of the specified zone (0-1)
//   Temperature.Celsius:                        target temperature for the zone, in degrees Celsius
//   OpenWindow.DurationInSeconds:               how long an open window has been detected in seconds
//   ActivityDataPoints.HeatingPower.Percentage: heating power for the zone (0-100%)
//   SensorDataPoints.Temperature.Celsius:       current temperature, in degrees Celsius
//   SensorDataPoints.Humidity.Percentage:       humidity (0-100%)
type ZoneInfo struct {
	Setting            ZoneInfoSetting            `json:"setting"`
	OpenWindow         ZoneInfoOpenWindow         `json:"openwindow,omitempty"`
	ActivityDataPoints ZoneInfoActivityDataPoints `json:"activityDataPoints"`
	SensorDataPoints   ZoneInfoSensorDataPoints   `json:"sensorDataPoints"`
}

// ZoneInfoSetting contains the zone's current power & target temperature
type ZoneInfoSetting struct {
	Power       string      `json:"power"`
	Temperature Temperature `json:"temperature"`
}

// ZoneInfoActivityDataPoints contains the zone's heating info
type ZoneInfoActivityDataPoints struct {
	HeatingPower Percentage `json:"heatingPower"`
}

// ZoneInfoSensorDataPoints contains the zone's current temperature & humidity
type ZoneInfoSensorDataPoints struct {
	Temperature Temperature `json:"insideTemperature"`
	Humidity    Percentage  `json:"humidity"`
}

// ZoneInfoOpenWindow contains info on an open window. Only set if a window is open
type ZoneInfoOpenWindow struct {
	DetectedTime           time.Time `json:"detectedTime"`
	DurationInSeconds      int       `json:"durationInSeconds"`
	Expiry                 time.Time `json:"expiry"`
	RemainingTimeInSeconds int       `json:"remainingTimeInSeconds"`
}

// GetZoneInfo gets the info for the specified Zone
func (client *APIClient) GetZoneInfo(zoneID int) (*ZoneInfo, error) {
	var (
		err          error
		body         []byte
		tadoZoneInfo ZoneInfo
	)
	if err = client.initialize(); err == nil {
		if body, err = client.call(client.apiURL("/zones/" + strconv.Itoa(zoneID) + "/state")); err == nil {
			err = json.Unmarshal(body, &tadoZoneInfo)
		}
	}
	return &tadoZoneInfo, err
}

// String serializes a ZoneInfo into a string. Used for logging.
func (zoneInfo *ZoneInfo) String() string {
	return fmt.Sprintf("target=%.1fºC, temp=%.1fºC, humidity=%.1f%%, heating=%.1f%%, power=%s, openwindow=%ds",
		zoneInfo.Setting.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Humidity.Percentage,
		zoneInfo.ActivityDataPoints.HeatingPower.Percentage,
		zoneInfo.Setting.Power,
		zoneInfo.OpenWindow.DurationInSeconds-zoneInfo.OpenWindow.RemainingTimeInSeconds,
	)
}
