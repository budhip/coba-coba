package recontools

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
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

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

func Test_Handler_reconToolsUpload(t *testing.T) {
	testHelper := filesTestHelper(t)

	type Expectation struct {
		wantRes  string
		wantCode int
	}
	tests := []struct {
		name        string
		expectation Expectation
		requestBody *models.UploadReconFileRequest
		file        *bytes.Buffer
		fileName    string
		doMock      func()
	}{
		{
			name: "happy path",
			expectation: Expectation{
				wantRes:  `{"kind":"reconTool","message":"Processing"}`,
				wantCode: 202,
			},
			requestBody: &models.UploadReconFileRequest{
				OrderType:       "TEST",
				TransactionType: "TEST",
				TransactionDate: "2023-01-01",
			},
			file:     createTestFile(),
			fileName: "test.csv",
			doMock: func() {
				testHelper.mockService.EXPECT().UploadReconTemplate(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.AssignableToTypeOf(&models.UploadReconFileRequest{}),
				).Return(nil)
			},
		},
		{
			name: "failed - missing field",
			expectation: Expectation{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"MISSING_FIELD","field":"orderType","message":"field is missing"},{"code":"MISSING_FIELD","field":"transactionType","message":"field is missing"},{"code":"MISSING_FIELD","field":"transactionDate","message":"field is missing"},{"code":"MISSING_FIELD","field":"reconFile","message":"field is missing"}]}`,
				wantCode: 422,
			},
			requestBody: &models.UploadReconFileRequest{},
			doMock:      func() {},
		},
		{
			name: "failed - invalid date",
			expectation: Expectation{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_VALUES","field":"transactionDate","message":"invalid format date"}]}`,
				wantCode: 422,
			},
			requestBody: &models.UploadReconFileRequest{
				OrderType:       "TEST",
				TransactionType: "TEST",
				TransactionDate: "INVALID",
			},
			file:     createTestFile(),
			fileName: "test.csv",
			doMock:   func() {},
		},
		{
			name: "failed - file must csv",
			expectation: Expectation{
				wantRes:  `{"status":"error","message":"validation failed","errors":[{"code":"INVALID_VALUES","field":"reconFile","message":"file must be csv"}]}`,
				wantCode: 422,
			},
			requestBody: &models.UploadReconFileRequest{
				OrderType:       "TEST",
				TransactionType: "TEST",
				TransactionDate: "2023-01-01",
			},
			file:     createTestFile(),
			fileName: "test.txt",
			doMock:   func() {},
		},
		{
			name: "failed - err service",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":400,"message":"assert.AnError general error for testing"}`,
				wantCode: 400,
			},
			requestBody: &models.UploadReconFileRequest{
				OrderType:       "TEST",
				TransactionType: "TEST",
				TransactionDate: "2023-01-01",
			},
			file:     createTestFile(),
			fileName: "test.csv",
			doMock: func() {
				testHelper.mockService.EXPECT().UploadReconTemplate(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.AssignableToTypeOf(&models.UploadReconFileRequest{}),
				).Return(assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			req := buildRequest(t, tc.requestBody, tc.file, tc.fileName)
			resp, err := testHelper.router.Test(req)
			require.NoError(t, err)
			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tc.expectation.wantRes, string(respBody))
			require.Equal(t, tc.expectation.wantCode, resp.StatusCode)
		})
	}
}

func Test_Handler_getAllReconHistory(t *testing.T) {
	testHelper := filesTestHelper(t)

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
			name: "success get list recon history",
			expectation: Expectation{
				wantRes:  `{"kind":"collection","contents":[],"pagination":{"prev":"","next":"","totalEntries":0}}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetListReconHistory(gomock.Any(), gomock.Any()).
					Return([]models.ReconToolHistory{}, 0, nil)
			},
		},
		{
			name: "failed to get data recon history",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetListReconHistory(gomock.Any(), gomock.Any()).
					Return([]models.ReconToolHistory{}, 0, assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/recon-tools", &b)
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

func Test_Handler_getResultURLReconHistory(t *testing.T) {
	testHelper := filesTestHelper(t)

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
			name: "success get url file history",
			expectation: Expectation{
				wantRes:  `{"kind":"reconToolResultUrl","resultFileUrl":"https://my_file.txt"}`,
				wantCode: 200,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetResultFileURL(gomock.Any(), uint64(1)).
					Return("https://my_file.txt", nil)
			},
		},
		{
			name: "failed to get data entity",
			expectation: Expectation{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func() {
				testHelper.mockService.EXPECT().GetResultFileURL(gomock.Any(), uint64(1)).
					Return("", assert.AnError)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			var b bytes.Buffer

			req := httptest.NewRequest(http.MethodGet, "/api/v1/recon-tools/1/download", &b)
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

type reconToolsHandlerTestHelper struct {
	router      *fiber.App
	mockCtrl    *gomock.Controller
	mockService *mock.MockReconService
}

func filesTestHelper(t *testing.T) reconToolsHandlerTestHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSvc := mock.NewMockReconService(mockCtrl)

	app := fiber.New()
	v1Group := app.Group("/api/v1")
	New(v1Group, mockSvc)

	return reconToolsHandlerTestHelper{
		router:      app,
		mockCtrl:    mockCtrl,
		mockService: mockSvc,
	}
}

func createTestFile() *bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString("header1,header2,header3\n")
	buf.WriteString("value1,value2,value3\n")
	return &buf
}

func buildRequest(t *testing.T, requestBody *models.UploadReconFileRequest, file *bytes.Buffer, fileName string) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	addFormField := func(fieldName, value string) {
		part, err := writer.CreateFormField(fieldName)
		assert.NoError(t, err)
		_, err = part.Write([]byte(value))
		assert.NoError(t, err)
	}

	addFormField("orderType", requestBody.OrderType)
	addFormField("transactionType", requestBody.TransactionType)
	addFormField("transactionDate", requestBody.TransactionDate)

	if file != nil {
		part, err := writer.CreateFormFile("reconFile", fileName)
		assert.NoError(t, err)
		_, err = part.Write(file.Bytes())
		assert.NoError(t, err)
	}

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recon-tools/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
