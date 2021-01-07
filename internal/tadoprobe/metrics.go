package tadoprobe

import (
	"github.com/clambin/gotools/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	tadoZoneTargetTempCelsius = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_target_temp_celsius",
		Help: "Target temperature of this zone",
	}, []string{"zone_name"})

	tadoZonePowerState = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_power_state",
		Help: "Power status of this zone",
	}, []string{"zone_name"})

	tadoTemperatureCelsius = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_temperature_celsius",
		Help: "Current temperature",
	}, []string{"zone_name"})

	tadoHeatingPercentage = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_heating_percentage",
		Help: "Current heating percentage",
	}, []string{"zone_name"})

	tadoHumidityPercentage = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_humidity_percentage",
		Help: "Current humidity percentage",
	}, []string{"zone_name"})
)
