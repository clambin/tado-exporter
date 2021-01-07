package tadoprobe_test

import (
	"github.com/clambin/gotools/httpstub"
	"github.com/clambin/gotools/metrics"
	"github.com/stretchr/testify/assert"
	"tado-exporter/internal/tadoprobe"
	"testing"
)

func TestTadoProbe_Run(t *testing.T) {
	probe := tadoprobe.TadoProbe{
		APIClient: tadoprobe.APIClient{
			HTTPClient: httpstub.NewTestClient(apiServer),
			Username:   "user@examle.com",
			Password:   "some-password",
		},
	}

	var (
		err   error
		value float64
	)

	err = probe.Run()
	// we didn't implement responses for zones 2 & 3, so Run will fail
	assert.NotNil(t, err)

	testCases := []struct {
		metric string
		value  float64
	}{
		{"tado_zone_target_temp_celsius", 20.0},
		{"tado_zone_power_state", 100.0},
		{"tado_temperature_celsius", 19.94},
		{"tado_heating_percentage", 11.0},
		{"tado_humidity_percentage", 37.7},
	}
	for _, testCase := range testCases {
		value, err = metrics.LoadValue(testCase.metric, "Living room")
		assert.Nil(t, err)
		assert.Equal(t, testCase.value, value, testCase.metric)
	}
}
