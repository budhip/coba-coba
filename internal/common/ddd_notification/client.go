package ddd_notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/ctxdata"
	"github.com/go-resty/resty/v2"
	"github.com/newrelic/go-agent/v3/newrelic"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type DDDNotification interface {
	SendEmail(ctx context.Context, request RequestEmail) error
}

type client struct {
	cfg        config.Config
	httpClient *resty.Client
}

func New(cfg config.Config) DDDNotification {
	retryWaitTime := time.Duration(cfg.DDDNotification.RetryWaitTime) * time.Millisecond
	restyClient := resty.New().SetRetryCount(cfg.DDDNotification.RetryCount).SetRetryWaitTime(retryWaitTime)
	restyClient.SetTransport(newrelic.NewRoundTripper(restyClient.GetClient().Transport))

	return &client{
		cfg:        cfg,
		httpClient: resty.New(),
	}
}

func (c *client) SendEmail(ctx context.Context, request RequestEmail) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	path := "/api/v1/email/mandrill"
	url := fmt.Sprintf("%s%s", c.cfg.DDDNotification.BaseUrl, path)

	xlog.Infof(ctx, "send request to %s with body %v", url, request)
	resp, err := c.httpClient.R().SetContext(ctx).
		SetHeader("Accept", "application/json;  charset=utf-8").
		SetHeader("Cache-Control", "no-cache").
		SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
		SetHeader("User-Agent", c.cfg.App.Name).
		SetBody(request).
		Post(url)
	if err != nil {
		return fmt.Errorf("error send request to %s: %w", url, err)
	}

	var response *ResponseSendMessage
	if err = json.Unmarshal(resp.Body(), &response); err != nil {
		return fmt.Errorf("error unmarshal response from %s: %w", url, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("error response from %s: %s", url, response.Message)
	}

	return nil
}
