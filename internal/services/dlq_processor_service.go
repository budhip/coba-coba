package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/queueunicorn"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
)

type DLQProcessorService interface {
	SendNotificationOrderFailure(ctx context.Context, message models.FailedMessage) (err error)
	SendNotificationAccountFailure(ctx context.Context, message models.FailedMessage) (err error)
	SendNotificationRetryFailure(ctx context.Context, operation, message string) (err error)
	SendNotificationBalanceHvtFailure(ctx context.Context, message models.FailedMessage) (err error)

	GetStatusRetry(ctx context.Context, processRetryId string) (status models.StatusRetryDLQ, err error)
	UpsertStatusRetry(ctx context.Context, processRetryId string, status models.StatusRetryDLQ) (err error)

	// RetryAccountMutation is a method to retry create account by consuming failed message and create task
	RetryAccountMutation(ctx context.Context, message models.FailedMessage) (err error)
	RetryCreateOrderTransaction(ctx context.Context, message models.FailedMessage) (err error)
}

type dlqProcessor service

var _ DLQProcessorService = (*dlqProcessor)(nil)

func (d dlqProcessor) SendNotificationOrderFailure(ctx context.Context, message models.FailedMessage) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var orderPayload goacuanlib.Payload[goacuanlib.DataOrder]
	err = json.Unmarshal(message.Payload, &orderPayload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	refNumber := orderPayload.Body.Data.Order.RefNumber
	operation := "Process Consumer Transaction"
	isManualTrx := strings.Contains(refNumber, models.TransactionIDManualPrefix)
	if isManualTrx {
		operation = "Process Manual Transaction"
	}

	xlog.Error(ctx, "[DLQ-ERROR]", 
		xlog.String("operation", operation),
		xlog.String("ref_number", refNumber),
		xlog.String("error_message", message.Error))

	return nil
}

func (d dlqProcessor) SendNotificationAccountFailure(ctx context.Context, message models.FailedMessage) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var accountPayload goacuanlib.Payload[goacuanlib.DataAccount]
	err = json.Unmarshal(message.Payload, &accountPayload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	accountNumber := accountPayload.Body.Data.Account.AccountNumber
	operation := "Process Account Mutation"

	xlog.Error(ctx, "[DLQ-ERROR]", 
		xlog.String("operation", operation),
		xlog.String("account_number", accountNumber),
		xlog.String("error_message", message.Error))

	return nil
}

func (d dlqProcessor) SendNotificationBalanceHvtFailure(ctx context.Context, message models.FailedMessage) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var hvtBalancePayload models.UpdateBalanceHVTPayload
	err = json.Unmarshal(message.Payload, &hvtBalancePayload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	accountNumber := hvtBalancePayload.AccountNumber
	operation := "Process HVT Balance Update"

	xlog.Error(ctx, "[DLQ-ERROR]", 
		xlog.String("operation", operation),
		xlog.String("account_number", accountNumber),
		xlog.String("error_message", message.Error))

	return nil
}

func (d dlqProcessor) SendNotificationRetryFailure(ctx context.Context, operation, message string) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	xlog.Error(ctx, "[DLQ-ERROR]", 
		xlog.String("operation", fmt.Sprintf("[DLQ Retry Failure]: %s", operation)),
		xlog.String("error_message", message))

	return nil
}


func (d dlqProcessor) RetryAccountMutation(ctx context.Context, message models.FailedMessage) (err error) {
	monitor := monitoring.New(ctx)

	defer func() {
		monitor.Finish(monitoring.WithFinishCheckError(err))

		if err != nil {
			xlog.Error(ctx, "[DLQ-ERROR]", 
				xlog.String("operation", "failed to retry account stream"),
				xlog.String("error_message", err.Error()))
		}
	}()

	var payload goacuanlib.Payload[goacuanlib.DataAccount]
	err = json.Unmarshal(message.Payload, &payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	reqAct := payload.Body.Data.Account

	var metadata models.AccountMetadata
	if reqAct.Metadata != nil {
		if metadataMap, ok := reqAct.Metadata.(map[string]any); ok {
			metadata = metadataMap
		}
	}

	req := models.DoCreateAccountRequest{
		AccountNumber:   reqAct.AccountNumber,
		Name:            reqAct.Name,
		OwnerID:         reqAct.OwnerId,
		CategoryCode:    reqAct.CategoryCode,
		SubCategoryCode: reqAct.SubCategoryCode,
		ProductTypeName: reqAct.ProductTypeName,
		EntityCode:      reqAct.EntityCode,
		Currency:        reqAct.Currency,
		AltId:           reqAct.AltId,
		LegacyId:        (*models.AccountLegacyId)(reqAct.LegacyId),
		Status:          reqAct.Status,
		Metadata:        metadata,
	}

	status := models.StatusRetryDLQ{
		ProcessId:   fmt.Sprintf("account:%s:%s", req.AccountNumber, time.Now()),
		ProcessName: "account mutation",
		MaxRetry:    5,
	}

	d.logRetry(ctx, "retryable error", req.AccountNumber, true, req, message.Error)

	return d.sendJob(ctx, "v1/accounts", req, status)
}

func (d dlqProcessor) RetryCreateOrderTransaction(ctx context.Context, message models.FailedMessage) (err error) {
	monitor := monitoring.New(ctx)
	defer func() {
		monitor.Finish(monitoring.WithFinishCheckError(err))

		if err != nil {
			xlog.Error(ctx, "[DLQ-ERROR]", 
				xlog.String("operation", "failed to retry create order transaction"),
				xlog.String("error_message", err.Error()))
		}
	}()

	var payload goacuanlib.Payload[goacuanlib.DataOrder]
	err = json.Unmarshal(message.Payload, &payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	req := payload.Body.Data.Order

	status := models.StatusRetryDLQ{
		ProcessId:   fmt.Sprintf("order:%s:%s", req.RefNumber, time.Now()),
		ProcessName: "create order transaction",
		MaxRetry:    5,
	}

	d.logRetry(ctx, "retryable error", req.RefNumber, true, req, message.Error)

	return d.sendJob(ctx, "v1/orders", req, status)
}

func (d dlqProcessor) logRetry(ctx context.Context, desc string, id string, isRetry bool, request, err any) {
	xlog.Info(ctx, "[PROCESS-RETRY]",
		xlog.String("request-id", id),
		xlog.String("description", desc),
		xlog.Any("request", request),
		xlog.Bool("is-retry", isRetry),
		xlog.Any("error-causer", err))
}

func (d dlqProcessor) sendJob(ctx context.Context, url, body any, status models.StatusRetryDLQ) error {
	err := d.UpsertStatusRetry(ctx, status.ProcessId, status)
	if err != nil {
		return fmt.Errorf("failed to insert status retry dlq: %w", err)
	}

	req := queueunicorn.RequestJobHTTP{
		Name: queueunicorn.HttpRequestJobKey,
		Payload: queueunicorn.PayloadJob{
			Host:   fmt.Sprintf("%s/%s", d.srv.conf.HostGoFPTransaction, url),
			Method: http.MethodPost,
			Body:   body,
			Headers: queueunicorn.RequestHeaderJob(
				d.srv.conf.SecretKey,
				status.ProcessId,
				status.ToHeaders(),
			),
		},
		Options: queueunicorn.Options{
			ProcessAt: 5,
			MaxRetry:  status.MaxRetry,
		},
	}

	return d.srv.queueUnicornClient.SendJobHTTP(ctx, req)
}

func (d dlqProcessor) GetStatusRetry(ctx context.Context, processRetryId string) (status models.StatusRetryDLQ, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	cacheKey := models.GetCacheKeyStatusRetryDLQ(processRetryId)

	rawData, err := d.srv.cacheRepo.Get(ctx, cacheKey)
	if err != nil {
		return status, fmt.Errorf("failed to get status retry from cache: %w", err)
	}

	err = json.Unmarshal([]byte(rawData), &status)
	if err != nil {
		return status, fmt.Errorf("failed to unmarshal status retry: %w", err)
	}

	return
}

func (d dlqProcessor) UpsertStatusRetry(ctx context.Context, processRetryId string, status models.StatusRetryDLQ) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	rawData, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status retry: %w", err)
	}

	cacheKey := models.GetCacheKeyStatusRetryDLQ(processRetryId)

	err = d.srv.cacheRepo.Set(ctx, cacheKey, rawData, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to set status retry to cache: %w", err)
	}

	return
}
