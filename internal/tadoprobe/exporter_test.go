package tadoprobe_test

import (
	"github.com/clambin/gotools/httpstub"
	"github.com/clambin/gotools/metrics"
	"github.com/stretchr/testify/assert"
	"tado-exporter/internal/tadoprobe"
	"tado-exporter/internal/testtools"
	"testing"
	"time"
)

func TestRunProbe(t *testing.T) {
	cfg := tadoprobe.Configuration{}
	probe := tadoprobe.CreateProbe(&cfg)
	assert.NotNil(t, probe)
	assert.NotNil(t, probe.HTTPClient)

	probe.APIClient.HTTPClient = httpstub.NewTestClient(testtools.APIServer)
	tadoprobe.RunProbe(probe, 5*time.Second)

	testCases := testtools.TestCases
	for _, testCase := range testCases {
		value, err := metrics.LoadValue(testCase.Metric, "Living room")
		assert.Nil(t, err)
		assert.Equal(t, testCase.Value, value, testCase.Metric)
	}

	value, err := metrics.LoadValue("tado_weather", "CLOUDY_MOSTLY")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)
}
