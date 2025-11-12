package graceful

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"
	"golang.org/x/exp/slices"
)

type ProcessStarter func() error

type ProcessStopper func(ctx context.Context) error

type ProcessStartStopper interface {
	Start() ProcessStarter
	Stop() ProcessStopper
}

func StartProcessAtBackground(ps ...ProcessStarter) {
	for _, p := range ps {
		if p != nil {
			go func(_p func() error) {
				_ = _p()
			}(p)
		}
	}
}

func StopProcessAtBackground(duration time.Duration, ps ...ProcessStopper) {
	ctx := context.Background()

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigterm:
		xlog.Infof(ctx, "received signal: %v, initiating graceful shutdown...", sig)
		time.Sleep(10 * time.Second)

		StopProcess(duration, ps...)

	case sig := <-sigusr1:
		xlog.Infof(ctx, "received signal: %v, initiating graceful shutdown...", sig)
		time.Sleep(10 * time.Second)

		StopProcess(duration, ps...)
	}
}

func StopProcess(duration time.Duration, ps ...ProcessStopper) {
	ctx := context.Background()
	slices.Reverse(ps)

	xlog.Infof(ctx, "stopping %d services (timeout: %v)...", len(ps), duration)

	for i, p := range ps {
		func() {
			if p == nil {
				return
			}

			stopCtx, cancel := context.WithTimeout(context.Background(), duration)
			defer cancel()

			xlog.Infof(ctx, "stopping service %d/%d...", i+1, len(ps))

			if err := p(stopCtx); err != nil {
				xlog.Errorf(ctx, "error stopping service %d: %v", i+1, err)
			} else {
				xlog.Infof(ctx, "service %d stopped successfully", i+1)
			}
		}()
	}

	xlog.Info(ctx, "all services shutdown complete")
}
