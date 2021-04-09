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
	ActivityDataPoints ZoneInfoActivityDataPoints `json:"activityDataPoints"`
	SensorDataPoints   ZoneInfoSensorDataPoints   `json:"sensorDataPoints"`
	OpenWindow         ZoneInfoOpenWindow         `json:"openwindow,omitempty"`
	Overlay            ZoneInfoOverlay            `json:"overlay,omitempty"`
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

// ZoneInfoOverlay contains the zone's manual settings
type ZoneInfoOverlay struct {
	Type        string                     `json:"type"`
	Setting     ZoneInfoOverlaySetting     `json:"setting"`
	Termination ZoneInfoOverlayTermination `json:"termination"`
}

// ZoneInfoOverlaySetting contains the zone's overlay settings
type ZoneInfoOverlaySetting struct {
	Type        string      `json:"type"`
	Power       string      `json:"power"`
	Temperature Temperature `json:"temperature"`
}

// ZoneInfoOverlayTermination contains the termination parameters for the zone's overlay
type ZoneInfoOverlayTermination struct {
	Type          string `json:"type"`
	RemainingTime int    `json:"remainingTimeInSeconds,omitempty"`
	// not specified:
	//  "typeSkillBasedApp":"NEXT_TIME_BLOCK",
	//  "durationInSeconds":1800,
	//  "expiry":"2021-02-05T23:00:00Z",
	//  "projectedExpiry":"2021-02-05T23:00:00Z"
}

// GetZoneInfo gets the info for the specified ZoneID
func (client *APIClient) GetZoneInfo(zoneID int) (ZoneInfo, error) {
	var (
		err          error
		body         []byte
		tadoZoneInfo ZoneInfo
	)
	if err = client.initialize(); err == nil {
		if body, err = client.call("GET", client.apiURL("/zones/"+strconv.Itoa(zoneID)+"/state"), ""); err == nil {
			err = json.Unmarshal(body, &tadoZoneInfo)
		}
	}
	return tadoZoneInfo, err
}

// SetZoneOverlay sets an overlay (manual temperature setting) for the specified ZoneID
func (client *APIClient) SetZoneOverlay(zoneID int, temperature float64) error {
	if temperature < 5 {
		temperature = 5
	}

	var (
		err     error
		payload []byte
	)

	if err = client.initialize(); err == nil {
		request := ZoneInfoOverlay{
			Type: "MANUAL",
			Setting: ZoneInfoOverlaySetting{
				Type:  "HEATING",
				Power: "ON",
				Temperature: Temperature{
					Celsius: temperature,
				},
			},
			Termination: ZoneInfoOverlayTermination{
				Type: "MANUAL",
			},
		}

		payload, err = json.Marshal(request)

		_, err = client.call("PUT", client.apiURL("/zones/"+strconv.Itoa(zoneID)+"/overlay"), string(payload))
	}

	return err
}

// DeleteZoneOverlay deletes the overlay (manual temperature setting) for the specified ZoneID
func (client *APIClient) DeleteZoneOverlay(zoneID int) error {
	var err error
	if err = client.initialize(); err == nil {
		_, err = client.call("DELETE", client.apiURL("/zones/"+strconv.Itoa(zoneID)+"/overlay"), "")
	}

	return err
}

// String serializes a ZoneInfo into a string. Used for logging.
func (zoneInfo *ZoneInfo) String() string {
	return fmt.Sprintf("target=%.1fºC, temp=%.1fºC, humidity=%.1f%%, heating=%.1f%%, power=%s, openwindow=%ds, overlay={%s}",
		zoneInfo.Setting.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Humidity.Percentage,
		zoneInfo.ActivityDataPoints.HeatingPower.Percentage,
		zoneInfo.Setting.Power,
		zoneInfo.OpenWindow.DurationInSeconds-zoneInfo.OpenWindow.RemainingTimeInSeconds,
		zoneInfo.Overlay.String(),
	)
}

// String serializes a ZoneInfoOverlay into a string. Used for logging.
func (overlay *ZoneInfoOverlay) String() string {
	return fmt.Sprintf(`type=%s, settings={%s}, termination={type="%s", remaining=%d}`,
		overlay.Type,
		overlay.Setting.String(),
		overlay.Termination.Type,
		overlay.Termination.RemainingTime,
	)
}

// String serializes a ZoneInfoOverlaySetting into a string. Used for logging.
func (setting *ZoneInfoOverlaySetting) String() string {
	return fmt.Sprintf("type=%s, power=%s, temp=%.1fºC",
		setting.Type,
		setting.Power,
		setting.Temperature.Celsius,
	)
}
