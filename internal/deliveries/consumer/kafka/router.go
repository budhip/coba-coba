package kafkaconsumer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/health"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type svc struct {
	e               *fiber.App
	addr            string
	gracefulTimeout time.Duration
}

var _ graceful.ProcessStartStopper = (*svc)(nil)

func (s *svc) Start() graceful.ProcessStarter {
	return func() error {
		err := s.e.Listen(s.addr)
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

func (s *svc) Stop() graceful.ProcessStopper {
	return func(ctx context.Context) error {
		return s.e.ShutdownWithTimeout(s.gracefulTimeout)
	}
}

func NewHTTPServer(
	ctx context.Context,
	conf config.Config,
	metrics metrics.Metrics,
) *svc {
	app := fiber.New(fiber.Config{
		AppName:      fmt.Sprintf("%s-consumer-server", conf.App.Name),
		ServerHeader: "Go FP Transaction Consumer Server",
	})
	svc := &svc{e: app, addr: fmt.Sprintf(":%d", conf.MessageBroker.HTTPPort), gracefulTimeout: conf.App.GracefulTimeout}

	// options middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New())

	// pprof
	// Endpoint debug/pprof/
	app.Use(pprof.New())

	// prometheus metrics
	app.Use(metrics.RegisterFiberMiddleware(app, "/metrics", conf.App.Name, "consumer"))

	// apiGroup
	apiGroup := app.Group("/api")

	// health check
	health.New(apiGroup)

	return svc
}
