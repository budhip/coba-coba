package internalwallet

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_listTransactionByAccountNumber(t *testing.T) {
	testHelper := internalWalletTestHelper(t)
	timeVal := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)

	type Expectation struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name        string
		expectation Expectation
		doMock      func()
	}{
		{
			name: "success",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"walletTransaction","transactionDate":"2023-01-10","transactionTime":"2023-01-10T07:00:00+07:00","transactionType":"6","amount":{"value":10,"currency":"IDR"},"amounts":[{"type":"TUPVA","amount":{"value":20,"currency":""}}],"status":"2","transactionWalletId":"1","refNumber":"5","description":"","transactionFlow":"7","metadata":{"abc":"def"}}],"pagination":{"prev":"","next":"","totalEntries":1}}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).
					Return([]models.WalletTransaction{{
						ID:                       "1",
						Status:                   "2",
						AccountNumber:            "3",
						DestinationAccountNumber: "4",
						RefNumber:                "5",
						TransactionType:          "6",
						TransactionTime:          timeVal,
						TransactionFlow:          "7",
						NetAmount: models.Amount{
							ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(10)),
						},
						Amounts: models.Amounts{models.AmountDetail{
							Type: "TUPVA",
							Amount: &models.Amount{
								ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(20)),
							},
						}},
						CreatedAt: timeVal,
						Metadata:  models.WalletMetadata{"abc": "def"},
					}}, 1, nil)
			},
		},
		{
			name: "error service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return([]models.WalletTransaction{}, 0, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/internal-wallets/accounts/%s/transactions", "accountNumber"), nil)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tt.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_listTransactionByAccountNumbers(t *testing.T) {
	testHelper := internalWalletTestHelper(t)
	timeVal := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)

	type Expectation struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name        string
		expectation Expectation
		doMock      func()
	}{
		{
			name: "success",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"walletTransaction","transactionDate":"2023-01-10","transactionTime":"2023-01-10T07:00:00+07:00","transactionType":"6","amount":{"value":10,"currency":"IDR"},"amounts":[{"type":"TUPVA","amount":{"value":20,"currency":""}}],"status":"2","transactionWalletId":"1","refNumber":"5","description":"","transactionFlow":"7","metadata":{"abc":"def"}}],"pagination":{"prev":"","next":"","totalEntries":1}}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).
					Return([]models.WalletTransaction{{
						ID:                       "1",
						Status:                   "2",
						AccountNumber:            "3",
						DestinationAccountNumber: "4",
						RefNumber:                "5",
						TransactionType:          "6",
						TransactionTime:          timeVal,
						TransactionFlow:          "7",
						NetAmount: models.Amount{
							ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(10)),
						},
						Amounts: models.Amounts{models.AmountDetail{
							Type: "TUPVA",
							Amount: &models.Amount{
								ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(20)),
							},
						}},
						CreatedAt: timeVal,
						Metadata:  models.WalletMetadata{"abc": "def"},
					}}, 1, nil)
			},
		},
		{
			name: "error service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return([]models.WalletTransaction{}, 0, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/internal-wallets/accounts/transactions%s", "?accountNumberList=1122,2233,4455"), nil)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tt.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

type testInternalWalletHelper struct {
	router      *echo.Echo
	mockCtrl    *gomock.Controller
	mockService *mock.MockWalletTrxService
}

func internalWalletTestHelper(t *testing.T) testInternalWalletHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockWalletTrxService(mockCtrl)

	app := echo.New()
	app.Pre(echomiddleware.RemoveTrailingSlash())
	v1Group := app.Group("/api/v1")
	New(v1Group, mockSvc)

	return testInternalWalletHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
