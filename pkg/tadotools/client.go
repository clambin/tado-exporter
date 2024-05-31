package tadotools

import (
	"fmt"
	"github.com/clambin/go-common/http/metrics"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/tado"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"strings"
)

func GetInstrumentedTadoClient(username, password, secret string, metrics metrics.RequestMetrics) (*tado.APIClient, error) {
	c, err := tado.New(username, password, secret)
	if err != nil {
		return nil, fmt.Errorf("tado: %w", err)
	}

	c.HTTPClient = &http.Client{Transport: getInstrumentedRoundTripper(c.HTTPClient.Transport, metrics)}

	return c, nil
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
