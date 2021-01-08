package exporter

import (
	"github.com/stretchr/testify/assert"
	"tado-exporter/pkg/tado"
	"testing"

	"github.com/clambin/gotools/metrics"
)

func TestWeatherInfoState(t *testing.T) {
	cfg := Configuration{}
	probe := CreateProbe(&cfg)

	weatherInfo := tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 25.5},
		SolarIntensity:     tado.Percentage{Percentage: 50.0},
		WeatherState:       tado.Value{Value: "SUNNY"},
	}
	probe.reportWeather(&weatherInfo)

	value, err := metrics.LoadValue("tado_weather", "SUNNY")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)

	weatherInfo.WeatherState.Value = "RAINING"
	probe.reportWeather(&weatherInfo)

	value, err = metrics.LoadValue("tado_weather", "RAINING")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)
	value, err = metrics.LoadValue("tado_weather", "SUNNY")
	assert.Nil(t, err)
	assert.Equal(t, 0.0, value)
}
