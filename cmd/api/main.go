package main

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/cmd/setup"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/health"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"
)

func main() {
	var (
		ctx      = context.Background()
		starters []graceful.ProcessStarter
		stoppers []graceful.ProcessStopper
	)

	s, stopperContract, err := setup.Init("api")
	if err != nil {
		xlog.Fatalf(ctx, "failed to setup app: %v", err)
	}

	healthCheck := health.NewHealthCheck()
	balanceService := services.NewReconBalanceService(s.Service)

	httpServer := http.NewHTTPServer(ctx, s.Config, s.NewRelic,
		s.RepoCache,
		s.Service.Transaction,
		s.Service.Account,
		s.Service.Balance,
		s.Service.Entity,
		s.Service.Category,
		s.Service.SubCategory,
		s.Service.File,
		s.Service.MasterData,
		balanceService,
		s.Service.DLQProcessor,
		s.Service.WalletAccount,
		s.Service.WalletTrx,
		s.Metrics,
		s.Service.MoneyFlowCalc,
		healthCheck,
	)

	starters = append(starters, httpServer.Start())
	stoppers = append(stoppers, stopperContract...) // Added FIRST → Will stop LAST (Kafka, DB, Cache)
	stoppers = append(stoppers, httpServer.Stop())  // Added LAST → Will stop FIRST (HTTP)

	xlog.Info(ctx, "starting services in background...")
	graceful.StartProcessAtBackground(starters...)

	xlog.Info(ctx, "services started, waiting for shutdown signal...")

	// This blocks until shutdown signal is received (includes 10 second sleep)
	graceful.StopProcessAtBackground(ctx)
	healthCheck.Shutdown()
	graceful.StopProcess(ctx, s.Config.App.GracefulTimeout, stoppers...)
	xlog.Info(ctx, "all services stopped successfully!")
}
