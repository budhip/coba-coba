package transaction

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_Handler_downloadTransaction(t *testing.T) {
	testHelper := transactionTestHelper(t)
	type args struct {
		ctx  context.Context
		opts models.DownloadTransactionRequest
	}
	type mockData struct {
		gotData bytes.Buffer

		wantRes  string
		wantCode int
	}
	tests := []struct {
		name      string
		urlCalled string
		args      args
		mockData  mockData
		doMock    func(args *args, mockData *mockData)
	}{
		{
			name:      "success data",
			urlCalled: "/api/v1/transaction/download",
			args: args{
				ctx: context.Background(),
				opts: models.DownloadTransactionRequest{
					Options: models.TransactionFilterOptions{},
					Writer:  nil,
				},
			},
			mockData: mockData{
				wantRes:  "Transaction ID,No Ref,Order Type,Transaction Type,Transaction Date,From Account,To Account,Amount,Status\nc172ca84-9ae2-489c-ae4f-8ef372a109ae,55aa66bb-e6e0-4065-9f4a-64182e97e9d9,TOPUP,TOPUP,0001-01-01 00:00:00,189513,222000000069,0,1\n",
				wantCode: 200,
			},
			doMock: func(args *args, mockData *mockData) {
				args.opts.Writer = bufio.NewWriter(&mockData.gotData)
				testHelper.mockTrxService.EXPECT().
					DownloadTransactionFileCSV(args.ctx, gomock.Any()).
					DoAndReturn(func(ctx context.Context, opts models.DownloadTransactionRequest) error {
						// write header
						args.opts.Writer.Write([]byte("Transaction ID,No Ref,Order Type,Transaction Type,Transaction Date,From Account,To Account,Amount,Status\n"))
						// write data
						args.opts.Writer.Write([]byte("c172ca84-9ae2-489c-ae4f-8ef372a109ae,55aa66bb-e6e0-4065-9f4a-64182e97e9d9,TOPUP,TOPUP,0001-01-01 00:00:00,189513,222000000069,0,1\n"))
						return nil
					})
			},
		},
		{
			name:      "error invalid startDate endDate",
			urlCalled: "/api/v1/transaction/download?startDate=2023-10-ERROR",
			args: args{
				ctx: context.Background(),
				opts: models.DownloadTransactionRequest{
					Options: models.TransactionFilterOptions{},
					Writer:  nil,
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":"INVALID_VALUES","message":"invalid format date caused by date 2023-10-ERROR format must be YYYY-MM-DD"}`,
				wantCode: 400,
			},
			doMock: func(args *args, mockData *mockData) {
			},
		},
		{
			name:      "error failed to get data",
			urlCalled: "/api/v1/transaction/download",
			args: args{
				ctx: context.Background(),
				opts: models.DownloadTransactionRequest{
					Options: models.TransactionFilterOptions{},
					Writer:  nil,
				},
			},
			mockData: mockData{
				wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
				wantCode: 500,
			},
			doMock: func(args *args, mockData *mockData) {
				c := testHelper.router.AcquireContext()
				args.opts.Writer = c.Response().Writer
				testHelper.router.ReleaseContext(c)
				testHelper.mockTrxService.EXPECT().DownloadTransactionFileCSV(args.ctx, gomock.Any()).Return(assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(&tt.args, &tt.mockData)
			}

			req := httptest.NewRequest(http.MethodGet, tt.urlCalled, nil)

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.mockData.wantCode, resp.StatusCode)
		})
	}
}
