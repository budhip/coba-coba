package entity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_createEntity(t *testing.T) {
	testHelper := entityTestHelper(t)

	type args struct {
		ctx context.Context
		req models.CreateEntityRequest
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
			urlCalled: "/api/v1/entities",
			args: args{
				ctx: context.Background(),
				req: models.CreateEntityRequest{
					Code:        "001",
					Name:        "test",
					Description: "",
				},
			},
			mockData: mockData{
				wantRes:  `{"kind":"entity","code":"001","name":"test","description":""}`,
				wantCode: 201,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateEntityIn(args.req)).Return(&models.Entity{
					Code: args.req.Code,
					Name: args.req.Name,
				}, nil)
			},
		},
		{
			name:      "test error validating request",
			urlCalled: "/api/v1/entities",
			args: args{
				ctx: context.Background(),
				req: models.CreateEntityRequest{
					Code: "12",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_LENGTH","field":"code","message":"field must be at least 3 characters"},{"code":"MISSING_FIELD","field":"name","message":"field is missing"}]}`,
				wantCode: 422,
			},
		},
		{
			name:      "test error",
			urlCalled: "/api/v1/entities",
			args: args{
				ctx: context.Background(),
				req: models.CreateEntityRequest{
					Code:        "001",
					Name:        "test",
					Description: "",
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockService.EXPECT().Create(args.ctx, models.CreateEntityIn(args.req)).Return(&models.Entity{}, assert.AnError)
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

func Test_Handler_getAllEntity(t *testing.T) {
	testHelper := entityTestHelper(t)

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
			name: "success get all entity",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[{"kind":"entity","code":"666","name":"ENT","description":"ini entity"}],"total_rows":1}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.Entity{{
						Code:        "666",
						Name:        "ENT",
						Description: "ini entity",
					}}, nil)
			},
		},
		{
			name: "failed to get data entity",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().
					GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.Entity{}, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/entities", &b)
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

type testEntityHelper struct {
	router      *fiber.App
	mockCtrl    *gomock.Controller
	mockService *mock.MockEntityService
}

func entityTestHelper(t *testing.T) testEntityHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockEntityService(mockCtrl)

	app := fiber.New()
	v1Group := app.Group("/api/v1")
	New(v1Group, mockSvc)

	return testEntityHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
