package tado

import "fmt"

// Zone contains the response to /api/v2/homes/<HomeID>/zones
type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ZoneInfo contains the response to /api/v2/homes/<HomeID>/zones/<zoneID>/state
type ZoneInfo struct {
	Setting            ZoneInfoSetting            `json:"setting"`
	OpenWindow         string                     `json:"openWindow"`
	ActivityDataPoints ZoneInfoActivityDataPoints `json:"activityDataPoints"`
	SensorDataPoints   ZoneInfoSensorDataPoints   `json:"sensorDataPoints"`
}

func (zoneInfo *ZoneInfo) String() string {
	return fmt.Sprintf("target=%.1f power=%s temp=%.1f, humidity=%.1f, heating=%.1f",
		zoneInfo.Setting.Temperature.Celsius,
		zoneInfo.Setting.Power,
		zoneInfo.SensorDataPoints.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Humidity.Percentage,
		zoneInfo.ActivityDataPoints.HeatingPower.Percentage,
	)
}

// WeatherInfo contains the response to /api/v2/homes/<HomeID>/weather
type WeatherInfo struct {
	OutsideTemperature Temperature `json:"outsideTemperature"`
	SolarIntensity     Percentage  `json:"solarIntensity"`
	WeatherState       Value       `json:"weatherState"`
}

func (weatherInfo *WeatherInfo) String() string {
	return fmt.Sprintf("temp=%.1f solar=%.1f weather=%s",
		weatherInfo.OutsideTemperature.Celsius,
		weatherInfo.SolarIntensity.Percentage,
		weatherInfo.WeatherState.Value,
	)
}

// Supporting data/json structs

// Temperature structure representing a temperature (in degrees Celsius)
type Temperature struct {
	Celsius float64 `json:"celsius"`
}

// Percentage structure representing a percentage
type Percentage struct {
	Percentage float64 `json:"percentage"`
}

// Value structure representing a value
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
