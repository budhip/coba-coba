package wallettrx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_createWalletTransaction(t *testing.T) {
	testHelper := walletTrxTestHelper(t)
	netAmountValue := decimal.NewFromFloat(10)

	tests := []struct {
		name     string
		wantRes  string
		wantCode int
		doMock   func(request models.CreateWalletTransactionRequest)
	}{
		{
			name:     "happy path",
			wantRes:  `{"kind":"walletTransaction","id":"ID1","status":"PENDING","accountNumber":"111","refNumber":"222","transactionType":"333","transactionFlow":"transfer","transactionTime":"2024-04-16T16:32:34+07:00","netAmount":{"value":10,"currency":""},"amounts":null,"destinationAccountNumber":"","description":"","metadata":null}`,
			wantCode: 201,
			doMock: func(request models.CreateWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					CreateTransaction(
						gomock.Any(),
						gomock.AssignableToTypeOf(request),
					).
					Return(&models.WalletTransaction{ID: "ID1", Status: "PENDING"}, nil)
			},
		},
		{
			name:     "failed - validation error",
			wantRes:  `{"status":"error","code":400,"message":"validation"}`,
			wantCode: 400,
			doMock: func(request models.CreateWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					CreateTransaction(
						gomock.Any(),
						gomock.AssignableToTypeOf(request),
					).
					Return(nil, errors.New("validation"))
			},
		},
		{
			name:     "failed - service error",
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			wantCode: 500,
			doMock: func(request models.CreateWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					CreateTransaction(
						gomock.Any(),
						gomock.AssignableToTypeOf(request),
					).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			reqPayload := models.CreateWalletTransactionRequest{
				AccountNumber:   "111",
				RefNumber:       "222",
				TransactionType: "333",
				TransactionFlow: "transfer",
				TransactionTime: "2024-04-16T16:32:34+07:00",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(netAmountValue),
				},
			}
			if tt.doMock != nil {
				tt.doMock(reqPayload)
			}

			var b bytes.Buffer
			errEncode := json.NewEncoder(&b).Encode(reqPayload)
			require.NoError(t, errEncode)

			req := httptest.NewRequest("POST", "/api/v1/wallet-transactions", &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			req.Header.Set("X-Idempotency-Key", "0f472815-8b37-4057-a594-a5617c91589d")

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.wantCode, resp.StatusCode)
			require.Equal(t, tt.wantRes, string(body))
		})
	}
}

func Test_Handler_updateStatusWalletTransaction(t *testing.T) {
	testHelper := walletTrxTestHelper(t)

	tests := []struct {
		name     string
		request  models.UpdateStatusWalletTransactionRequest
		wantRes  string
		wantCode int
		doMock   func(request models.UpdateStatusWalletTransactionRequest)
	}{
		{
			name: "success commit",
			request: models.UpdateStatusWalletTransactionRequest{
				Action: "commit",
			},
			wantRes:  `{"kind":"walletTransaction","transactionId":"TransactionId","status":"SUCCESS"}`,
			wantCode: 200,
			doMock: func(request models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					ProcessReservedTransaction(gomock.Any(), gomock.Any()).
					Return(&models.WalletTransaction{ID: request.TransactionId, Status: "SUCCESS"}, nil)
			},
		},
		{
			name: "success cancel",
			request: models.UpdateStatusWalletTransactionRequest{
				Action: "cancel",
			},
			wantRes:  `{"kind":"walletTransaction","transactionId":"TransactionId","status":"CANCEL"}`,
			wantCode: 200,
			doMock: func(request models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					ProcessReservedTransaction(gomock.Any(), gomock.Any()).
					Return(&models.WalletTransaction{ID: request.TransactionId, Status: "CANCEL"}, nil)
			},
		},
		{
			name: "failed - validation error",
			request: models.UpdateStatusWalletTransactionRequest{
				Action: "non_existent_action",
			},
			wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_VALUES","field":"action","message":"action must be commit or cancel"}]}`,
			wantCode: 422,
		},
		{
			name: "failed - service error",
			request: models.UpdateStatusWalletTransactionRequest{
				Action: "commit",
			},
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			wantCode: 500,
			doMock: func(request models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					ProcessReservedTransaction(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
		},
		{
			name: "failed - transaction is not in a pending state",
			request: models.UpdateStatusWalletTransactionRequest{
				Action: "commit",
			},
			wantRes:  `{"status":"error","code":409,"message":"transaction status not reserved"}`,
			wantCode: 409,
			doMock: func(request models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletService.EXPECT().
					ProcessReservedTransaction(gomock.Any(), gomock.Any()).
					Return(nil, common.ErrTransactionNotReserved)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.request.TransactionId = "TransactionId"
			if tt.doMock != nil {
				tt.doMock(tt.request)
			}

			var b bytes.Buffer
			errEncode := json.NewEncoder(&b).Encode(tt.request)
			require.NoError(t, errEncode)

			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/wallet-transactions/%s", tt.request.TransactionId), &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.wantCode, resp.StatusCode)
			require.Equal(t, tt.wantRes, string(body))
		})
	}
}

type testWalletTrxHelper struct {
	router              *fiber.App
	mockCtrl            *gomock.Controller
	mockWalletService   *mock.MockWalletTrxService
	mockAccountService  *mock.MockAccountService
	mockCacheRepository *mockRepo.MockCacheRepository
}

func walletTrxTestHelper(t *testing.T) testWalletTrxHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockWalletSvc := mock.NewMockWalletTrxService(mockCtrl)
	mockAccountSvc := mock.NewMockAccountService(mockCtrl)
	mockCacheRepo := mockRepo.NewMockCacheRepository(mockCtrl)
	mockDlqProcessorService := mock.NewMockDLQProcessorService(mockCtrl)

	cfg := config.Config{}

	m := middleware.NewMiddleware(cfg, mockCacheRepo, mockDlqProcessorService)

	app := fiber.New()
	v1Group := app.Group("/api/v1")
	New(cfg, v1Group, mockWalletSvc, mockAccountSvc, m)

	return testWalletTrxHelper{
		router:              app,
		mockCtrl:            mockCtrl,
		mockWalletService:   mockWalletSvc,
		mockAccountService:  mockAccountSvc,
		mockCacheRepository: mockCacheRepo,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
