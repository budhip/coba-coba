package subcategory

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

func Test_Handler_createSubCategory(t *testing.T) {
	testHelper := subCategoryTestHelper(t)

	type args struct {
		ctx context.Context
		req models.CreateSubCategoryRequest
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
				req: models.CreateSubCategoryRequest{
					CategoryCode: "001",
					Code:         "00001",
					Name:         "test",
					Description:  "TEST DESC",
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"subCategory","categoryCode":"","code":"","name":"","description":""}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateSubCategory(args.req)).
					Return(&models.SubCategory{}, nil)
			},
		},
		{
			name: "error request validation",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategoryRequest{
					CategoryCode: "12",
					Code:         "001333",
					Name:         "",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_LENGTH","field":"categoryCode","message":"field must be at least 3 characters"},{"code":"INVALID_LENGTH","field":"code","message":"field can have a maximum length of 5 characters"},{"code":"MISSING_FIELD","field":"name","message":"field is missing"}]}`,
				wantCode: 422,
			},
		},
		{
			name: "category not found",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategoryRequest{
					CategoryCode: "001",
					Code:         "00001",
					Name:         "test",
					Description:  "TEST DESC",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":404,"message":"data not found"}`,
				wantCode: 404,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateSubCategory(args.req)).
					Return(&models.SubCategory{}, common.ErrDataNotFound)
			},
		},
		{
			name: "data exist",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategoryRequest{
					CategoryCode: "001",
					Code:         "00001",
					Name:         "test",
					Description:  "TEST DESC",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":409,"message":"data exist"}`,
				wantCode: 409,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateSubCategory(args.req)).
					Return(&models.SubCategory{}, common.ErrDataExist)
			},
		},
		{
			name: "error service",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategoryRequest{
					CategoryCode: "001",
					Code:         "00001",
					Name:         "test",
					Description:  "TEST DESC",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateSubCategory(args.req)).
					Return(&models.SubCategory{}, assert.AnError)
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

			req := httptest.NewRequest(http.MethodPost, "/api/v1/sub-categories", &b)
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

type testSubCategoryHelper struct {
	router      *echo.Echo
	mockCtrl    *gomock.Controller
	mockService *mock.MockSubCategoryService
}

func subCategoryTestHelper(t *testing.T) testSubCategoryHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockSubCategoryService(mockCtrl)

	app := echo.New()

	v1Group := app.Group("/api/v1")
	app.Pre(echomiddleware.RemoveTrailingSlash())
	New(v1Group, mockSvc)

	return testSubCategoryHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func Test_Handler_getAllSubCategory(t *testing.T) {
	testHelper := subCategoryTestHelper(t)
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
			name: "success get all sub category",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"subCategory","categoryCode":"221","code":"10000","name":"RETAIL","description":"sub category"}],"total_rows":1}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.SubCategory{{
						CategoryCode: "221",
						Code:         "10000",
						Name:         "RETAIL",
						Description:  "sub category",
					}}, nil)
			},
		},
		{
			name: "failed to get data sub category",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.SubCategory{}, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}
			var b bytes.Buffer
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sub-categories", &b)
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
