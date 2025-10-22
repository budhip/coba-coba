package masterdata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
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

func Test_Handler_createOrderType(t *testing.T) {
	testHelper := masterDataTestHelper(t)

	type args struct {
		ctx context.Context
		req models.OrderType
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				req: models.OrderType{
					OrderTypeCode: "001",
					OrderTypeName: "TOPUP LENDER P2P",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "001001",
							TransactionTypeName: "TOPUP Mandiri VA",
						},
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"orderType","orderTypeCode":"001","orderTypeName":"TOPUP LENDER P2P","transactionTypes":[{"kind":"transactionType","transactionTypeCode":"001001","transactionTypeName":"TOPUP Mandiri VA"}]}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().CreateOrderType(args.ctx, args.req).Return(nil)
			},
		},
		{
			name: "test error",
			args: args{
				ctx: context.Background(),
				req: models.OrderType{
					OrderTypeCode: "001",
					OrderTypeName: "TOPUP LENDER P2P",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "001001",
							TransactionTypeName: "TOPUP Mandiri VA",
						},
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().CreateOrderType(args.ctx, args.req).Return(assert.AnError)
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

			req := httptest.NewRequest(http.MethodPost, "/api/v1/order-types", &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_updateOrderType(t *testing.T) {
	testHelper := masterDataTestHelper(t)

	type args struct {
		ctx context.Context
		req models.OrderType
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				req: models.OrderType{
					OrderTypeCode: "001",
					OrderTypeName: "TOPUP LENDER P2P",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "001001",
							TransactionTypeName: "TOPUP Mandiri VA",
						},
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"orderType","orderTypeCode":"001","orderTypeName":"TOPUP LENDER P2P","transactionTypes":[{"kind":"transactionType","transactionTypeCode":"001001","transactionTypeName":"TOPUP Mandiri VA"}]}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().UpdateOrderType(args.ctx, args.req).Return(nil)
			},
		},
		{
			name: "error data not found",
			args: args{
				ctx: context.Background(),
				req: models.OrderType{
					OrderTypeCode: "001XXXX",
					OrderTypeName: "TOPUP LENDER P2P",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "001001",
							TransactionTypeName: "TOPUP Mandiri VA",
						},
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":404,"message":"data not found"}`,
				wantCode: 404,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().UpdateOrderType(args.ctx, args.req).Return(common.ErrDataNotFound)
			},
		},
		{
			name: "test error",
			args: args{
				ctx: context.Background(),
				req: models.OrderType{
					OrderTypeCode: "001",
					OrderTypeName: "TOPUP LENDER P2P",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "001001",
							TransactionTypeName: "TOPUP Mandiri VA",
						},
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().UpdateOrderType(args.ctx, args.req).Return(assert.AnError)
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

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/order-types", &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_getAllOrderType(t *testing.T) {
	testHelper := masterDataTestHelper(t)

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
			name: "success get all order types",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"orderType","orderTypeCode":"1002","orderTypeName":"Cashout Lender P2P","transactionTypes":[{"kind":"transactionType","transactionTypeCode":"1002001","transactionTypeName":"Request Cashout"},{"kind":"transactionType","transactionTypeCode":"1002002","transactionTypeName":"Reject Request Cashout"}]}],"total_rows":1}`,
				wantCode: 200,
			},
			doMock: func() {
				res := []models.OrderType{
					{
						OrderTypeCode: "1002",
						OrderTypeName: "Cashout Lender P2P",
						TransactionTypes: []models.TransactionType{
							{
								TransactionTypeCode: "1002001",
								TransactionTypeName: "Request Cashout",
							},
							{
								TransactionTypeCode: "1002002",
								TransactionTypeName: "Reject Request Cashout",
							},
						},
					},
				}

				testHelper.mockService.EXPECT().GetAllOrderType(gomock.Any(), gomock.Any()).Return(res, nil)
			},
		},
		{
			name: "failed to get data order type",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetAllOrderType(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/order-types", &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tc.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_getAllTransactionType(t *testing.T) {
	testHelper := masterDataTestHelper(t)

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
			name: "success get all transaction types",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"transactionType","transactionTypeCode":"1002001","transactionTypeName":"Request Cashout"},{"kind":"transactionType","transactionTypeCode":"1002002","transactionTypeName":"Reject Request Cashout"}],"total_rows":2}`,
				wantCode: 200,
			},
			doMock: func() {
				res := []models.TransactionType{
					{
						TransactionTypeCode: "1002001",
						TransactionTypeName: "Request Cashout",
					},
					{
						TransactionTypeCode: "1002002",
						TransactionTypeName: "Reject Request Cashout",
					},
				}

				testHelper.mockService.EXPECT().GetAllTransactionType(gomock.Any(), gomock.Any()).Return(res, nil)
			},
		},
		{
			name: "failed to get data transaction type",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetAllTransactionType(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/transaction-types", &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tc.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_getOrderType(t *testing.T) {
	testHelper := masterDataTestHelper(t)
	orderTypeCode := "TEST"

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
				wantRes:  `{"kind":"orderType","orderTypeCode":"1002","orderTypeName":"Cashout Lender P2P","transactionTypes":[{"kind":"transactionType","transactionTypeCode":"1002001","transactionTypeName":"Request Cashout"},{"kind":"transactionType","transactionTypeCode":"1002002","transactionTypeName":"Reject Request Cashout"}]}`,
				wantCode: 200,
			},
			doMock: func() {
				res := models.OrderType{
					OrderTypeCode: "1002",
					OrderTypeName: "Cashout Lender P2P",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "1002001",
							TransactionTypeName: "Request Cashout",
						},
						{
							TransactionTypeCode: "1002002",
							TransactionTypeName: "Reject Request Cashout",
						},
					},
				}

				testHelper.mockService.EXPECT().
					GetOneOrderType(gomock.AssignableToTypeOf(context.Background()), orderTypeCode).Return(&res, nil)
			},
		},
		{
			name: "failed - not found",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":404,"message":"data not found"}`,
				wantCode: 404,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetOneOrderType(gomock.AssignableToTypeOf(context.Background()), orderTypeCode).
					Return(nil, common.ErrDataNotFound)
			},
		},
		{
			name: "failed - err service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetOneOrderType(gomock.AssignableToTypeOf(context.Background()), orderTypeCode).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, fmt.Sprint("/api/v1/order-types/", orderTypeCode), &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tc.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_getTransactionType(t *testing.T) {
	testHelper := masterDataTestHelper(t)
	trxTypeCode := "TEST"

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
				wantRes:  `{"kind":"transactionType","transactionTypeCode":"1002001","transactionTypeName":"Request Cashout"}`,
				wantCode: 200,
			},
			doMock: func() {
				res := models.TransactionType{
					TransactionTypeCode: "1002001",
					TransactionTypeName: "Request Cashout",
				}

				testHelper.mockService.EXPECT().
					GetOneTransactionType(gomock.AssignableToTypeOf(context.Background()), trxTypeCode).Return(&res, nil)
			},
		},
		{
			name: "failed - not found",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":404,"message":"data not found"}`,
				wantCode: 404,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetOneTransactionType(gomock.AssignableToTypeOf(context.Background()), trxTypeCode).
					Return(nil, common.ErrDataNotFound)
			},
		},
		{
			name: "failed - err service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetOneTransactionType(gomock.AssignableToTypeOf(context.Background()), trxTypeCode).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, fmt.Sprint("/api/v1/transaction-types/", trxTypeCode), &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tc.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_getAllVatConfig(t *testing.T) {
	testHelper := masterDataTestHelper(t)

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
				wantRes:  `{"kind":"collection","contents":[{"kind":"configVatRevenue","percentage":"0.11","startTime":"0001-01-01T00:00:00Z","endTime":"0001-01-01T00:00:00Z"}],"total_rows":1}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetAllVATConfig(gomock.AssignableToTypeOf(context.Background())).
					Return([]models.ConfigVatRevenue{
						{
							Percentage: decimal.NewFromFloat(0.11),
							StartTime:  time.Time{},
							EndTime:    time.Time{},
						},
					}, nil)
			},
		},
		{
			name: "failed - err service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetAllVATConfig(gomock.AssignableToTypeOf(context.Background())).
					Return(nil, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, fmt.Sprint("/api/v1/vat-configs/"), &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
			require.Equal(t, tc.expectation.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_upsertVatConfig(t *testing.T) {
	testHelper := masterDataTestHelper(t)

	type args struct {
		ctx context.Context
		req []models.ConfigVatRevenue
	}
	type mockData struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				req: []models.ConfigVatRevenue{
					{
						Percentage: decimal.NewFromFloat(0.11),
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"collection","contents":[{"kind":"configVatRevenue","percentage":"0.11","startTime":"0001-01-01T00:00:00Z","endTime":"0001-01-01T00:00:00Z"}],"total_rows":1}`,
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().
					UpsertVATConfig(args.ctx, args.req).
					Return(nil)
			},
		},
		{
			name: "test error",
			args: args{
				ctx: context.Background(),
				req: []models.ConfigVatRevenue{
					{
						Percentage: decimal.NewFromFloat(0.11),
					},
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().
					UpsertVATConfig(args.ctx, args.req).
					Return(assert.AnError)
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

			req := httptest.NewRequest(http.MethodPatch, fmt.Sprint("/api/v1/vat-configs/"), &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			require.Equal(t, tt.mockData.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

type testMasterDataHelper struct {
	router      *echo.Echo
	mockCtrl    *gomock.Controller
	mockService *mock.MockMasterDataService
}

func masterDataTestHelper(t *testing.T) testMasterDataHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockMasterDataService(mockCtrl)

	app := echo.New()

	v1Group := app.Group("/api/v1")
	app.Pre(echomiddleware.RemoveTrailingSlash())
	New(v1Group, mockSvc)

	return testMasterDataHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
