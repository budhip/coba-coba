package files

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

func Test_Handler_uploadFile(t *testing.T) {
	testHelper := filesTestHelper(t)

	var requestBody bytes.Buffer

	type Expectation struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name        string
		expectation Expectation
		writer      *multipart.Writer
		doMock      func(writer *multipart.Writer)
	}{
		{
			name: "success",
			expectation: Expectation{
				wantRes:  `{"kind":"file","file":"test.csv","status":"processing"}`,
				wantCode: 200,
			},
			writer: multipart.NewWriter(&requestBody),
			doMock: func(writer *multipart.Writer) {
				// Add a CSV file to the request
				fileWriter, _ := writer.CreateFormFile("files", "test.csv")
				fileWriter.Write([]byte("csv content here"))

				// Close the multipart writer to finalize the request body
				writer.Close()

				testHelper.mockService.EXPECT().Upload(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "failed - missing file",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":"MISSING_FIELD","message":"field is missing caused by files can not empty"}`,
				wantCode: 400,
			},
			writer: multipart.NewWriter(&requestBody),
			doMock: func(writer *multipart.Writer) {},
		},
		{
			name: "failed - invalid extension",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":"INVALID_VALUES","message":"invalid format file caused by files must be .csv"}`,
				wantCode: 400,
			},
			writer: multipart.NewWriter(&requestBody),
			doMock: func(writer *multipart.Writer) {
				// Add a CSV file to the request
				fileWriter, _ := writer.CreateFormFile("files", "test.json")
				fileWriter.Write([]byte("csv content here"))

				// Close the multipart writer to finalize the request body
				writer.Close()
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.writer)
			}

			// Create a test request with the multipart form-data body
			req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &requestBody)
			req.Header.Set("Content-Type", tc.writer.FormDataContentType())

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tc.expectation.wantRes, strings.TrimSuffix(string(respBody), "\n"))
			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
		})
	}
}

type filesHandlerTestHelper struct {
	router      *echo.Echo
	mockCtrl    *gomock.Controller
	mockService *mock.MockFileService
}

func filesTestHelper(t *testing.T) filesHandlerTestHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockFileService(mockCtrl)

	app := echo.New()
	app.Pre(echomiddleware.RemoveTrailingSlash())
	v1Group := app.Group("/api/v1")
	New(v1Group, mockSvc)

	return filesHandlerTestHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}
