package httpclient

import (
	"context"
	"fmt"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"

	"github.com/go-resty/resty/v2"
)

type RequestWrapper struct {
	client      *resty.Client
	metrics     metrics.Metrics
	serviceName string
	logPrefix   string
}

func NewRequestWrapper(client *resty.Client, metrics metrics.Metrics, serviceName, logPrefix string) *RequestWrapper {
	return &RequestWrapper{
		client:      client,
		metrics:     metrics,
		serviceName: serviceName,
		logPrefix:   logPrefix,
	}
}

func (w *RequestWrapper) DoRequest(ctx context.Context, method, url string, reqFunc func(*resty.Request) *resty.Request) (*resty.Response, error) {
	startTime := time.Now()

	logFields := []xlog.Field{
		xlog.String("url", url),
		xlog.String("method", method),
	}

	xlog.Info(ctx, w.logPrefix, append(logFields, xlog.String("message", "send request"))...)

	req := w.client.R().SetContext(ctx)
	if reqFunc != nil {
		req = reqFunc(req)
	}

	var httpRes *resty.Response
	var err error

	switch method {
	case "GET":
		httpRes, err = req.Get(url)
	case "POST":
		httpRes, err = req.Post(url)
	case "PUT":
		httpRes, err = req.Put(url)
	case "DELETE":
		httpRes, err = req.Delete(url)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if err != nil {
		xlog.Warn(ctx, w.logPrefix, append(logFields, xlog.Err(err))...)
		return nil, fmt.Errorf("failed send request: %w", err)
	}

	if w.metrics != nil {
		w.metrics.GetHTTPClientPrometheus().Record(
			time.Since(startTime),
			w.serviceName,
			method,
			url,
			httpRes.StatusCode(),
		)
	}

	logFields = append(logFields,
		xlog.String("httpStatusCode", httpRes.Status()),
		xlog.Any("httpResponse", httpRes.Body()),
	)

	if httpRes.StatusCode() < 200 || httpRes.StatusCode() >= 300 {
		xlog.Warn(ctx, w.logPrefix, logFields...)
	} else {
		xlog.Info(ctx, w.logPrefix, logFields...)
	}

	return httpRes, nil
}
