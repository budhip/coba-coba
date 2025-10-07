package monitoring

import (
	"context"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

const (
	LayerRepository = "repositories"
	LayerService    = "services"
	LayerDelivery   = "deliveries"
	LayerUnknown    = "unknown"
)

type Monitor struct {
	ctx         context.Context
	segmentName string

	// layer is which this struct places, is it in repository, delivery, or service
	layer string

	start time.Time

	// add observability here
	segment *newrelic.Segment
}

type initOptions struct {
	layer       string
	segmentName string
}

type InitOption func(*initOptions)

func WithLayer(layer string) InitOption {
	return func(o *initOptions) {
		o.layer = layer
	}
}

func WithSegmentName(segmentName string) InitOption {
	return func(o *initOptions) {
		o.segmentName = segmentName
	}
}

func New(ctx context.Context, opts ...InitOption) *Monitor {
	fOpts := &initOptions{}
	for _, opt := range opts {
		opt(fOpts)
	}

	// TODO: add support singleton prometheus metrics,
	// so we can use it in on multiple places without use dependency injection
	// and make it like nr go-agent (easy to implement)

	if fOpts.segmentName == "" {

		// WARNING: don't refactor lines below, it will break the segment name
		pc, file, _, ok := runtime.Caller(1)
		if !ok {
			// Handle cases where runtime information is not available
			pc = 0
		}

		var segmentName string

		fn := runtime.FuncForPC(pc)
		if fn != nil {
			segmentName = getSegmentName(fn.Name())
		} else {
			segmentName = "unknown"
		}

		fOpts.segmentName = segmentName

		if strings.Contains(file, LayerRepository) {
			fOpts.layer = LayerRepository
		} else if strings.Contains(file, LayerService) {
			fOpts.layer = LayerService
		} else if strings.Contains(file, LayerDelivery) {
			fOpts.layer = LayerDelivery
		} else {
			fOpts.layer = LayerUnknown
		}
	}

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment(fOpts.segmentName)

	if segment != nil {
		segment.AddAttribute("layer", fOpts.layer)
	}

	return &Monitor{
		ctx:   ctx,
		layer: fOpts.layer,
		start: time.Now(),

		segmentName: fOpts.segmentName,
		segment:     segment,
	}
}

func NewMiddlewareRoundTripper(next http.RoundTripper) http.RoundTripper {
	// nr txn already exists on request.Context(), so no need to pass context

	if next == nil {
		next = http.DefaultTransport
	}

	return newrelic.NewRoundTripper(next)
}
