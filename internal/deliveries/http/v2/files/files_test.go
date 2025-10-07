package files

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/gofiber/fiber/v2"
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

				testHelper.mockService.EXPECT().UploadWalletTransaction(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
			req := httptest.NewRequest(http.MethodPost, "/api/v2/files/upload", &requestBody)
			req.Header.Set("Content-Type", tc.writer.FormDataContentType())
			req.Header.Set("X-Ngmis-Username", "test@gmail.com")
			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)
			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tc.expectation.wantRes, string(respBody))
			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
		})
	}
}

type filesHandlerTestHelper struct {
	router      *fiber.App
	mockCtrl    *gomock.Controller
	mockService *mock.MockFileService
}

func filesTestHelper(t *testing.T) filesHandlerTestHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockFileService(mockCtrl)

	app := fiber.New()
	v2Group := app.Group("/api/v2")
	New(v2Group, mockSvc)

	return filesHandlerTestHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}
