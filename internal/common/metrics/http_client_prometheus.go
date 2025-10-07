package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type HTTPClientPrometheusMetrics struct {
	apiRequestDurationHist *prometheus.HistogramVec
}

func newHTTPClientPrometheusMetrics(reg prometheus.Registerer) *HTTPClientPrometheusMetrics {
	apiRequestDurationHist := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "external_api_request_duration_seconds",
			Help:    "Duration of external API requests in seconds.",
			Buckets: []float64{0, 0.0001, 0.001, 0.010, 0.100, 0.200, 0.500, 1, 2, 5, 10, 100, 1000},
		},
		[]string{"service", "method", "endpoint", "response_code"},
	)

	reg.MustRegister(apiRequestDurationHist)

	return &HTTPClientPrometheusMetrics{apiRequestDurationHist}
}

func (m *HTTPClientPrometheusMetrics) Record(duration time.Duration, service, method, endpoint string, statusCode int) {
	m.apiRequestDurationHist.WithLabelValues(service, method, endpoint, fmt.Sprint(statusCode)).
		Observe(duration.Seconds())
}
