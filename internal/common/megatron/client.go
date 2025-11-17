package megatron

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/ctxdata"
	"github.com/go-resty/resty/v2"
)

var logMessage = "[MEGATRON-CLIENT]"

// Client adalah interface untuk memanggil go-megatron service
type Client interface {
	// Transform single transaction
	Transform(ctx context.Context, req TransformRequest) (*TransformResponse, error)

	// Transform multiple transactions in batch
	BatchTransform(ctx context.Context, req BatchTransformRequest) (*BatchTransformResponse, error)

	// Get rule information
	GetRule(ctx context.Context, transactionType string) (*RuleResponse, error)
}

type client struct {
	baseURL    string
	secretKey  string
	httpClient *resty.Client
	metrics    metrics.Metrics
	config     config.Config
}

func New(
	configuration config.HTTPConfiguration,
	appConfig config.Config,
	metrics metrics.Metrics,
) Client {
	retryWaitTime := time.Duration(configuration.RetryWaitTime) * time.Millisecond

	restyClient := resty.New()
	restyClient = restyClient.AddRetryCondition(func(r *resty.Response, err error) bool {
		if r == nil {
			return false
		}
		_, shouldRetry := models.RetryableHTTPCodes[r.StatusCode()]
		return shouldRetry
	})

	restyClient = restyClient.
		SetRetryCount(configuration.RetryCount).
		SetRetryWaitTime(retryWaitTime).
		SetTimeout(configuration.Timeout)

	return &client{
		baseURL:    configuration.BaseURL,
		secretKey:  configuration.SecretKey,
		httpClient: restyClient,
		metrics:    metrics,
		config:     appConfig,
	}
}

func (c *client) Transform(ctx context.Context, req TransformRequest) (*TransformResponse, error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish()

	startTime := time.Now()
	url := fmt.Sprintf("%s/api/v1/transform", c.baseURL)

	logFields := []xlog.Field{
		xlog.String("url", url),
		xlog.String("transactionType", req.TransactionType),
		xlog.String("walletTransactionId", req.ParentTransaction.ID),
	}

	xlog.Info(ctx, logMessage, append(logFields, xlog.String("message", "send transform request"))...)

	httpRes, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json; charset=utf-8").
		SetHeader("Cache-Control", "no-cache").
		SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
		SetHeader("X-Secret-Key", c.secretKey).
		SetBody(req).
		Post(url)

	if err != nil {
		xlog.Error(ctx, logMessage, append(logFields, xlog.Err(err))...)
		return nil, fmt.Errorf("failed to send transform request: %w", err)
	}

	defer func() {
		if c.metrics != nil {
			c.metrics.GetHTTPClientPrometheus().Record(
				time.Since(startTime),
				"go-megatron",
				http.MethodPost,
				url,
				httpRes.StatusCode(),
			)
		}
	}()

	logFields = append(logFields,
		xlog.String("httpStatusCode", httpRes.Status()),
		xlog.Any("httpResponse", string(httpRes.Body())))

	if httpRes.StatusCode() != http.StatusOK {
		xlog.Error(ctx, logMessage, logFields...)
		return nil, fmt.Errorf("invalid response http code: got %d, body: %s",
			httpRes.StatusCode(), string(httpRes.Body()))
	}

	var res TransformResponse
	err = json.Unmarshal(httpRes.Body(), &res)
	if err != nil {
		xlog.Error(ctx, logMessage, append(logFields, xlog.Err(err))...)
		return nil, fmt.Errorf("error unmarshal response: %w", err)
	}

	xlog.Info(ctx, logMessage, append(logFields,
		xlog.String("message", "transform success"),
		xlog.Int("transactionCount", len(res.Transactions)),
		xlog.Int("executionTimeMs", res.Metadata.ExecutionTimeMs))...)

	return &res, nil
}

func (c *client) BatchTransform(ctx context.Context, req BatchTransformRequest) (*BatchTransformResponse, error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish()

	startTime := time.Now()
	url := fmt.Sprintf("%s/api/v1/transform/batch", c.baseURL)

	logFields := []xlog.Field{
		xlog.String("url", url),
		xlog.String("walletTransactionId", req.ParentTransaction.ID),
		xlog.Int("transformCount", len(req.Transforms)),
	}

	xlog.Info(ctx, logMessage, append(logFields, xlog.String("message", "send batch transform request"))...)

	httpRes, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json; charset=utf-8").
		SetHeader("Cache-Control", "no-cache").
		SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
		SetHeader("X-Secret-Key", c.secretKey).
		SetBody(req).
		Post(url)

	if err != nil {
		xlog.Error(ctx, logMessage, append(logFields, xlog.Err(err))...)
		return nil, fmt.Errorf("failed to send batch transform request: %w", err)
	}

	defer func() {
		if c.metrics != nil {
			c.metrics.GetHTTPClientPrometheus().Record(
				time.Since(startTime),
				"go-megatron",
				http.MethodPost,
				url,
				httpRes.StatusCode(),
			)
		}
	}()

	logFields = append(logFields,
		xlog.String("httpStatusCode", httpRes.Status()))

	if httpRes.StatusCode() != http.StatusOK {
		xlog.Error(ctx, logMessage, logFields...)
		return nil, fmt.Errorf("invalid response http code: got %d, body: %s",
			httpRes.StatusCode(), string(httpRes.Body()))
	}

	var res BatchTransformResponse
	err = json.Unmarshal(httpRes.Body(), &res)
	if err != nil {
		xlog.Error(ctx, logMessage, append(logFields, xlog.Err(err))...)
		return nil, fmt.Errorf("error unmarshal response: %w", err)
	}

	xlog.Info(ctx, logMessage, append(logFields,
		xlog.String("message", "batch transform success"),
		xlog.Int("transactionCount", len(res.Transactions)),
		xlog.Int("errorCount", len(res.Errors)),
		xlog.Int("executionTimeMs", res.Metadata.ExecutionTimeMs))...)

	return &res, nil
}

func (c *client) GetRule(ctx context.Context, transactionType string) (*RuleResponse, error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish()

	startTime := time.Now()
	url := fmt.Sprintf("%s/api/v1/rules/%s", c.baseURL, transactionType)

	logFields := []xlog.Field{
		xlog.String("url", url),
		xlog.String("transactionType", transactionType),
	}

	httpRes, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json; charset=utf-8").
		SetHeader("Cache-Control", "no-cache").
		SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
		SetHeader("X-Secret-Key", c.secretKey).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	defer func() {
		if c.metrics != nil {
			c.metrics.GetHTTPClientPrometheus().Record(
				time.Since(startTime),
				"go-megatron",
				http.MethodGet,
				url,
				httpRes.StatusCode(),
			)
		}
	}()

	if httpRes.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("rule not found for transaction type: %s", transactionType)
	}

	if httpRes.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("invalid response http code: got %d", httpRes.StatusCode())
	}

	var res RuleResponse
	err = json.Unmarshal(httpRes.Body(), &res)
	if err != nil {
		return nil, fmt.Errorf("error unmarshal response: %w", err)
	}

	return &res, nil
}

// Helper functions untuk convert wallet transaction ke request format

func (c *client) ConvertWalletTransactionToInput(
	ctx context.Context,
	wt models.WalletTransaction,
	account models.Account,
) WalletTransactionInput {
	return WalletTransactionInput{
		ID:                       wt.ID,
		AccountNumber:            wt.AccountNumber,
		DestinationAccountNumber: wt.DestinationAccountNumber,
		RefNumber:                wt.RefNumber,
		TransactionType:          wt.TransactionType,
		TransactionFlow:          string(wt.TransactionFlow),
		TransactionTime:          wt.TransactionTime,
		Description:              wt.Description,
		Metadata:                 wt.Metadata,
		Status:                   c.transformStatus(wt.Status),
		Account: AccountInfo{
			AccountNumber:   account.AccountNumber,
			Name:            account.Name,
			Entity:          account.Entity,
			CategoryCode:    account.CategoryCode,
			SubCategoryCode: account.SubCategoryCode,
		},
	}
}

func (c *client) BuildTransformContext() TransformContext {
	return TransformContext{
		SystemAccountNumber: c.config.AccountConfig.SystemAccountNumber,
		AccountNumberInsurancePremiumDisbursementByEntity: c.config.AccountConfig.AccountNumberInsurancePremiumDisbursementByEntity,
		MapAccountEntity: c.config.AccountConfig.MapAccountEntity,
	}
}

func (c *client) transformStatus(status models.WalletTransactionStatus) string {
	switch status {
	case models.WalletTransactionStatusSuccess:
		return "1" // TransactionStatusSuccessNum
	case models.WalletTransactionStatusCancel:
		return "2" // TransactionStatusCancelNum
	case models.WalletTransactionStatusPending:
		return "0" // TransactionStatusPendingNum
	default:
		return "0"
	}
}

func (c *client) ConvertResponseToTransactionReq(outputs []TransactionOutput) []models.TransactionReq {
	var results []models.TransactionReq

	for _, output := range outputs {
		results = append(results, models.TransactionReq{
			TransactionID:   output.TransactionID,
			FromAccount:     output.FromAccount,
			ToAccount:       output.ToAccount,
			FromNarrative:   output.FromNarrative,
			ToNarrative:     output.ToNarrative,
			TransactionDate: output.TransactionDate,
			Amount:          output.Amount.Decimal,
			Status:          output.Status,
			Method:          output.Method,
			TypeTransaction: output.TypeTransaction,
			Description:     output.Description,
			RefNumber:       output.RefNumber,
			Metadata:        output.Metadata,
			OrderTime:       output.OrderTime,
			OrderType:       output.OrderType,
			TransactionTime: output.TransactionTime,
			Currency:        output.Currency,
		})
	}

	return results
}
