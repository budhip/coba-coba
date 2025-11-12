package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	commonhttp "bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/health"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	xlog "bitbucket.org/Amartha/go-x/log"

	v1account "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/account"
	v1accountBalance "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/account_balances"
	v1category "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/category"
	v1entity "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/entity"
	v1Files "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/files"
	v1finSnapshot "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/fin_snapshot"
	v1internalWallet "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/internal_wallet"
	v1masterData "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/masterdata"
	v1moneyflow "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/money_flow_summaries"
	v1subcategory "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/sub_category"
	v1transaction "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/transaction"
	v1walletTrx "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v1/wallet_transaction"
	v2Files "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/v2/files"

	"bitbucket.org/Amartha/go-x/log/ctxdata"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	echoSwagger "github.com/swaggo/echo-swagger"

	// for swagger docs
	_ "bitbucket.org/Amartha/go-fp-transaction/docs"
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
		err := s.e.Shutdown(ctx)

		if err != nil {
			xlog.Errorf(ctx, "[SHUTDOWN] HTTP server error: %v", err)
		} else {
			xlog.Info(ctx, "[SHUTDOWN] HTTP server stopped successfully")
		}

		return err
	}
}

// @title GO FP TRANSACTION API DUCUMENTATION
// @version 1.0
// @description This is a go fp transaction api docs.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:9567
// @BasePath /api
// @schemes http
func NewHTTPServer(
	ctx context.Context,
	conf config.Config,
	nr *newrelic.Application,
	cacheRepo repositories.CacheRepository,
	transactionService services.TransactionService,
	accountService services.AccountService,
	balanceService services.BalanceService,
	entityService services.EntityService,
	categoryService services.CategoryService,
	subCategoryService services.SubCategoryService,
	fileService services.FileService,
	masterDataService services.MasterDataService,
	reconService services.ReconService,
	dlqProcessorService services.DLQProcessorService,
	walletAccountService services.WalletAccountService,
	walletTrxService services.WalletTrxService,
	metrics metrics.Metrics,
	moneyFlowService services.MoneyFlowService,
) *svc {
	app := echo.New()

	svc := &svc{
		e:               app,
		addr:            fmt.Sprintf(":%d", conf.App.HTTPPort),
		gracefulTimeout: conf.App.GracefulTimeout,
	}

	m := middleware.NewMiddleware(conf, cacheRepo, dlqProcessorService)
	// options middleware
	app.Pre(echomiddleware.RemoveTrailingSlash())
	app.Use(echomiddleware.Recover())
	app.Use(echomiddleware.RequestID())
	app.Use(m.Context())
	app.Use(m.Logger())

	if nr != nil {
		app.Use(nrecho.Middleware(nr))

		app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				txn := newrelic.FromContext(c.Request().Context())
				if txn != nil {
					txn.AddAttribute("x-correlation-id", ctxdata.GetCorrelationId(c.Request().Context()))
				}

				return next(c)
			}
		})
	}

	// pprof
	// Endpoint debug/pprof/
	env := config.StringToEnvironment(conf.App.Env)
	if env != config.PROD_ENV {
		pprof.Register(app)
	}

	// prometheus metrics
	app.Use(echoprometheus.NewMiddleware(conf.App.Name))
	app.GET("/metrics", echoprometheus.NewHandler())

	// swagger
	app.GET("/swagger/*", echoSwagger.WrapHandler)

	// apiGroup
	apiGroup := app.Group("/api")

	// health check
	health.New(apiGroup)

	// v1Group
	v1Group := apiGroup.Group("/v1")
	v1finSnapshot.New(v1Group, transactionService)

	// v1Group middleware
	v1Group.Use(m.InternalAuth)
	// v1Group register api
	v1transaction.New(v1Group, transactionService, m)
	v1account.New(v1Group, accountService, walletAccountService, balanceService, m)
	v1accountBalance.New(v1Group, balanceService)
	v1entity.New(v1Group, entityService)
	v1category.New(v1Group, categoryService)
	v1subcategory.New(v1Group, subCategoryService)
	v1Files.New(v1Group, fileService)
	v1masterData.New(v1Group, masterDataService)
	v1walletTrx.New(conf, v1Group, walletTrxService, accountService, m)
	v1internalWallet.New(v1Group, walletTrxService)
	v1moneyflow.New(v1Group, moneyFlowService)

	// v2Group
	v2Group := apiGroup.Group("/v2")
	// v2Group middleware
	v2Group.Use(m.InternalAuth)
	// v2Group register api
	v2Files.New(v2Group, fileService)

	// prepare an endpoint for 'Not Found'.
	app.Any("*", func(c echo.Context) error {
		errorMessage := fmt.Errorf("route '%s' does not exist in this API", c.Request().URL)
		return commonhttp.RestErrorResponse(c, nethttp.StatusNotFound, errorMessage)
	})

	return svc
}
