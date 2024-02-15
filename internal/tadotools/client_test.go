package tadotools

import (
	"bytes"
	"errors"
	"github.com/clambin/go-common/httpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestGetInstrumentedTadoClient(t *testing.T) {
	finalRoundTripper := httpclient.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		//return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(&bytes.Buffer{})}, nil
		return nil, errors.New("call failed")
	})

	testCases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "root",
			path: "/",
			want: `
# HELP tado_monitor_api_errors_total Number of failed HTTP calls
# TYPE tado_monitor_api_errors_total counter
tado_monitor_api_errors_total{application="tado",method="GET",path="/"} 1
`,
		},
		{
			name: "blank",
			path: "",
			want: `
# HELP tado_monitor_api_errors_total Number of failed HTTP calls
# TYPE tado_monitor_api_errors_total counter
tado_monitor_api_errors_total{application="tado",method="GET",path=""} 1
`,
		},
		{
			name: "home",
			path: "/api/v2/homes/631798/mobileDevices",
			want: `
# HELP tado_monitor_api_errors_total Number of failed HTTP calls
# TYPE tado_monitor_api_errors_total counter
tado_monitor_api_errors_total{application="tado",method="GET",path="/api/v2/homes"} 1
`,
		},
		{
			name: "me",
			path: "/api/v2/me",
			want: `
# HELP tado_monitor_api_errors_total Number of failed HTTP calls
# TYPE tado_monitor_api_errors_total counter
tado_monitor_api_errors_total{application="tado",method="GET",path="/api/v2/me"} 1
`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := prometheus.NewPedanticRegistry()
			roundTripper := getRegisteredRoundTripper(finalRoundTripper, r)

			c := http.Client{Transport: roundTripper}
			_, err := c.Get(tt.path)
			require.Error(t, err)

			assert.NoError(t, testutil.GatherAndCompare(r, bytes.NewBufferString(tt.want), "tado_monitor_api_errors_total"))
		})
	}
}
