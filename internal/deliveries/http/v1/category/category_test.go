package category

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

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_createCategory(t *testing.T) {
	testHelper := categoryTestHelper(t)

	type args struct {
		ctx context.Context
		req models.CreateCategoryRequest
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
			urlCalled: "/api/v1/categories",
			args: args{
				ctx: context.Background(),
				req: models.CreateCategoryRequest{
					Code:        "001",
					Name:        "test",
					Description: "",
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"category","code":"001","name":"test","description":"","createdAt":null,"updatedAt":null}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateCategoryIn(args.req)).Return(&models.Category{
					Code: args.req.Code,
					Name: args.req.Name,
				}, nil)
			},
		},
		{
			name:      "error validating request",
			urlCalled: "/api/v1/categories",
			args: args{
				ctx: context.Background(),
				req: models.CreateCategoryRequest{
					Code: "12",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_LENGTH","field":"code","message":"field must be at least 3 characters"},{"code":"MISSING_FIELD","field":"name","message":"field is missing"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "error service",
			urlCalled: "/api/v1/categories",
			args: args{
				ctx: context.Background(),
				req: models.CreateCategoryRequest{
					Code:        "001",
					Name:        "test",
					Description: "1",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateCategoryIn(args.req)).Return(&models.Category{}, assert.AnError)
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

func Test_Handler_getAllCategory(t *testing.T) {
	testHelper := categoryTestHelper(t)

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
				wantRes:  `{"kind":"collection","contents":[{"kind":"category","code":"01","name":"tes","description":"test","createdAt":null,"updatedAt":null}],"total_rows":1}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.Category{{
						ID:          0,
						Code:        "01",
						Name:        "tes",
						Description: "test",
					}}, nil)
			},
		},
		{
			name: "error service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.Category{}, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", &b)
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

type testCategoryHelper struct {
	router      *echo.Echo
	mockCtrl    *gomock.Controller
	mockService *mock.MockCategoryService
}

func categoryTestHelper(t *testing.T) testCategoryHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockCategoryService(mockCtrl)

	app := echo.New()
	app.Pre(echomiddleware.RemoveTrailingSlash())
	v1Group := app.Group("/api/v1")
	New(v1Group, mockSvc)

	return testCategoryHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
