package tadotools

import (
	"bytes"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetInstrumentedTadoClient(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "root",
			path: "/",
			want: `
# HELP tado_monitor_http_requests_total total number of http requests
# TYPE tado_monitor_http_requests_total counter
tado_monitor_http_requests_total{application="tado",code="404",method="GET",path="/"} 1
`,
		},
		{
			name: "blank",
			path: "",
			want: `
# HELP tado_monitor_http_requests_total total number of http requests
# TYPE tado_monitor_http_requests_total counter
tado_monitor_http_requests_total{application="tado",code="404",method="GET",path="/"} 1
`,
		},
		{
			name: "home",
			path: "/api/v2/homes/631798/mobileDevices",
			want: `
# HELP tado_monitor_http_requests_total total number of http requests
# TYPE tado_monitor_http_requests_total counter
tado_monitor_http_requests_total{application="tado",code="404",method="GET",path="/api/v2/homes"} 1
`,
		},
		{
			name: "me",
			path: "/api/v2/me",
			want: `
# HELP tado_monitor_http_requests_total total number of http requests
# TYPE tado_monitor_http_requests_total counter
tado_monitor_http_requests_total{application="tado",code="404",method="GET",path="/api/v2/me"} 1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metrics := NewTadoCallMetrics("tado", "monitor", map[string]string{"application": "tado"})
			finalRoundTripper := roundtripper.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(&bytes.Buffer{})}, nil
				//return nil, errors.New("call failed")
			})

			c := http.Client{Transport: getInstrumentedRoundTripper(finalRoundTripper, metrics)}

			_, err := c.Get(tt.path)
			require.NoError(t, err)

			assert.NoError(t, testutil.CollectAndCompare(metrics, strings.NewReader(tt.want), "tado_monitor_http_requests_total"))
		})
	}
}
