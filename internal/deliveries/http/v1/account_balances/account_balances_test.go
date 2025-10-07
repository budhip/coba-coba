package account_balances

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_getAccountBalance(t *testing.T) {
	testHelper := balanceTestHelper(t)
	ct := time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC)

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
			name: "success get account balance",
			expectation: Expectation{
				wantRes:  `{"kind":"accountBalance","accountNumber":"1234567","currency":"IDR","actualBalance":"10000","pendingBalance":"0","availableBalance":"10000","lastUpdatedAt":"2025-04-20T07:00:00+07:00"}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.
					EXPECT().
					Get(gomock.AssignableToTypeOf(context.Background()), "1234567").
					Return(models.AccountBalance{
						AccountNumber: "1234567",
						Balance: models.NewBalance(
							decimal.NewFromInt(10_000),
							decimal.Zero,
							models.WithLastUpdatedAt(ct)),
					}, nil)
			},
		},
		{
			name: "failed to get data",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.
					EXPECT().
					Get(gomock.AssignableToTypeOf(context.Background()), "1234567").
					Return(models.AccountBalance{}, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/account-balances/1234567", &b)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tc.expectation.wantRes, string(body))
		})
	}
}

type testBaanceHelper struct {
	router      *fiber.App
	mockCtrl    *gomock.Controller
	mockService *mock.MockBalanceService
}

func balanceTestHelper(t *testing.T) testBaanceHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockBalanceService(mockCtrl)

	app := fiber.New()
	v1Group := app.Group("/api/v1")
	New(v1Group, mockSvc)

	return testBaanceHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
