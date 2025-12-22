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

func StopProcessAtBackground(ctx context.Context) {
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)

	defer signal.Stop(sigterm)
	sig := <-sigterm
	xlog.Infof(ctx, "received signal %v, starting graceful shutdown", sig)
}

func StopProcess(ctx context.Context, gracePeriod time.Duration, ps ...ProcessStopper) {
	if gracePeriod <= 0 {
		gracePeriod = 10 * time.Second
	}

	xlog.Infof(ctx, "waiting %v before stopping services", gracePeriod)
	time.Sleep(gracePeriod)

	slices.Reverse(ps)

	xlog.Infof(ctx, "stopping %d services...", len(ps))

	for i, p := range ps {
		if p == nil {
			xlog.Warnf(ctx, "service %d is nil, skipping", i+1)
			continue
		}

		xlog.Infof(ctx, "stopping service %d/%d", i+1, len(ps))

		if err := p(ctx); err != nil {
			xlog.Errorf(ctx, "service %d shutdown error: %v", i+1, err)
		} else {
			xlog.Infof(ctx, "service %d stopped gracefully", i+1)
		}
	}

	xlog.Info(ctx, "shutdown completed, waiting for container exit")
}
