package monitor

import (
	"context"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	requestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tado",
		Subsystem: "monitor",
		Name:      "http_requests_total",
		Help:      "total number of http requests",
	},
		[]string{"code", "method"},
	)

	requestDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "tado",
		Subsystem: "monitor",
		Name:      "http_request_duration_seconds",
		Help:      "duration of http requests",
	},
		[]string{"code", "method"},
	)
)

func instrumentedTadoClient(ctx context.Context, username string, password string, counter *prometheus.CounterVec, obs prometheus.ObserverVec) (*tado.ClientWithResponses, error) {
	tadoHttpClient, err := tado.NewOAuth2Client(ctx, username, password)
	if err != nil {
		return nil, err
	}

	rt := promhttp.InstrumentRoundTripperCounter(counter,
		promhttp.InstrumentRoundTripperDuration(obs,
			tadoHttpClient.Transport,
		),
	)
	return tado.NewClientWithResponses(tado.ServerURL, tado.WithHTTPClient(&http.Client{Transport: rt}))
}
