package tadotools

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/tado"
	"net/http"
	"strings"
	"time"
)

func GetInstrumentedTadoClient(username, password, secret string, metrics roundtripper.RoundTripMetrics) (*tado.APIClient, error) {
	c, err := tado.New(username, password, secret)
	if err != nil {
		return nil, fmt.Errorf("tado: %w", err)
	}

	c.HTTPClient = &http.Client{Transport: getInstrumentedRoundTripper(c.HTTPClient.Transport, metrics)}

	return c, nil
}

func getInstrumentedRoundTripper(rt http.RoundTripper, metrics roundtripper.RoundTripMetrics) http.RoundTripper {
	return roundtripper.New(
		roundtripper.WithInstrumentedRoundTripper(metrics),
		roundtripper.WithRoundTripper(rt),
	)
}

var _ roundtripper.RoundTripMetrics = &tadoCallMetrics{}

type tadoCallMetrics struct {
	roundtripper.RoundTripMetrics
}

func NewTadoCallMetrics(namespace, subsystem, application string) roundtripper.RoundTripMetrics {
	return &tadoCallMetrics{
		RoundTripMetrics: roundtripper.NewDefaultRoundTripMetrics(namespace, subsystem, application),
	}
}

func (t *tadoCallMetrics) Measure(req *http.Request, resp *http.Response, err error, duration time.Duration) {
	req2 := req.Clone(context.Background())
	req2.URL.Path = filterPath(req.URL.Path)
	t.RoundTripMetrics.Measure(req2, resp, err, duration)
}

func filterPath(path string) string {
	const homePath = "/api/v2/homes"
	if strings.HasPrefix(path, homePath) {
		return homePath
	}
	return path
}
