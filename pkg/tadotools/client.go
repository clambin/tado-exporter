package tadotools

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/http/metrics"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/tado"
	"net/http"
	"strings"
	"time"
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

var _ metrics.RequestMetrics = &tadoCallMetrics{}

type tadoCallMetrics struct {
	metrics.RequestMetrics
}

func NewTadoCallMetrics(namespace, subsystem string, labels map[string]string) metrics.RequestMetrics {
	return &tadoCallMetrics{
		RequestMetrics: metrics.NewRequestSummaryMetrics(namespace, subsystem, labels),
	}
}

func (t *tadoCallMetrics) Measure(req *http.Request, statusCode int, duration time.Duration) {
	req2 := req.Clone(context.Background())
	req2.URL.Path = filterPath(req.URL.Path)
	t.RequestMetrics.Measure(req2, statusCode, duration)
}

func filterPath(path string) string {
	const homePath = "/api/v2/homes"
	if strings.HasPrefix(path, homePath) {
		return homePath
	}
	return path
}
