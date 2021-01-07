package tadoprobe

import "fmt"

type TadoZone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TadoZoneInfo struct {
	Setting struct {
		Power       string `json:"power"`
		Temperature struct {
			Celsius float64 `json:"celsius"`
		} `json:"temperature"`
	} `json:"setting"`
	OpenWindow         string `json:"openWindow"`
	ActivityDataPoints struct {
		HeatingPower struct {
			Percentage float64 `json:"percentage"`
		} `json:"heatingPower"`
	} `json:"activityDataPoints"`
	SensorDataPoints struct {
		Temperature struct {
			Celsius float64 `json:"celsius"`
		} `json:"insideTemperature"`
		Humidity struct {
			Percentage float64 `json:"percentage"`
		} `json:"humidity"`
	} `json:"sensorDataPoints"`
}

func (zoneInfo *TadoZoneInfo) String() string {
	return fmt.Sprintf("target=%.1f power=%s temp=%.1f, humidity=%.1f, heating=%.1f",
		zoneInfo.Setting.Temperature.Celsius,
		zoneInfo.Setting.Power,
		zoneInfo.SensorDataPoints.Temperature.Celsius,
		zoneInfo.SensorDataPoints.Humidity.Percentage,
		zoneInfo.ActivityDataPoints.HeatingPower.Percentage,
	)
}

type TadoWeatherInfo struct {
	OutsideTemperature struct {
		Celsius float64 `json:"celsius"`
	} `json:"outsideTemperature"`
	SolarIntensity struct {
		Percentage float64 `json:"percentage"`
	} `json:"solarIntensity"`
	WeatherState struct {
		Value string `json:"value"`
	} `json:"weatherState"`
}

func (weatherInfo *TadoWeatherInfo) String() string {
	return fmt.Sprintf("temp=%.1f solar=%.1f weather=%s",
		weatherInfo.OutsideTemperature.Celsius,
		weatherInfo.SolarIntensity.Percentage,
		weatherInfo.WeatherState.Value,
	)
}
