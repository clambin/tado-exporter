package tadotools

import (
	"fmt"
	"github.com/clambin/go-common/httpclient"
	"github.com/clambin/tado"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strings"
	"time"
)

func GetInstrumentedTadoClient(username, password, secret string, registry prometheus.Registerer) (*tado.APIClient, error) {
	c, err := tado.New(username, password, secret)
	if err != nil {
		return nil, fmt.Errorf("tado: %w", err)
	}

	c.HTTPClient = &http.Client{
		Transport: getRegisteredRoundTripper(c.HTTPClient.Transport, registry),
	}

	return c, nil
}

func getRegisteredRoundTripper(r http.RoundTripper, registry prometheus.Registerer) *httpclient.RoundTripper {
	rt := httpclient.NewRoundTripper(
		httpclient.WithCustomMetrics(newTadoCallMetrics("tado", "monitor", "tado")),
		httpclient.WithRoundTripper(r),
	)
	registry.MustRegister(rt)
	return rt
}

var _ httpclient.RequestMeasurer = &tadoCallMetrics{}

type tadoCallMetrics struct {
	latency *prometheus.SummaryVec
	errors  *prometheus.CounterVec
}

func newTadoCallMetrics(namespace, subsystem, application string) *tadoCallMetrics {
	return &tadoCallMetrics{
		latency: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:        prometheus.BuildFQName(namespace, subsystem, "api_latency"),
			Help:        "latency of HTTP calls",
			ConstLabels: map[string]string{"application": application},
		}, []string{"method", "path"}), // TODO: return code?
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        prometheus.BuildFQName(namespace, subsystem, "api_errors_total"),
			Help:        "Number of failed HTTP calls",
			ConstLabels: map[string]string{"application": application},
		}, []string{"method", "path"}),
	}
}

func (t tadoCallMetrics) MeasureRequest(request *http.Request, _ *http.Response, err error, duration time.Duration) {
	path := filterPath(request.URL.Path)
	t.latency.WithLabelValues(request.Method, path).Observe(duration.Seconds())
	var val float64
	if err != nil {
		val = 1
	}
	t.errors.WithLabelValues(request.Method, path).Add(val)
}

func filterPath(path string) string {
	const homePath = "/api/v2/homes"
	if strings.HasPrefix(path, homePath) {
		return homePath
	}
	return path
}

func (t tadoCallMetrics) Describe(ch chan<- *prometheus.Desc) {
	t.latency.Describe(ch)
	t.errors.Describe(ch)
}

func (t tadoCallMetrics) Collect(ch chan<- prometheus.Metric) {
	t.latency.Collect(ch)
	t.errors.Collect(ch)
}
