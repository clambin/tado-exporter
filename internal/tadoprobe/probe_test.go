package tadoprobe_test

import (
	"github.com/clambin/gotools/httpstub"
	"github.com/clambin/gotools/metrics"
	"github.com/stretchr/testify/assert"
	"tado-exporter/internal/tadoprobe"
	"tado-exporter/internal/testtools"
	"testing"
)

func TestTadoProbe_Run(t *testing.T) {
	probe := tadoprobe.TadoProbe{
		APIClient: tadoprobe.APIClient{
			HTTPClient: httpstub.NewTestClient(testtools.APIServer),
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

	for _, testCase := range testtools.TestCases {
		value, err = metrics.LoadValue(testCase.Metric, "Living room")
		assert.Nil(t, err)
		assert.Equal(t, testCase.Value, value, testCase.Metric)
	}
}
