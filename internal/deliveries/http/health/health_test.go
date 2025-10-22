package health

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testHealthCheckHelper struct {
	mockCtrl *gomock.Controller
	router   *echo.Echo
}

func toolTestHealthCheckHelper(t *testing.T) testHealthCheckHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	app := echo.New()
	apiGroup := app.Group("/api")
	New(apiGroup)

	return testHealthCheckHelper{
		mockCtrl: mockCtrl,
		router:   app,
	}
}
func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

func Test_Handler_healthCheck(t *testing.T) {
	testHelper := toolTestHealthCheckHelper(t)

	type args struct{}
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
			urlCalled: "/api/health",
			args:      args{},
			mockData: mockData{
				wantRes:  `{"kind":"health","status":"server is up and running"}`,
				wantCode: 200,
			},
			doMock: func(args args, mockData mockData) {
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
