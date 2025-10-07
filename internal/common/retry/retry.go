package retry

import (
	"context"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	"github.com/cenkalti/backoff/v4"
)

const DefaultMaxRetries uint64 = 3

type Retryer interface {
	Retry(ctx context.Context, operation, dlqCallback func() error) error
	StopRetryWithErr(err error) error
}

type exponentialBackoff struct {
	ebCfg *config.ExponentialBackOffConfig
}

/*
NewExponentialBackOff will init Retryer interface.
This retryer implement exponential backoff mechanism.

Example:

Retry(consumer.ctx, func() error { return someOperation() }, func() error { return dlqOperation() })
*/
func NewExponentialBackOff(ebCfg *config.ExponentialBackOffConfig) Retryer {
	if ebCfg.MaxBackoffTime < 0 {
		ebCfg.MaxBackoffTime = backoff.DefaultMaxElapsedTime
	}

	if ebCfg.BackoffMultiplier <= 0 {
		ebCfg.BackoffMultiplier = backoff.DefaultMultiplier
	}

	if ebCfg.MaxRetries <= 0 {
		ebCfg.MaxRetries = DefaultMaxRetries
	}

	return &exponentialBackoff{ebCfg: ebCfg}
}

/*
Retry will create ExponentialBackOff instance for every execution.

You will need to pass 2 function. "operation" func will keep retried until certain condition is meet; and "callback" func is called if the "operation" func is keep failing.

This Retry function will return the error from "callback" func.
*/
func (r *exponentialBackoff) Retry(ctx context.Context, operation, dlqCallback func() error) error {
	eb := backoff.NewExponentialBackOff()
	eb.MaxElapsedTime = r.ebCfg.MaxBackoffTime
	eb.Multiplier = r.ebCfg.BackoffMultiplier

	err := backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(eb, r.ebCfg.MaxRetries), ctx))
	if err != nil {
		// Handle DLQ
		xlog.Debugf(ctx, "DLQ reached with err: %v\n", err)
		if err := dlqCallback(); err != nil {
			return err
		}
		return nil
	}

	return nil
}

// StopRetryWithErr will stop retrying and return the error.
// This function should be called inside "operation" func.
func (r *exponentialBackoff) StopRetryWithErr(err error) error {
	return backoff.Permanent(err)
}
