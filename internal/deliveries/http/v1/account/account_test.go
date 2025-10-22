package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_getAllAccount(t *testing.T) {
	testHelper := accountTestHelper(t)
	timeNow := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)

	type args struct {
		queryURL string
	}
	type Expectation struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name        string
		args        args
		expectation Expectation
		doMock      func(args args)
	}{
		{
			name: "success",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"account","ownerId":"c","accountNumber":"b","accountName":"a","currency":"g","availableBalance":"0","pendingBalance":"0","actualBalance":"0","status":"h","features":null,"createdAt":"2023-01-10 07:00:00","updatedAt":"2023-01-10 07:00:00"}],"pagination":{"prev":"","next":"","totalEntries":1}}`,
				wantCode: 200,
			},
			doMock: func(args args) {
				testHelper.mockAccountService.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return([]models.GetAccountOut{{
						ID:            1,
						AccountName:   "a",
						AccountNumber: "b",
						OwnerID:       "c",
						Category:      "d",
						SubCategory:   "e",
						Entity:        "f",
						Currency:      "g",
						Status:        "h",
						Balance:       models.Balance{},
						IsHVT:         false,
						CreatedAt:     timeNow,
						UpdatedAt:     timeNow,
					}}, 1, nil)
			},
		},
		{
			name: "error limit not int",
			args: args{
				queryURL: "?limit=invalid_string_here",
			},
			expectation: Expectation{
				wantRes:  `{"status":"error","code":400,"message":"strconv.ParseInt: parsing \"invalid_string_here\": invalid syntax"}`,
				wantCode: 400,
			},
		},
		{
			name: "error",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args) {
				testHelper.mockAccountService.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return([]models.GetAccountOut{}, 0, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", "/api/v1/accounts", tt.args.queryURL), nil)
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

func Test_Handler_createAccount(t *testing.T) {
	testHelper := accountTestHelper(t)

	type args struct {
		ctx context.Context
		req models.DoCreateAccountRequest
	}
	mockCreateAccountOut := models.CreateAccount{
		AccountNumber:   "21100100000001",
		OwnerID:         "12345",
		CategoryCode:    "211",
		SubCategoryCode: "10000",
		EntityCode:      "001",
		Currency:        "IDR",
		Status:          common.AccountStatusActive,
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
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx: context.Background(),
				req: models.DoCreateAccountRequest{
					AccountNumber:   "21100100000001",
					Name:            "John",
					OwnerID:         "12345",
					CategoryCode:    "211",
					SubCategoryCode: "12345",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          "active",
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"account","accountNumber":"21100100000001","name":"","ownerId":"12345","categoryCode":"211","subCategoryCode":"10000","entityCode":"001","currency":"IDR","altId":"","legacyId":null,"status":"active"}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().Create(args.ctx, models.CreateAccount{
					AccountNumber:   args.req.AccountNumber,
					Name:            args.req.Name,
					OwnerID:         args.req.OwnerID,
					CategoryCode:    args.req.CategoryCode,
					SubCategoryCode: args.req.SubCategoryCode,
					EntityCode:      args.req.EntityCode,
					Currency:        args.req.Currency,
					Status:          args.req.Status,
				}).Return(mockCreateAccountOut, nil)
			},
		},
		{
			name:      "error validating required",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx: context.Background(),
				req: models.DoCreateAccountRequest{},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"MISSING_FIELD","field":"accountNumber","message":"field is missing"},{"code":"MISSING_FIELD","field":"name","message":"field is missing"},{"code":"MISSING_FIELD","field":"ownerId","message":"field is missing"},{"code":"MISSING_FIELD","field":"categoryCode","message":"field is missing"},{"code":"MISSING_FIELD","field":"subCategoryCode","message":"field is missing"},{"code":"MISSING_FIELD","field":"entityCode","message":"field is missing"},{"code":"MISSING_FIELD","field":"currency","message":"field is missing"},{"code":"MISSING_FIELD","field":"status","message":"field is missing"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "error validating CategoryCode numeric",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx: context.Background(),
				req: models.DoCreateAccountRequest{
					AccountNumber:   "21100100000001",
					Name:            "John",
					OwnerID:         "12345",
					CategoryCode:    "21a",
					SubCategoryCode: "10000",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          "active",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_VALUES","field":"categoryCode","message":"field can only contain numeric values"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "error validating CategoryCode min",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx: context.Background(),
				req: models.DoCreateAccountRequest{
					AccountNumber:   "21100100000001",
					Name:            "John",
					OwnerID:         "12345",
					CategoryCode:    "21",
					SubCategoryCode: "10000",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          "active",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_LENGTH","field":"categoryCode","message":"field must be at least 3 characters"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "error validating CategoryCode max",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx: context.Background(),
				req: models.DoCreateAccountRequest{
					AccountNumber:   "21100100000001",
					Name:            "John",
					OwnerID:         "12345",
					CategoryCode:    "2111",
					SubCategoryCode: "10000",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          "active",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_LENGTH","field":"categoryCode","message":"field can have a maximum length of 3 characters"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "error internal server error",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx: context.Background(),
				req: models.DoCreateAccountRequest{
					AccountNumber:   "21100100000001",
					Name:            "John",
					OwnerID:         "12345",
					CategoryCode:    "211",
					SubCategoryCode: "10000",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          "active",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"internal server error"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().Create(args.ctx, models.CreateAccount{
					AccountNumber:   args.req.AccountNumber,
					Name:            args.req.Name,
					OwnerID:         args.req.OwnerID,
					CategoryCode:    args.req.CategoryCode,
					SubCategoryCode: args.req.SubCategoryCode,
					EntityCode:      args.req.EntityCode,
					Currency:        args.req.Currency,
					Status:          args.req.Status,
				}).Return(mockCreateAccountOut, common.ErrInternalServerError)
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

func Test_Handler_getOneAccount(t *testing.T) {
	testHelper := accountTestHelper(t)

	accountNumber := "[TEST]"

	type args struct {
		ctx           context.Context
		accountNumber string
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
			name:      "error - internal service error",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().GetOneByAccountNumberOrLegacyId(args.ctx, args.accountNumber).Return(models.GetAccountOut{}, assert.AnError)
			},
		},
		{
			name:      "error - not found",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":404,"message":"data not found"}`,
				wantCode: 404,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().GetOneByAccountNumberOrLegacyId(args.ctx, args.accountNumber).Return(models.GetAccountOut{}, common.ErrDataNotFound)
			},
		},
		{
			name:      "success",
			urlCalled: "/api/v1/accounts",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{
				wantRes:  `{"kind":"account","ownerId":"","accountNumber":"","accountName":"","currency":"","availableBalance":"0","pendingBalance":"0","actualBalance":"0","status":"","features":null,"createdAt":"0001-01-01 07:07:12","updatedAt":"0001-01-01 07:07:12"}`,
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().GetOneByAccountNumberOrLegacyId(args.ctx, args.accountNumber).Return(models.GetAccountOut{}, nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", tt.urlCalled, tt.args.accountNumber), nil)
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

func Test_Handler_getTotalBalance(t *testing.T) {
	testHelper := accountTestHelper(t)
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
			name: "happy path",
			expectation: Expectation{
				wantRes:  `{"kind":"account","totalBalance":"100"}`,
				wantCode: 200,
			},
			doMock: func() {
				totalBalance := decimal.NewFromFloat(100)
				testHelper.mockAccountService.EXPECT().GetTotalBalance(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(&totalBalance, nil)
			},
		},
		{
			name: "failed - error service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockAccountService.EXPECT().GetTotalBalance(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/balances", &b)
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

func Test_Handler_getAccountBalance(t *testing.T) {
	testHelper := accountTestHelper(t)
	accountNumber := "123456"
	type args struct {
		ctx           context.Context
		accountNumber string
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
			name:      "error - internal service error",
			urlCalled: "/api/v1/accounts/123456/balances",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockBalanceService.EXPECT().Get(args.ctx, args.accountNumber).Return(models.AccountBalance{}, assert.AnError)
			},
		},
		{
			name:      "error - not found",
			urlCalled: "/api/v1/accounts/123456/balances",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":404,"message":"data not found"}`,
				wantCode: 404,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockBalanceService.EXPECT().Get(args.ctx, args.accountNumber).Return(models.AccountBalance{}, common.ErrDataNotFound)
			},
		},
		{
			name:      "success",
			urlCalled: "/api/v1/accounts/123456/balances",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{
				wantRes:  `{"kind":"accountBalance","accountNumber":"[TEST]","currency":"IDR","actualBalance":"420.69","pendingBalance":"20.1","availableBalance":"400.59","lastUpdatedAt":"2024-01-23T07:01:02+07:00"}`,
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
				updatedAt, err := time.Parse(common.DateFormatYYYYMMDDWithTime, "2024-01-23 00:01:02")
				assert.NoError(t, err)

				testHelper.mockBalanceService.EXPECT().
					Get(args.ctx, args.accountNumber).
					Return(models.AccountBalance{
						AccountNumber: "[TEST]",
						Balance: models.NewBalance(
							decimal.NewFromFloat(420.69),
							decimal.NewFromFloat(20.10),
							models.WithLastUpdatedAt(updatedAt)),
					}, nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			req := httptest.NewRequest(http.MethodGet, tt.urlCalled, nil)
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

func Test_Handler_createAccountFeature(t *testing.T) {
	testHelper := accountTestHelper(t)
	validPresetCustomer := "customer"
	validModelRequest := models.CreateWalletReq{
		AccountNumber: "40000133919",
		Features: models.WalletFeatureReq{
			Preset: validPresetCustomer,
		},
	}
	validModelPayload, _ := validModelRequest.TransformAndValidate()
	validModelResponse := models.WalletOut{
		AccountNumber: "40000133919",
		Feature: &models.WalletFeature{
			Preset: &validPresetCustomer,
		},
	}

	type args struct {
		ctx context.Context
		req models.CreateWalletReq
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
			urlCalled: "/api/v1/accounts/40000133919/features",
			args: args{
				ctx: context.Background(),
				req: validModelRequest,
			},
			mockData: mockData{
				wantRes:  `{"kind":"accountFeature","accountNumber":"40000133919","features":{"preset":"customer","allowedNegativeBalance":null,"balanceRangeMin":null,"negativeBalanceLimit":null}}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockWalletAccountService.EXPECT().CreateAccountFeature(args.ctx, validModelPayload).Return(&validModelResponse, nil)
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
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			//body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
			//require.Equal(t, tt.mockData.wantRes, strings.TrimSuffix(string(body), "\n"))
		})
	}
}

func Test_Handler_updateAccountBySubCategory(t *testing.T) {
	testHelper := accountTestHelper(t)

	type args struct {
		ctx context.Context
		req models.UpdateAccountBySubCategoryRequest
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
			name:      "error - internal service error",
			urlCalled: "/api/v1/accounts/sub-category/10000",
			args: args{
				ctx: context.Background(),
				req: models.UpdateAccountBySubCategoryRequest{
					ProductTypeName: &[]string{"test"}[0],
					Currency:        &[]string{"IDR"}[0],
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().UpdateBySubCategory(args.ctx, gomock.Any()).Return(assert.AnError)
			},
		},
		{
			name:      "success",
			urlCalled: "/api/v1/accounts/sub-category/10000",
			args: args{
				ctx: context.Background(),
				req: models.UpdateAccountBySubCategoryRequest{
					Code:            "10000",
					ProductTypeName: &[]string{"test"}[0],
					Currency:        &[]string{"IDR"}[0],
				},
			},
			mockData: mockData{
				wantRes:  "null",
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().UpdateBySubCategory(args.ctx, gomock.Any()).Return(nil)
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

			req := httptest.NewRequest(http.MethodPatch, tt.urlCalled, &b)
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

func Test_Handler_deleteAccount(t *testing.T) {
	testHelper := accountTestHelper(t)

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
		args      args
		mockData  mockData
		doMock    func(args args, mockData mockData)
	}{
		{
			name:      "error - internal service error",
			urlCalled: "/api/v1/accounts/123456",
			args: args{
				ctx: context.Background(),
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().
					Delete(args.ctx, "123456").
					Return(assert.AnError)
			},
		},
		{
			name:      "success",
			urlCalled: "/api/v1/accounts/123456",
			args: args{
				ctx: context.Background(),
			},
			mockData: mockData{
				wantRes:  "null",
				wantCode: 204,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccountService.EXPECT().
					Delete(args.ctx, "123456").
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

			req := httptest.NewRequest(http.MethodDelete, tt.urlCalled, nil)
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
