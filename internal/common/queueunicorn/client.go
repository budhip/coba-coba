package queueunicorn

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	queueunicorn "bitbucket.org/Amartha/go-queue-unicorn/client"
	queueunicornmodel "bitbucket.org/Amartha/go-queue-unicorn/client/model"
	xlog "bitbucket.org/Amartha/go-x/log"
)

type Client interface {
	SendJobHTTP(ctx context.Context, request RequestJobHTTP) error
}

type client struct {
	cfg    config.Config
	client queueunicorn.GoQueueServiceClient
}

func New(cfg config.Config) (Client, error) {
	queueUnicornClient, err := queueunicorn.NewGoQueueServiceClient(
		queueunicorn.Hostname(cfg.GoQueueUnicorn.BaseURL),
		queueunicorn.UseHttps(true))
	if err != nil {
		return nil, fmt.Errorf("failed init to go-queue-unicorn client: %v", err)
	}

	return &client{cfg: cfg, client: queueUnicornClient}, nil
}

func (c *client) SendJobHTTP(ctx context.Context, req RequestJobHTTP) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var resp *queueunicornmodel.JobResponse

	defer func() {
		logGoQueue(ctx, req, resp, err)
	}()

	resp, err = c.client.PostJob(ctx, &queueunicornmodel.RequestBody{
		Name: req.Name,
		Payload: queueunicornmodel.Payload{
			Host:    req.Payload.Host,
			Method:  req.Payload.Method,
			Body:    req.Payload.Body,
			Headers: req.Payload.Headers,
			Tag:     c.cfg.App.Name,
		},
		Options: queueunicornmodel.Option(req.Options),
	})
	if err != nil {
		err = models.GetErrMap(models.ErrKeyFailedFromExternalClient, err.Error())
		return
	}

	return
}

func logGoQueue(ctx context.Context, req, resp any, err error) {
	if err != nil {
		xlog.Error(ctx, "[GO-QUEUE]", xlog.String("status", "error"), xlog.Any("request", req), xlog.Any("response", resp), xlog.Err(err))
	} else {
		xlog.Info(ctx, "[GO-QUEUE]", xlog.String("status", "success"), xlog.Any("request", req), xlog.Any("response", resp))
	}
}
