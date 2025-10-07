package main

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/cmd/setup"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http"
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
		timeout := 5 * time.Second
		if s != nil && s.Config.App.GracefulTimeout != 0 {
			timeout = s.Config.App.GracefulTimeout
		}

		graceful.StopProcess(timeout, stopperContract...)

		xlog.Fatalf(ctx, "failed to setup app: %v", err)
	}

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
		services.NewReconBalanceService(s.Service),
		s.Service.DLQProcessor,
		s.Service.WalletAccount,
		s.Service.WalletTrx,
		s.Metrics,
	)

	starters = append(starters, httpServer.Start())
	stoppers = append(stoppers, httpServer.Stop())
	stoppers = append(stoppers, stopperContract...)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		graceful.StartProcessAtBackground(starters...)
		graceful.StopProcessAtBackground(s.Config.App.GracefulTimeout, stoppers...)
		wg.Done()
	}()
	wg.Wait()
	xlog.Info(ctx, "http server stopped!")
}
