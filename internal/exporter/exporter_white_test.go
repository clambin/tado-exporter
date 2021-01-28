package exporter

import (
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWeatherInfoState(t *testing.T) {
	exporter := Exporter{}

	weatherInfo := tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 25.5},
		SolarIntensity:     tado.Percentage{Percentage: 50.0},
		WeatherState:       tado.Value{Value: "SUNNY"},
	}
	exporter.reportWeather(&weatherInfo)

	value, err := metrics.LoadValue("tado_weather", "SUNNY")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)

	weatherInfo.WeatherState.Value = "RAINING"
	exporter.reportWeather(&weatherInfo)

	value, err = metrics.LoadValue("tado_weather", "RAINING")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)
	value, err = metrics.LoadValue("tado_weather", "SUNNY")
	assert.Nil(t, err)
	assert.Equal(t, 0.0, value)
}
