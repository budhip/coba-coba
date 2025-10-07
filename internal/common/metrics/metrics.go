package metrics

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	saramaMetrics "github.com/rcrowley/go-metrics"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"github.com/redis/go-redis/v9"
)

type Metrics interface {
	RegisterDB(db *sql.DB, role string, dbName string) error
	RegisterRedis(client *redis.Client, serviceName, namespace string) error
	RegisterFiberMiddleware(app *fiber.App, path, serviceName, namespace string) func(ctx *fiber.Ctx) error
	SaramaRegistry(name string, flushInterval time.Duration) saramaMetrics.Registry
	PrometheusRegisterer() prometheus.Registerer
	GetHTTPClientPrometheus() *HTTPClientPrometheusMetrics
	GetPublisherPrometheus() *PublisherPrometheusMetrics
	GetBalancePrometheus() *BalancePrometheusMetrics
}

type metrics struct {
	reg               prometheus.Registerer
	httpClientMetrics *HTTPClientPrometheusMetrics
	publisherMetrics  *PublisherPrometheusMetrics
	balanceMetrics    *BalancePrometheusMetrics
}

func New() Metrics {
	reg := prometheus.DefaultRegisterer
	return &metrics{
		reg:               reg,
		httpClientMetrics: newHTTPClientPrometheusMetrics(reg),
		publisherMetrics:  newPublisherPrometheusMetrics(reg),
		balanceMetrics:    newBalancePrometheusMetrics(reg),
	}
}

func (m *metrics) RegisterDB(db *sql.DB, role string, dbName string) error {
	return m.reg.Register(collectors.NewDBStatsCollector(db, fmt.Sprintf("%s_%s", dbName, role)))
}

func (m *metrics) RegisterRedis(client *redis.Client, serviceName, namespace string) error {
	return m.reg.Register(redisprometheus.NewCollector(BuildFQName(serviceName, namespace), "redis", client))
}

func (m *metrics) RegisterFiberMiddleware(app *fiber.App, path, serviceName, namespace string) func(ctx *fiber.Ctx) error {
	prom := fiberprometheus.NewWithRegistry(m.reg, BuildFQName(serviceName, namespace), FlattenName(serviceName), "http", nil)
	prom.RegisterAt(app, path)
	return prom.Middleware
}

func (m *metrics) SaramaRegistry(name string, flushInterval time.Duration) saramaMetrics.Registry {
	appMetrics := saramaMetrics.NewPrefixedRegistry(name + "_")
	prometheusClient := prometheusmetrics.NewPrometheusProvider(
		appMetrics, "", "", m.reg, flushInterval,
	)
	go prometheusClient.UpdatePrometheusMetrics()

	return appMetrics
}

func (m *metrics) PrometheusRegisterer() prometheus.Registerer {
	return m.reg
}

func (m *metrics) GetHTTPClientPrometheus() *HTTPClientPrometheusMetrics {
	return m.httpClientMetrics
}

func (m *metrics) GetPublisherPrometheus() *PublisherPrometheusMetrics {
	return m.publisherMetrics
}

func (m *metrics) GetBalancePrometheus() *BalancePrometheusMetrics {
	return m.balanceMetrics
}
