package metrics

import (
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	goMetrics "github.com/rcrowley/go-metrics"
)

type ConsumerMetrics struct {
	namespace          string
	subsystem          string
	flushInterval      time.Duration
	registerer         prometheus.Registerer
	metrics            goMetrics.Registry
	consumeTimeHist    *prometheus.HistogramVec
	processingTimeHist *prometheus.HistogramVec
	getMessageTimeHist *prometheus.HistogramVec
}

func NewConsumerMetrics(namespace, subsystem string, flushInterval time.Duration, reg prometheus.Registerer) *ConsumerMetrics {
	appMetrics := goMetrics.NewPrefixedRegistry(namespace + "_")

	consumeTimeHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kafka_consumer_consume_time",
		Help:    "consume time of kafka consumer handler",
		Buckets: []float64{0, 0.0001, 0.001, 0.010, 0.100, 0.200, 0.500, 1, 2, 5, 10, 100, 1000},
	}, []string{"topic", "consumer_group"})

	processingTimeHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kafka_consumer_processing_time",
		Help:    "processing time of kafka consumer handler",
		Buckets: []float64{0, 0.0001, 0.001, 0.010, 0.100, 0.200, 0.500, 1, 2, 5, 10, 100, 1000},
	}, []string{"topic", "success", "consumer_group"})

	getMessageTimeHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kafka_consumer_get_message_time",
		Help:    "get message time of kafka consumer handler",
		Buckets: []float64{0, 0.0001, 0.001, 0.010, 0.100, 0.200, 0.500, 1, 2, 5, 10, 100, 1000},
	}, []string{"topic", "consumer_group"})

	reg.MustRegister(consumeTimeHist, processingTimeHist, getMessageTimeHist)

	return &ConsumerMetrics{
		namespace:          namespace,
		subsystem:          subsystem,
		flushInterval:      flushInterval,
		registerer:         reg,
		metrics:            appMetrics,
		consumeTimeHist:    consumeTimeHist,
		processingTimeHist: processingTimeHist,
		getMessageTimeHist: getMessageTimeHist,
	}
}

func (m *ConsumerMetrics) Run() {
	prometheusClient := prometheusmetrics.NewPrometheusProvider(
		m.metrics, m.namespace, m.subsystem, m.registerer, m.flushInterval,
	)
	go prometheusClient.UpdatePrometheusMetrics()
}

func (m *ConsumerMetrics) GenerateMetrics(startTime time.Time, message *sarama.ConsumerMessage, processErr error) {
	endTime := time.Now() // time when a process consumes a message finished

	if message != nil {
		m.consumeTimeHist.WithLabelValues(message.Topic, m.namespace).
			Observe(endTime.Sub(message.Timestamp).Seconds())

		m.processingTimeHist.WithLabelValues(message.Topic, strconv.FormatBool(processErr == nil), m.namespace).
			Observe(endTime.Sub(startTime).Seconds())

		m.getMessageTimeHist.WithLabelValues(message.Topic, m.namespace).
			Observe(startTime.Sub(message.Timestamp).Seconds())
	}
}
