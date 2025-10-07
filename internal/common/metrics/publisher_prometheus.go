package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PublisherPrometheusMetrics struct {
	kafkaPublishDurationHist *prometheus.HistogramVec
}

func newPublisherPrometheusMetrics(reg prometheus.Registerer) *PublisherPrometheusMetrics {
	kafkaPublishDurationHist := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_publisher_duration_seconds",
			Help:    "Duration of Kafka message publishing in seconds.",
			Buckets: []float64{0, 0.0001, 0.001, 0.010, 0.100, 0.200, 0.500, 1, 2, 5, 10, 100, 1000},
		},
		[]string{"topic", "success"},
	)

	reg.MustRegister(kafkaPublishDurationHist)

	return &PublisherPrometheusMetrics{kafkaPublishDurationHist}
}

func (m *PublisherPrometheusMetrics) GenerateMetrics(startTime time.Time, topic string, processErr error) {
	duration := time.Since(startTime).Seconds()

	// Record the duration in Prometheus metrics
	m.kafkaPublishDurationHist.WithLabelValues(topic, strconv.FormatBool(processErr == nil)).Observe(duration)
}
