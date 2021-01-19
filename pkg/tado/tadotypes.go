package tado

import (
	"fmt"
	"strings"
	"time"
)

// Zone contains the response to /api/v2/homes/<HomeID>/zones
type Zone struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Devices []Device `json:"devices"`
}

func (zone Zone) String() string {
	devicesAsStr := make([]string, len(zone.Devices))
	for i, device := range zone.Devices {
		devicesAsStr[i] = device.String()
	}
	devicesStr := strings.Join(devicesAsStr, ", ")

	return fmt.Sprintf("id=%d name=%s devices={%s}",
		zone.ID,
		zone.Name,
		devicesStr,
	)
}

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

func (zoneInfo *ZoneInfo) String() string {
	return fmt.Sprintf("target=%.1fºC power=%s temp=%.1fºC, humidity=%.1f%%, heating=%.1f%%, openwindow=%ds",
		zoneInfo.Setting.Temperature.Celsius,
		zoneInfo.Setting.Power,
		zoneInfo.SensorDataPoints.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Humidity.Percentage,
		zoneInfo.ActivityDataPoints.HeatingPower.Percentage,
		zoneInfo.OpenWindow.DurationInSeconds,
	)
}

// WeatherInfo contains the response to /api/v2/homes/<HomeID>/weather
//
// This structure provides the following key information:
//   OutsideTemperature.Celsius:  outside temperate, in degrees Celsius
//   SolarIntensity.Percentage:   solar intensity (0-100%)
//   WeatherState.Value:          string describing current weather (list TBD)
type WeatherInfo struct {
	OutsideTemperature Temperature `json:"outsideTemperature"`
	SolarIntensity     Percentage  `json:"solarIntensity"`
	WeatherState       Value       `json:"weatherState"`
}

// String converts WeatherInfo to a loggable string
func (weatherInfo *WeatherInfo) String() string {
	return fmt.Sprintf("temp=%.1fºC, solar=%.1f%%, weather=%s",
		weatherInfo.OutsideTemperature.Celsius,
		weatherInfo.SolarIntensity.Percentage,
		weatherInfo.WeatherState.Value,
	)
}

// MobileDevice contains the response to /api/v2/homes/<HomeID>/mobileDevices
type MobileDevice struct {
	ID       int                  `json:"id"`
	Name     string               `json:"name"`
	Settings MobileDeviceSettings `json:"settings"`
	Location MobileDeviceLocation `json:"location"`
}

// MobileDeviceSettings is a sub-structure of MobileDevice
type MobileDeviceSettings struct {
	GeoTrackingEnabled bool `json:"geoTrackingEnabled"`
}

// MobileDeviceLocation is a sub-structure of MobileDevice
type MobileDeviceLocation struct {
	Stale  bool `json:"stale"`
	AtHome bool `json:"atHome"`
}

func (mobileDevice *MobileDevice) String() string {
	return fmt.Sprintf("name=%s, geotrack=%v, stale=%v, athome=%v",
		mobileDevice.Name,
		mobileDevice.Settings.GeoTrackingEnabled,
		mobileDevice.Location.Stale,
		mobileDevice.Location.AtHome,
	)
}

// Supporting data/json structs

// Temperature contains a temperature in degrees Celsius
type Temperature struct {
	Celsius float64 `json:"celsius"`
}

// Percentage contains a percentage (0-100%)
type Percentage struct {
	Percentage float64 `json:"percentage"`
}

// Value contains a string value
type Value struct {
	Value string `json:"value"`
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

// ConnectionState contains the connection state of a Tado device
type ConnectionState struct {
	Value     bool      `json:"value"`
	Timestamp time.Time `json:"timeStamp"`
}

// Device contains attributes of a Tado device
type Device struct {
	DeviceType      string          `json:"deviceType"`
	Firmware        string          `json:"currentFwVersion"`
	ConnectionState ConnectionState `json:"connectionState"`
	BatteryState    string          `json:"batteryState"`
}

func (device *Device) String() string {
	return fmt.Sprintf("type=%s firmware=%s connection=%v battery=%s",
		device.DeviceType,
		device.Firmware,
		device.ConnectionState.Value,
		device.BatteryState,
	)
}
