package monitoring

import (
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"
)

var messagePrefix = map[string]string{
	LayerRepository: "[REPOSITORY]",
	LayerService:    "[SERVICE]",
	LayerDelivery:   "[DELIVERY]",
	LayerUnknown:    "[-]",
}

type finishOptions struct {
	err        error
	xlogFields []xlog.Field
}

type FinishOption func(*finishOptions)

func WithFinishCheckError(err error) FinishOption {
	return func(o *finishOptions) {
		o.err = err
	}
}

func WithFinishXlogFields(fields ...xlog.Field) FinishOption {
	return func(o *finishOptions) {
		o.xlogFields = fields
	}
}

func (m *Monitor) Finish(opts ...FinishOption) {
	fOpts := &finishOptions{}
	for _, opt := range opts {
		opt(fOpts)
	}

	fOpts.xlogFields = append(fOpts.xlogFields,
		xlog.Duration("processDuration", time.Since(m.start)))

	if fOpts.err != nil {
		fOpts.xlogFields = append(
			fOpts.xlogFields,
			xlog.String("status", "error"),
			xlog.Err(fOpts.err))

		xlog.Warn(m.ctx, messagePrefix[m.layer], fOpts.xlogFields...)
	} else {
		// only log info from delivery layer & service layer to avoid duplicate log
		if m.layer == LayerDelivery || m.layer == LayerService {
			fOpts.xlogFields = append(
				fOpts.xlogFields,
				xlog.String("status", "success"))

			xlog.Info(m.ctx, messagePrefix[m.layer], fOpts.xlogFields...)
		}
	}

	if m.segment != nil {
		m.segment.End()
	}

	return
}
