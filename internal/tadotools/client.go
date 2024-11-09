package tadotools

import (
	"context"
	"github.com/clambin/go-common/http/metrics"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"strings"
)

func GetInstrumentedTadoClient(ctx context.Context, username string, password string, metrics metrics.RequestMetrics) (*tado.ClientWithResponses, error) {
	tadoHttpClient, err := tado.NewOAuth2Client(ctx, username, password)
	if err != nil {
		return nil, err
	}
	origTP := tadoHttpClient.Transport
	tadoHttpClient.Transport = roundtripper.New(
		roundtripper.WithRequestMetrics(metrics),
		roundtripper.WithRoundTripper(origTP),
	)
	return tado.NewClientWithResponses(tado.ServerURL, tado.WithHTTPClient(tadoHttpClient))
}

func getInstrumentedRoundTripper(rt http.RoundTripper, metrics metrics.RequestMetrics) http.RoundTripper {
	return roundtripper.New(
		roundtripper.WithRequestMetrics(metrics),
		roundtripper.WithRoundTripper(rt),
	)
}

func NewTadoCallMetrics(namespace, subsystem string, labels prometheus.Labels) metrics.RequestMetrics {
	return metrics.NewRequestMetrics(metrics.Options{
		Namespace:   namespace,
		Subsystem:   subsystem,
		ConstLabels: labels,
		LabelValues: func(request *http.Request, i int) (string, string, string) {
			const homePath = "/api/v2/homes"
			path := request.URL.Path
			if strings.HasPrefix(path, homePath) {
				path = homePath
			}
			return request.Method, path, strconv.Itoa(i)
		},
	})
}
