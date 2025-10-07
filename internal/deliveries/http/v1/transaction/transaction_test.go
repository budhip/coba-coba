package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_getAllTransaction(t *testing.T) {
	testHelper := transactionTestHelper(t)
	type args struct {
		ctx  context.Context
		opts models.TransactionFilterOptions
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name      string
		urlCalled string
		args      args
		mockData  mockData
		doMock    func(args args, mockData mockData)
	}{
		{
			name:      "success data null",
			urlCalled: "/api/v1/transaction?limit=%v",
			args: args{
				ctx: context.Background(),
				opts: models.TransactionFilterOptions{
					Limit: 10,
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"collection","contents":[],"pagination":{"prev":"","next":"","totalEntries":0}}`,
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTrxService.EXPECT().GetAllTransaction(args.ctx, gomock.Any()).Return([]models.GetTransactionOut{}, 0, nil)
			},
		},
		{
			name:      "error limit",
			urlCalled: "/api/v1/transaction?limit=%v",
			args: args{
				ctx: context.Background(),
				opts: models.TransactionFilterOptions{
					Limit: -10,
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":"INVALID_VALUES","message":"the limit must be greater than zero"}`,
				wantCode: 400,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTrxService.EXPECT().GetAllTransaction(args.ctx, gomock.Any()).Return([]models.GetTransactionOut{}, 0, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(tt.urlCalled, tt.args.opts.Limit), nil)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, string(body))
		})
	}
}

func TestHandlerGenerateTransactionReport(t *testing.T) {
	testHelper := transactionTestHelper(t)

	type args struct {
		ctx context.Context
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name      string
		urlCalled string
		method    string
		args      args
		mockData  mockData
		doMock    func(args args, mockData mockData)
	}{
		{
			name:      "failed",
			urlCalled: "/api/v1/transaction/report",
			method:    http.MethodPost,
			args: args{
				ctx: context.Background(),
			},
			mockData: mockData{
				wantRes:  `{"status":"badRequest","error":{"code":400,"message":"assert.AnError general error for testing"}}`,
				wantCode: 400,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTrxService.EXPECT().GenerateTransactionReport(gomock.Any()).Return([]string{""}, assert.AnError)
			},
		},
		{
			name:      "success",
			urlCalled: "/api/v1/transaction/report",
			method:    http.MethodPost,
			args: args{
				ctx: context.Background(),
			},
			mockData: mockData{
				wantRes:  `{"code":200,"status":"success","message":"Successfully generate transaction report","data":["TEST"]}`,
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTrxService.EXPECT().GenerateTransactionReport(gomock.Any()).Return([]string{"TEST"}, nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			req := httptest.NewRequest(tt.method, fmt.Sprint(tt.urlCalled), nil)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, string(body))
		})
	}
}

func TestHandlerCreateTransaction(t *testing.T) {
	testHelper := transactionTestHelper(t)

	ct := time.Date(2024, 1, 1, 1, 1, 0, 0, time.UTC)

	type args struct {
		ctx            context.Context
		idempotencyKey string
		req            models.DoCreateTransactionRequest
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name      string
		urlCalled string
		method    string
		args      args
		mockData  mockData
		doMock    func(args args, mockData mockData)
	}{
		{
			name:      "success - create transaction",
			urlCalled: "/api/v1/transactions",
			method:    http.MethodPost,
			args: args{
				ctx:            context.Background(),
				idempotencyKey: "0f472815-8b37-4057-a594-a5617c91589d",
				req: models.DoCreateTransactionRequest{
					IsReserved:      false,
					FromAccount:     "1111111111",
					ToAccount:       "2222222222",
					Amount:          decimal.NewFromInt(10000),
					OrderType:       "DSB",
					TransactionType: "DSBAA",
					TransactionTime: ct,
					Description:     "test disburesement",
					RefNumber:       "TRX-1234567890",
					Metadata:        map[string]any{"test": "test"},
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"transaction","transactionId":"81a932b1-f4f5-445e-bd28-c2d365ff27b4","refNumber":"TRX-1234567890","orderType":"DSB","orderTypeName":"","method":"","transactionType":"DSBAA","transactionTypeName":"","transactionDate":"2024-01-01","transactionTime":"2024-01-01T08:01:00+07:00","fromAccount":"1111111111","fromAccountName":"","fromAccountProductTypeName":"","toAccount":"2222222222","toAccountName":"","toAccountProductTypeName":"","currency":"IDR","amount":"10000","status":"SUCCESS","description":"test disburesement","metadata":{"test":"test"},"createdAt":"0001-01-01T07:07:12+07:07","updatedAt":"0001-01-01T07:07:12+07:07"}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil)

				payload := args.req.ToTransactionReq()
				en, err := payload.ToRequest()
				assert.NoError(t, err)
				en.TransactionID = "81a932b1-f4f5-445e-bd28-c2d365ff27b4"
				testHelper.mockTrxService.EXPECT().
					StoreTransaction(
						gomock.Any(),
						gomock.AssignableToTypeOf(models.TransactionReq{}),
						models.TransactionStoreProcessNormal,
						gomock.Any()).
					Return(en.ToGetTransactionOut(map[string]string{}, map[string]string{}), nil)

				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:      "success - create reserve transaction",
			urlCalled: "/api/v1/transactions",
			method:    http.MethodPost,
			args: args{
				ctx:            context.Background(),
				idempotencyKey: "0f472815-8b37-4057-a594-a5617c91589d",
				req: models.DoCreateTransactionRequest{
					IsReserved:      true,
					FromAccount:     "1111111111",
					ToAccount:       "2222222222",
					Amount:          decimal.NewFromInt(10000),
					OrderType:       "DSB",
					TransactionType: "DSBAA",
					TransactionTime: ct,
					Description:     "test disburesement",
					RefNumber:       "TRX-1234567890",
					Metadata:        map[string]any{"test": "test"},
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"transaction","transactionId":"81a932b1-f4f5-445e-bd28-c2d365ff27b4","refNumber":"TRX-1234567890","orderType":"DSB","orderTypeName":"","method":"","transactionType":"DSBAA","transactionTypeName":"","transactionDate":"2024-01-01","transactionTime":"2024-01-01T08:01:00+07:00","fromAccount":"1111111111","fromAccountName":"","fromAccountProductTypeName":"","toAccount":"2222222222","toAccountName":"","toAccountProductTypeName":"","currency":"IDR","amount":"10000","status":"PENDING","description":"test disburesement","metadata":{"test":"test"},"createdAt":"0001-01-01T07:07:12+07:07","updatedAt":"0001-01-01T07:07:12+07:07"}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil)

				payload := args.req.ToTransactionReq()
				en, err := payload.ToRequest()
				assert.NoError(t, err)
				en.TransactionID = "81a932b1-f4f5-445e-bd28-c2d365ff27b4"
				testHelper.mockTrxService.EXPECT().
					StoreTransaction(
						gomock.Any(),
						gomock.AssignableToTypeOf(models.TransactionReq{}),
						models.TransactionStoreProcessReserved,
						gomock.Any()).
					Return(en.ToGetTransactionOut(map[string]string{}, map[string]string{}), nil)

				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:      "success - get from idempotency key",
			urlCalled: "/api/v1/transactions",
			method:    http.MethodPost,
			args: args{
				ctx:            context.Background(),
				idempotencyKey: "0f472815-8b37-4057-a594-a5617c91589d",
				req: models.DoCreateTransactionRequest{
					IsReserved:      true,
					FromAccount:     "1111111111",
					ToAccount:       "2222222222",
					Amount:          decimal.NewFromInt(10000),
					OrderType:       "DSB",
					TransactionType: "DSBAA",
					TransactionTime: ct,
					Description:     "test disburesement",
					RefNumber:       "TRX-1234567890",
					Metadata:        map[string]any{"test": "test"},
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"transaction","transactionId":"81a932b1-f4f5-445e-bd28-c2d365ff27b4","refNumber":"TRX-1234567890","orderType":"DSB","method":"","transactionType":"DSBAA","transactionDate":"2024-01-01","transactionTime":"0001-01-01 07:07:12","fromAccount":"1111111111","toAccount":"2222222222","amount":"10000","status":"PENDING","description":"test disburesement","metadata":{"test":"test"},"createdAt":"0001-01-01 07:07:12","updatedAt":"0001-01-01 07:07:12"}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(`{"status":"finished","fingerprint":"9411289b7e822dc27a0108f3bcb8139ab3f3053b","httpStatusCode":201,"responseBody":"{\"kind\":\"transaction\",\"transactionId\":\"81a932b1-f4f5-445e-bd28-c2d365ff27b4\",\"refNumber\":\"TRX-1234567890\",\"orderType\":\"DSB\",\"method\":\"\",\"transactionType\":\"DSBAA\",\"transactionDate\":\"2024-01-01\",\"transactionTime\":\"0001-01-01 07:07:12\",\"fromAccount\":\"1111111111\",\"toAccount\":\"2222222222\",\"amount\":\"10000\",\"status\":\"PENDING\",\"description\":\"test disburesement\",\"metadata\":{\"test\":\"test\"},\"createdAt\":\"0001-01-01 07:07:12\",\"updatedAt\":\"0001-01-01 07:07:12\"}","responseHeaders":{"Content-Type":"application/json"}}`, nil)
			},
		},
		{
			name:      "failed - error store from transaction service",
			urlCalled: "/api/v1/transactions",
			method:    http.MethodPost,
			args: args{
				ctx:            context.Background(),
				idempotencyKey: "0f472815-8b37-4057-a594-a5617c91589d",
				req: models.DoCreateTransactionRequest{
					IsReserved:      true,
					FromAccount:     "1111111111",
					ToAccount:       "2222222222",
					Amount:          decimal.NewFromInt(10000),
					OrderType:       "DSB",
					TransactionType: "DSBAA",
					TransactionTime: time.Now(),
					Description:     "test disburesement",
					RefNumber:       "TRX-1234567890",
					Metadata:        map[string]any{"test": "test"},
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil)

				payload := args.req.ToTransactionReq()
				en, err := payload.ToRequest()
				assert.NoError(t, err)
				en.TransactionID = "81a932b1-f4f5-445e-bd28-c2d365ff27b4"
				testHelper.mockTrxService.EXPECT().
					StoreTransaction(
						gomock.Any(),
						gomock.AssignableToTypeOf(models.TransactionReq{}),
						models.TransactionStoreProcessReserved,
						gomock.Any()).
					Return(en.ToGetTransactionOut(map[string]string{}, map[string]string{}), assert.AnError)

				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				testHelper.mockCacheRepository.EXPECT().
					Del(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			var b bytes.Buffer
			errEncode := json.NewEncoder(&b).Encode(tt.args.req)
			require.NoError(t, errEncode)

			req := httptest.NewRequest(tt.method, tt.urlCalled, &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			req.Header.Set("X-Idempotency-Key", tt.args.idempotencyKey)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, string(body))
		})
	}
}

func TestHandlerCreateBulkTransaction(t *testing.T) {
	testHelper := transactionTestHelper(t)

	type args struct {
		ctx context.Context
		req []models.TransactionReq
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name      string
		urlCalled string
		method    string
		args      args
		mockData  mockData
		doMock    func(args args, mockData mockData)
	}{
		{
			name:      "success - create bulk transaction",
			urlCalled: "/api/v1/transaction/bulk",
			method:    http.MethodPost,
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
						FromAccount:     "1202517699",
						ToAccount:       "123233333",
						FromNarrative:   "TOPUP.TRX",
						ToNarrative:     "TOPUP",
						TransactionDate: "2023-02-01",
						Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
						Status:          "",
						Method:          "TOPUP",
						TypeTransaction: "ACRF",
						Description:     "TOP UP",
						RefNumber:       "FT2303000001",
					}, {

						FromAccount:     "1202517699",
						ToAccount:       "123233333",
						FromNarrative:   "TOPUP.TRX",
						ToNarrative:     "TOPUP",
						TransactionDate: "2023-02-01",
						Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
						Status:          "",
						Method:          "TOPUP",
						TypeTransaction: "ACRF",
						Description:     "TOP UP",
						RefNumber:       "FT2303000002",
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"code":201,"status":"created"}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil)

				testHelper.mockTrxService.EXPECT().
					StoreBulkTransaction(gomock.Any(), args.req).
					Return(nil)

				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:      "failed - create bulk transaction - error from service",
			urlCalled: "/api/v1/transaction/bulk",
			method:    http.MethodPost,
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
						FromAccount:     "1202517699",
						ToAccount:       "123233333",
						FromNarrative:   "TOPUP.TRX",
						ToNarrative:     "TOPUP",
						TransactionDate: "2023-02-01",
						Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
						Status:          "",
						Method:          "TOPUP",
						TypeTransaction: "ACRF",
						Description:     "TOP UP",
						RefNumber:       "FT2303000001",
					}, {

						FromAccount:     "1202517699",
						ToAccount:       "123233333",
						FromNarrative:   "TOPUP.TRX",
						ToNarrative:     "TOPUP",
						TransactionDate: "2023-02-01",
						Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
						Status:          "",
						Method:          "TOPUP",
						TypeTransaction: "ACRF",
						Description:     "TOP UP",
						RefNumber:       "FT2303000002",
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"badRequest","error":{"code":400,"message":"assert.AnError general error for testing"}}`,
				wantCode: 400,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil)

				testHelper.mockTrxService.EXPECT().
					StoreBulkTransaction(gomock.Any(), args.req).
					Return(assert.AnError)

				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			var b bytes.Buffer
			errEncode := json.NewEncoder(&b).Encode(tt.args.req)
			require.NoError(t, errEncode)

			req := httptest.NewRequest(tt.method, tt.urlCalled, &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			req.Header.Set("X-Idempotency-Key", uuid.New().String())

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, string(body))
		})
	}
}

func TestHandlerCreateOrderTransaction(t *testing.T) {
	testHelper := transactionTestHelper(t)
	uuid, err := uuid.Parse("0afb1662-0763-4ec2-bc72-01dfe0f91ff8")
	assert.NoError(t, err)
	now, err := time.Parse(common.DateFormatYYYYMMDD, "2022-01-01")
	assert.NoError(t, err)
	statusActive := 0

	tests := []struct {
		name     string
		req      models.CreateOrderRequest
		wantRes  string
		wantCode int
		doMock   func(req models.CreateOrderRequest)
	}{
		{
			name: "happy path",
			req: models.CreateOrderRequest{
				OrderTime: &now,
				OrderType: "OrderType",
				RefNumber: "RefNumber",
				Transactions: []models.CreateOrderTransactionRequest{
					{
						ID:                   &uuid,
						Amount:               decimal.Zero,
						Currency:             "Currency",
						SourceAccountId:      "SourceAccountId",
						DestinationAccountId: "DestinationAccountId",
						Description:          "Description",
						Method:               "Method",
						TransactionType:      "TransactionType",
						TransactionTime:      &now,
						Status:               &statusActive,
						Meta:                 nil,
					},
				},
			},
			wantCode: fiber.StatusCreated,
			wantRes:  `{"kind":"order","orderTime":"2022-01-01T00:00:00Z","orderType":"OrderType","refNumber":"RefNumber","transactions":[{"id":"0afb1662-0763-4ec2-bc72-01dfe0f91ff8","amount":"0","currency":"Currency","sourceAccountId":"SourceAccountId","destinationAccountId":"DestinationAccountId","description":"Description","method":"Method","transactionType":"TransactionType","transactionTime":"2022-01-01T00:00:00Z","status":0,"meta":null}]}`,
			doMock: func(req models.CreateOrderRequest) {
				testHelper.mockTrxService.EXPECT().
					NewStoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf([]models.TransactionReq{})).
					Return(nil)
			},
		},
		{
			name: "failed - error service",
			req: models.CreateOrderRequest{
				OrderTime: &now,
				OrderType: "OrderType",
				RefNumber: "RefNumber",
				Transactions: []models.CreateOrderTransactionRequest{
					{
						ID:                   &uuid,
						Amount:               decimal.Zero,
						Currency:             "Currency",
						SourceAccountId:      "SourceAccountId",
						DestinationAccountId: "DestinationAccountId",
						Description:          "Description",
						Method:               "Method",
						TransactionType:      "TransactionType",
						TransactionTime:      &now,
						Status:               &statusActive,
						Meta:                 nil,
					},
				},
			},
			wantCode: fiber.StatusInternalServerError,
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			doMock: func(req models.CreateOrderRequest) {
				testHelper.mockTrxService.EXPECT().
					NewStoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf([]models.TransactionReq{})).
					Return(assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.req)
			}

			var b bytes.Buffer
			errEncode := json.NewEncoder(&b).Encode(tt.req)
			require.NoError(t, errEncode)

			req := httptest.NewRequest("POST", "/api/v1/transactions/orders", &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.wantRes, string(body))
			require.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}
