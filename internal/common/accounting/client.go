package accounting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/cache"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/go-resty/resty/v2"
)

var logMessage = "[ACCOUNTING-CLIENT]"

type Client interface {
	GetInvestedAccountNumber(ctx context.Context, cihAccountNumber string) (accountNumber string, err error)
	GetReceivableAccountNumber(ctx context.Context, cihAccountNumber string) (accountNumber string, err error)
	GetLoanAdvancePayment(ctx context.Context, loanAccountNumber string) (accountNumber string, err error)
	GetLoanPartnerAccounts(ctx context.Context, loanAccountNumber string, loanKind string) (res ResponseGetListAccountNumber, err error)
}

type client struct {
	baseURL    string
	secretKey  string
	httpClient *resty.Client
	metrics    metrics.Metrics

	cache     cache.Client[string]
	cacheList cache.Client[ResponseGetListAccountNumber]
	ttlCache  time.Duration
}

func New(
	configuration config.HTTPConfiguration,
	metrics metrics.Metrics,
	cache cache.Client[string],
	cacheList cache.Client[ResponseGetListAccountNumber],
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
		SetTransport(monitoring.NewMiddlewareRoundTripper(restyClient.GetClient().Transport)).
		SetRetryCount(configuration.RetryCount).
		SetRetryWaitTime(retryWaitTime).
		SetTimeout(configuration.Timeout)

	return client{
		baseURL:    configuration.BaseURL,
		secretKey:  configuration.SecretKey,
		httpClient: restyClient,
		metrics:    metrics,

		cache:     cache,
		cacheList: cacheList,
		ttlCache:  10 * time.Minute,
	}
}

func (c client) GetInvestedAccountNumber(ctx context.Context, cihAccountNumber string) (res string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return c.cache.GetOrSet(ctx, cache.GetOrSetOpts[string]{
		Key: fmt.Sprintf("go-fp:pas:invested-account:%s", cihAccountNumber),
		TTL: c.ttlCache,
		Callback: func() (string, error) {
			startTime := time.Now()
			url := fmt.Sprintf("%s/api/v1/lender-accounts/%s", c.baseURL, cihAccountNumber)

			logFields := []xlog.Field{
				xlog.String("url", url),
				xlog.String("cihAccountNumber", cihAccountNumber),
			}

			xlog.Info(ctx, logMessage, append(logFields, xlog.String("message", "send request to go_accounting"))...)

			httpRes, err := c.httpClient.
				R().
				SetContext(ctx).
				SetHeader("Accept", "application/json;  charset=utf-8").
				SetHeader("Cache-Control", "no-cache").
				SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
				SetHeader("X-Secret-Key", c.secretKey).
				Get(url)
			if err != nil {
				return "", fmt.Errorf("failed send request: %w", err)
			}

			defer func() {
				if err != nil {
					xlog.Warn(ctx, logMessage, append(logFields, xlog.Err(err))...)
				}
				if c.metrics != nil {
					groupUrl := fmt.Sprintf("%s/api/v1/lender-accounts/:account-number", c.baseURL)
					c.metrics.GetHTTPClientPrometheus().Record(time.Since(startTime), SERVICE_NAME, httpRes.Request.Method, groupUrl, httpRes.StatusCode())
				}
			}()

			logFields = append(logFields,
				xlog.String("httpStatusCode", httpRes.Status()),
				xlog.Any("httpResponse", httpRes.Body()))

			if httpRes.StatusCode() != http.StatusOK {
				if httpRes.StatusCode() == http.StatusNotFound {
					return "", common.ErrAccountNumberNotFoundInAccounting
				}

				return "", fmt.Errorf("invalid response http code: got %d", httpRes.StatusCode())
			}

			var res ResponseGetLenderAccount
			err = json.Unmarshal(httpRes.Body(), &res)
			if err != nil {
				return "", fmt.Errorf("error unmarshal response: %w", err)
			}

			if res.InvestedAccountNumber == "" {
				return "", common.ErrInvestedAccountNumberNotFound
			}

			return res.InvestedAccountNumber, nil
		},
	})
}

func (c client) GetReceivableAccountNumber(ctx context.Context, cihAccountNumber string) (res string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return c.cache.GetOrSet(ctx, cache.GetOrSetOpts[string]{
		Key: fmt.Sprintf("go-fp:pas:receivable-account:%s", cihAccountNumber),
		TTL: c.ttlCache,
		Callback: func() (string, error) {
			startTime := time.Now()
			url := fmt.Sprintf("%s/api/v1/lender-accounts/%s", c.baseURL, cihAccountNumber)

			logFields := []xlog.Field{
				xlog.String("url", url),
				xlog.String("cihAccountNumber", cihAccountNumber),
			}

			xlog.Info(ctx, logMessage, append(logFields, xlog.String("message", "send request to go_accounting"))...)

			httpRes, err := c.httpClient.
				R().
				SetContext(ctx).
				SetHeader("Accept", "application/json;  charset=utf-8").
				SetHeader("Cache-Control", "no-cache").
				SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
				SetHeader("X-Secret-Key", c.secretKey).
				Get(url)
			if err != nil {
				return "", fmt.Errorf("failed send request: %w", err)
			}

			defer func() {
				if err != nil {
					xlog.Warn(ctx, logMessage, append(logFields, xlog.Err(err))...)
				}
				if c.metrics != nil {
					groupUrl := fmt.Sprintf("%s/api/v1/lender-accounts/:account-number", c.baseURL)
					c.metrics.GetHTTPClientPrometheus().Record(time.Since(startTime), SERVICE_NAME, httpRes.Request.Method, groupUrl, httpRes.StatusCode())
				}
			}()

			logFields = append(logFields,
				xlog.String("httpStatusCode", httpRes.Status()),
				xlog.Any("httpResponse", httpRes.Body()))

			if httpRes.StatusCode() != http.StatusOK {
				if httpRes.StatusCode() == http.StatusNotFound {
					return "", common.ErrAccountNumberNotFoundInAccounting
				}

				return "", fmt.Errorf("invalid response http code: got %d", httpRes.StatusCode())
			}

			var res ResponseGetLenderAccount
			err = json.Unmarshal(httpRes.Body(), &res)
			if err != nil {
				return "", fmt.Errorf("error unmarshal response: %w", err)
			}

			if res.ReceivableAccountNumber == "" {
				return "", common.ErrReceivableAccountNumberNotFound
			}

			return res.ReceivableAccountNumber, nil
		},
	})
}

func (c client) GetLoanAdvancePayment(ctx context.Context, loanAccountNumber string) (res string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return c.cache.GetOrSet(ctx, cache.GetOrSetOpts[string]{
		Key: fmt.Sprintf("go-fp:pas:loan-adv-payment:%s", loanAccountNumber),
		TTL: c.ttlCache,
		Callback: func() (string, error) {
			startTime := time.Now()
			url := fmt.Sprintf("%s/api/v1/loan-accounts/advance-account/%s", c.baseURL, loanAccountNumber)

			logFields := []xlog.Field{
				xlog.String("url", url),
				xlog.String("loanAccountNumber", loanAccountNumber),
			}

			httpRes, err := c.httpClient.
				R().
				SetContext(ctx).
				SetHeader("Accept", "application/json;  charset=utf-8").
				SetHeader("Cache-Control", "no-cache").
				SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
				SetHeader("X-Secret-Key", c.secretKey).
				Get(url)
			if err != nil {
				return "", fmt.Errorf("failed send request: %w", err)
			}

			defer func() {
				if err != nil {
					xlog.Warn(ctx, logMessage, append(logFields, xlog.Err(err))...)
				}
				if c.metrics != nil {
					groupUrl := fmt.Sprintf("%s/api/v1/loan-accounts/advance-account/:account-number", c.baseURL)
					c.metrics.GetHTTPClientPrometheus().Record(time.Since(startTime), SERVICE_NAME, httpRes.Request.Method, groupUrl, httpRes.StatusCode())
				}
			}()

			if httpRes.StatusCode() != http.StatusOK {
				if httpRes.StatusCode() == http.StatusNotFound {
					return "", common.ErrAccountNumberNotFoundInAccounting
				}
				return "", fmt.Errorf("invalid response http code: got %d", httpRes.StatusCode())
			}

			var res ResponseGetLoanAccount
			err = json.Unmarshal(httpRes.Body(), &res)
			if err != nil {
				return "", fmt.Errorf("error unmarshal response: %w", err)
			}

			if res.LoanAdvancePaymentAccountNumber == "" {
				return "", common.ErrLoanAdvanceAccountNumberNotFound
			}

			return res.LoanAdvancePaymentAccountNumber, nil
		},
	})
}

func (c client) GetLoanPartnerAccounts(ctx context.Context, loanAccountNumber string, loanKind string) (res ResponseGetListAccountNumber, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	return c.cacheList.GetOrSet(ctx, cache.GetOrSetOpts[ResponseGetListAccountNumber]{
		Key: fmt.Sprintf("go-fp:pas:loan-partner-accounts:%s", loanAccountNumber),
		TTL: c.ttlCache,
		Callback: func() (ResponseGetListAccountNumber, error) {
			startTime := time.Now()
			url := fmt.Sprintf("%s/api/v1/loan-partner-accounts", c.baseURL)

			logFields := []xlog.Field{
				xlog.String("url", url),
				xlog.String("loanAccountNumber", loanAccountNumber),
			}

			restClient := c.httpClient.
				R().
				SetContext(ctx).
				SetHeader("Accept", "application/json;  charset=utf-8").
				SetHeader("Cache-Control", "no-cache").
				SetHeader("X-Correlation-Id", ctxdata.GetCorrelationId(ctx)).
				SetHeader("X-Secret-Key", c.secretKey)

			if loanKind != "" {
				restClient = restClient.SetQueryParam("loanKind", loanKind)
			}

			if loanAccountNumber != "" {
				restClient = restClient.SetQueryParam("loanAccountNumber", loanAccountNumber)
			}

			httpRes, err := restClient.Get(url)
			if err != nil {
				return ResponseGetListAccountNumber{}, fmt.Errorf("failed send request: %w", err)
			}

			defer func() {
				if err != nil {
					xlog.Warn(ctx, logMessage, append(logFields, xlog.Err(err))...)
				}
				if c.metrics != nil {
					groupUrl := fmt.Sprintf("%s/api/v1/loan-partner-accounts", c.baseURL)
					c.metrics.GetHTTPClientPrometheus().Record(time.Since(startTime), SERVICE_NAME, httpRes.Request.Method, groupUrl, httpRes.StatusCode())
				}
			}()

			if httpRes.StatusCode() != http.StatusOK {
				if httpRes.StatusCode() == http.StatusNotFound {
					return ResponseGetListAccountNumber{}, common.ErrAccountNumberNotFoundInAccounting
				}
				return ResponseGetListAccountNumber{}, fmt.Errorf("invalid response http code: got %d", httpRes.StatusCode())
			}

			var res ResponseGetListAccountNumber
			err = json.Unmarshal(httpRes.Body(), &res)
			if err != nil {
				return ResponseGetListAccountNumber{}, fmt.Errorf("error unmarshal response: %w", err)
			}

			return res, nil
		},
	})
}
