package graceful

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigterm:
		StopProcess(duration, ps...)
	case <-sigusr1:
		StopProcess(duration, ps...)
	}
}

func StopProcess(duration time.Duration, ps ...ProcessStopper) {
	slices.Reverse(ps)

	for _, p := range ps {
		func() {
			if p == nil {
				return
			}
			ctx, stop := context.WithTimeout(context.Background(), duration)
			defer stop()
			_ = p(ctx)
		}()
	}
}
