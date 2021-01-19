package tado

import (
	"encoding/json"
	"fmt"
	"strconv"
)

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

// GetWeatherInfo retrieves weather information for the user's Home.
func (client *APIClient) GetWeatherInfo() (*WeatherInfo, error) {
	var (
		err             error
		tadoWeatherInfo WeatherInfo
		body            []byte
	)
	if err = client.initialize(); err == nil {
		apiURL := "https://my.tado.com/api/v2/homes/" + strconv.Itoa(client.HomeID) + "/weather"
		if body, err = client.call(apiURL); err == nil {
			err = json.Unmarshal(body, &tadoWeatherInfo)
		}
	}
	return &tadoWeatherInfo, err
}

// String converts WeatherInfo to a loggable string
func (weatherInfo *WeatherInfo) String() string {
	return fmt.Sprintf("temp=%.1fºC, solar=%.1f%%, weather=%s",
		weatherInfo.OutsideTemperature.Celsius,
		weatherInfo.SolarIntensity.Percentage,
		weatherInfo.WeatherState.Value,
	)
}
