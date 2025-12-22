package kafkaconsumer

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/health"
)

type svc struct {
	e               *echo.Echo
	addr            string
	gracefulTimeout time.Duration
}

var _ graceful.ProcessStartStopper = (*svc)(nil)

func (s *svc) Start() graceful.ProcessStarter {
	return func() error {
		return s.e.Start(s.addr)
	}
}

func (s *svc) Stop() graceful.ProcessStopper {
	return func(ctx context.Context) error {
		return s.e.Shutdown(ctx)
	}
}

func NewHTTPServer(
	ctx context.Context,
	conf config.Config,
	metrics metrics.Metrics,
	check *health.HealthCheck,
) *svc {
	app := echo.New()
	svc := &svc{e: app, addr: fmt.Sprintf(":%d", conf.MessageBroker.HTTPPort), gracefulTimeout: conf.App.GracefulTimeout}

	// options middleware
	app.Pre(echomiddleware.RemoveTrailingSlash())
	app.Use(echomiddleware.Recover())
	app.Use(echomiddleware.RequestID())

	// pprof
	// Endpoint debug/pprof/
	env := config.StringToEnvironment(conf.App.Env)
	if env != config.PROD_ENV {
		pprof.Register(app)
	}

	// prometheus metrics
	app.Use(echoprometheus.NewMiddleware(fmt.Sprintf("%s_consumer", conf.App.Name)))
	app.GET("/metrics", echoprometheus.NewHandler())

	// apiGroup
	apiGroup := app.Group("/api")

	// health check
	check.Route(apiGroup.Group("/health"))

	return svc
}
