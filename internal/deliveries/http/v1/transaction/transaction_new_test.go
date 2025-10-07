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

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_publishTransaction(t *testing.T) {
	testHelper := transactionTestHelper(t)

	type args struct {
		ctx context.Context
		req models.DoPublishTransactionRequest
	}
	mockPublishTransactionReq := models.DoPublishTransactionRequest{
		FromAccount:     "666",
		ToAccount:       "777",
		Amount:          "420000.69",
		Method:          "BANK_TRANSFER",
		TransactionType: "DISBNORMBPEBSA",
		TransactionDate: "2006-01-02",
		OrderType:       "DISBURSEMENT",
		RefNumber:       "12345abcd",
		Description:     "TEST DISBURSEMENT",
	}
	mockPublishTransactionRes := models.DoPublishTransactionResponse{
		Kind:            "transaction",
		FromAccount:     "666",
		ToAccount:       "777",
		Amount:          "420000.69",
		Method:          "BANK_TRANSFER",
		TransactionType: "DISBNORMBPEBSA",
		TransactionDate: "2006-01-02",
		OrderType:       "DISBURSEMENT",
		RefNumber:       "12345abcd",
		Description:     "TEST DISBURSEMENT",
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
			name:      "success",
			urlCalled: "/api/v1/transaction/publish",
			args: args{
				ctx: context.Background(),
				req: mockPublishTransactionReq,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTrxService.EXPECT().PublishTransaction(args.ctx, args.req).Return(mockPublishTransactionRes, nil)
			},
			mockData: mockData{
				wantRes:  `{"kind":"transaction","fromAccount":"666","toAccount":"777","amount":"420000.69","method":"BANK_TRANSFER","transactionType":"DISBNORMBPEBSA","transactionDate":"2006-01-02","orderType":"DISBURSEMENT","refNumber":"12345abcd","description":"TEST DISBURSEMENT"}`,
				wantCode: 201,
			},
		},
		{
			name:      "error validating required",
			urlCalled: "/api/v1/transaction/publish",
			args: args{
				ctx: context.Background(),
				req: models.DoPublishTransactionRequest{},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"MISSING_FIELD","field":"fromAccount","message":"field is missing"},{"code":"MISSING_FIELD","field":"toAccount","message":"field is missing"},{"code":"MISSING_FIELD","field":"amount","message":"field is missing"},{"code":"MISSING_FIELD","field":"transactionType","message":"field is missing"},{"code":"MISSING_FIELD","field":"transactionDate","message":"field is missing"},{"code":"MISSING_FIELD","field":"orderType","message":"field is missing"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "error internal server",
			urlCalled: "/api/v1/transaction/publish",
			args: args{
				ctx: context.Background(),
				req: mockPublishTransactionReq,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTrxService.EXPECT().PublishTransaction(args.ctx, args.req).Return(mockPublishTransactionRes, common.ErrInternalServerError)
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"internal server error"}`,
				wantCode: 500,
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
			err := json.NewEncoder(&b).Encode(tt.args.req)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, tt.urlCalled, &b)
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

func Test_Handler_getByTypeAndRefNumber(t *testing.T) {
	testHelper := transactionTestHelper(t)

	tests := []struct {
		name     string
		request  *models.TransactionGetByTypeAndRefNumberRequest
		wantRes  string
		wantCode int
		doMock   func(request *models.TransactionGetByTypeAndRefNumberRequest)
	}{
		{
			name: "happy path",
			request: &models.TransactionGetByTypeAndRefNumberRequest{
				TransactionType: "TransactionType",
				RefNumber:       "RefNumber",
			},
			wantRes:  `{"kind":"transaction","transactionId":"","refNumber":"","orderType":"","orderTypeName":"","method":"","transactionType":"","transactionTypeName":"","transactionDate":"0001-01-01","transactionTime":"0001-01-01T07:07:12+07:07","fromAccount":"","fromAccountName":"","fromAccountProductTypeName":"","toAccount":"","toAccountName":"","toAccountProductTypeName":"","currency":"","amount":"0","status":"","description":"","metadata":null,"createdAt":"0001-01-01T07:07:12+07:07","updatedAt":"0001-01-01T07:07:12+07:07"}`,
			wantCode: 200,
			doMock: func(request *models.TransactionGetByTypeAndRefNumberRequest) {
				testHelper.mockTrxService.EXPECT().
					GetByTransactionTypeAndRefNumber(gomock.AssignableToTypeOf(context.Background()), request).
					Return(&models.GetTransactionOut{}, nil)
			},
		},
		{
			name: "failed - err db",
			request: &models.TransactionGetByTypeAndRefNumberRequest{
				TransactionType: "TransactionType",
				RefNumber:       "RefNumber",
			},
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			wantCode: 500,
			doMock: func(request *models.TransactionGetByTypeAndRefNumberRequest) {
				testHelper.mockTrxService.EXPECT().
					GetByTransactionTypeAndRefNumber(gomock.AssignableToTypeOf(context.Background()), request).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.request)
			}

			var b bytes.Buffer
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transaction/%s/%s", tc.request.TransactionType, tc.request.RefNumber), &b)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.wantCode, resp.StatusCode)
			require.Equal(t, tc.wantRes, string(body))
		})
	}
}

func Test_Handler_UpdateStatusReserved(t *testing.T) {
	testHelper := transactionTestHelper(t)

	tests := []struct {
		name     string
		request  *models.UpdateStatusReservedTransactionRequest
		wantRes  string
		wantCode int
		doMock   func(request *models.UpdateStatusReservedTransactionRequest)
	}{
		{
			name: "happy path - commit",
			request: &models.UpdateStatusReservedTransactionRequest{
				TransactionId: "123",
				Status:        models.TransactionRequestCommitStatus,
			},
			wantRes:  `{"kind":"transaction","transactionId":"123","status":"SUCCESS"}`,
			wantCode: fiber.StatusOK,
			doMock: func(request *models.UpdateStatusReservedTransactionRequest) {
				testHelper.mockTrxService.EXPECT().
					CommitReservedTransaction(gomock.AssignableToTypeOf(context.Background()), request.TransactionId, gomock.Any()).
					Return(&models.Transaction{TransactionID: request.TransactionId, Status: "1"}, nil)
			},
		},
		{
			name: "happy path - cancel",
			request: &models.UpdateStatusReservedTransactionRequest{
				TransactionId: "123",
				Status:        models.TransactionRequestCancelStatus,
			},
			wantRes:  `{"kind":"transaction","transactionId":"123","status":"CANCEL"}`,
			wantCode: fiber.StatusOK,
			doMock: func(request *models.UpdateStatusReservedTransactionRequest) {
				testHelper.mockTrxService.EXPECT().
					CancelReservedTransaction(gomock.AssignableToTypeOf(context.Background()), request.TransactionId).
					Return(&models.Transaction{TransactionID: request.TransactionId, Status: "2"}, nil)
			},
		},
		{
			name: "failed - internal err",
			request: &models.UpdateStatusReservedTransactionRequest{
				TransactionId: "123",
				Status:        models.TransactionRequestCommitStatus,
			},
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			wantCode: fiber.StatusInternalServerError,
			doMock: func(request *models.UpdateStatusReservedTransactionRequest) {
				testHelper.mockTrxService.EXPECT().
					CommitReservedTransaction(gomock.AssignableToTypeOf(context.Background()), request.TransactionId, gomock.Any()).
					Return(nil, assert.AnError)
			},
		},
		{
			name: "failed - trx not found",
			request: &models.UpdateStatusReservedTransactionRequest{
				TransactionId: "123",
				Status:        models.TransactionRequestCommitStatus,
			},
			wantRes:  `{"status":"error","code":"DATA_NOT_FOUND","message":"data not found"}`,
			wantCode: fiber.StatusNotFound,
			doMock: func(request *models.UpdateStatusReservedTransactionRequest) {
				testHelper.mockTrxService.EXPECT().
					CommitReservedTransaction(gomock.AssignableToTypeOf(context.Background()), request.TransactionId, gomock.Any()).
					Return(nil, models.GetErrMap(models.ErrKeyDataNotFound))
			},
		},
		{
			name: "failed - trx not reserved",
			request: &models.UpdateStatusReservedTransactionRequest{
				TransactionId: "123",
				Status:        models.TransactionRequestCommitStatus,
			},
			wantRes:  `{"status":"error","code":409,"message":"transaction status not reserved"}`,
			wantCode: fiber.StatusConflict,
			doMock: func(request *models.UpdateStatusReservedTransactionRequest) {
				testHelper.mockTrxService.EXPECT().
					CommitReservedTransaction(gomock.AssignableToTypeOf(context.Background()), request.TransactionId, gomock.Any()).
					Return(nil, common.ErrTransactionNotReserved)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.request)
			}

			var b bytes.Buffer
			err := json.NewEncoder(&b).Encode(tc.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/transactions/%s", tc.request.TransactionId), &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.wantCode, resp.StatusCode)
			require.Equal(t, tc.wantRes, string(body))
		})
	}
}

func Test_Handler_getTransactionStatusCount(t *testing.T) {
	testHelper := transactionTestHelper(t)

	tests := []struct {
		name     string
		request  *models.DoGetStatusCountTransactionRequest
		wantRes  string
		wantCode int
		doMock   func(request *models.DoGetStatusCountTransactionRequest)
	}{
		{
			name: "happy path",
			request: &models.DoGetStatusCountTransactionRequest{
				Threshold: 100,
			},
			wantRes:  `{"kind":"status-count-transaction","exceedThreshold":false,"threshold":100}`,
			wantCode: 200,
			doMock: func(request *models.DoGetStatusCountTransactionRequest) {
				opts, threshold, _ := request.ToFilterOpts()

				testHelper.mockTrxService.EXPECT().
					GetStatusCount(gomock.Any(), threshold, *opts).
					Return(models.StatusCountTransaction{
						ExceedThreshold: false,
						Threshold:       threshold,
					}, nil)
			},
		},
		{
			name: "failed - err db",
			request: &models.DoGetStatusCountTransactionRequest{
				Threshold: 100,
			},
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			wantCode: 500,
			doMock: func(request *models.DoGetStatusCountTransactionRequest) {
				opts, threshold, _ := request.ToFilterOpts()

				testHelper.mockTrxService.EXPECT().
					GetStatusCount(gomock.Any(), threshold, *opts).
					Return(models.StatusCountTransaction{}, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.request)
			}

			var b bytes.Buffer
			url := fmt.Sprintf("/api/v1/transaction/status-count/?threshold=%v", tc.request.Threshold)
			req := httptest.NewRequest(http.MethodGet, url, &b)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.wantCode, resp.StatusCode)
			require.Equal(t, tc.wantRes, string(body))
		})
	}
}
