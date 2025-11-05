package services_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_StoreBulkTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)
	type args struct {
		ctx       context.Context
		req       []models.TransactionReq
		batchSize int
	}

	type mockData struct {
	}

	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "success - create bulk transaction",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{{
					TransactionID:   "TRX1678947359NAVTaI2QQK6AyxkR5GLIGw",
					FromAccount:     "1202517699",
					ToAccount:       "123233333",
					FromNarrative:   "TOPUP.TRX",
					ToNarrative:     "TOPUP",
					TransactionDate: "2023-02-01",
					Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
					Status:          "",
					Method:          "TOPUP",
					TypeTransaction: "ACRF",
					Description:     "TOP UP",
					RefNumber:       "FT2303000001",
				},
				},
				batchSize: 1000,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository)

				var (
					transactionDBReq []*models.Transaction
					refNumbers       []string
					existsRefNumber  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					transactionDBReq = append(transactionDBReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					existsRefNumber[req.RefNumber] = false
				}

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), refNumbers).
					Return(existsRefNumber, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), gomock.Any()).Return(map[string]bool{"FT2303000001": false}, nil)

				testHelper.mockTrxRepository.EXPECT().StoreBulkTransaction(gomock.Any(), transactionDBReq).Return(nil)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{"1202517699", "123233333"}).
					Return(map[string]bool{"123233333": true, "1202517699": true}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - error create bulk transaction",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{{
					TransactionID:   "TRX1678947359NAVTaI2QQK6AyxkR5GLIGw",
					FromAccount:     "1202517699",
					ToAccount:       "123233333",
					FromNarrative:   "TOPUP.TRX",
					ToNarrative:     "TOPUP",
					TransactionDate: "2023-02-01",
					Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
					Status:          "",
					Method:          "TOPUP",
					TypeTransaction: "ACRF",
					Description:     "TOP UP",
					RefNumber:       "FT2303000001",
				},
				},
				batchSize: 1000,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository)

				var (
					transactionDBReq []*models.Transaction
					refNumbers       []string
					existsRefNumber  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					transactionDBReq = append(transactionDBReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					existsRefNumber[req.RefNumber] = false
				}

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), refNumbers).
					Return(existsRefNumber, nil)
				testHelper.mockTrxRepository.EXPECT().StoreBulkTransaction(gomock.Any(), transactionDBReq).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - error ensureAccountExists",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{{
					TransactionID:   "TRX1678947359NAVTaI2QQK6AyxkR5GLIGw",
					FromAccount:     "1202517699",
					ToAccount:       "123233333",
					FromNarrative:   "TOPUP.TRX",
					ToNarrative:     "TOPUP",
					TransactionDate: "2023-02-01",
					Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
					Status:          "",
					Method:          "TOPUP",
					TypeTransaction: "ACRF",
					Description:     "TOP UP",
					RefNumber:       "FT2303000001",
				},
				},
				batchSize: 1000,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository)

				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), gomock.Any()).Return(map[string]bool{"FT2303000001": false}, nil)
				testHelper.mockTrxRepository.EXPECT().StoreBulkTransaction(gomock.Any(), gomock.Any()).Return(nil)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{"1202517699", "123233333"}).
					Return(map[string]bool{}, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}
			err := testHelper.transactionService.StoreBulkTransaction(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestService_GetAllTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type mockData struct {
		data []models.Transaction
	}
	tests := []struct {
		name         string
		mockData     mockData
		doMock       func(mockData mockData)
		wantResponse []models.GetTransactionOut
		wantErr      bool
	}{
		{
			name: "test success",
			doMock: func(mockData mockData) {
				mockData.data = []models.Transaction{
					{
						TransactionID:   "c172ca84-9ae2-489c-ae4f-8ef372a109ae",
						RefNumber:       "55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
						OrderType:       "TOPUP",
						Method:          "TOPUP.VA",
						TypeTransaction: "TOPUP",
						FromAccount:     "189513",
						ToAccount:       "222000000069",
						Status:          "1",
					},
				}

				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, nil)
				testHelper.mockFlagClient.EXPECT().IsEnabled(gomock.Any()).Return(false)
				testHelper.mockTrxRepository.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.TransactionFilterOptions{})).Return(mockData.data, nil)
				testHelper.mockTrxRepository.EXPECT().CountAll(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.TransactionFilterOptions{})).Return(1, nil)
			},
			wantResponse: []models.GetTransactionOut{
				{
					TransactionID:   "c172ca84-9ae2-489c-ae4f-8ef372a109ae",
					RefNumber:       "55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
					OrderType:       "TOPUP",
					Method:          "TOPUP.VA",
					TransactionType: "TOPUP",
					FromAccount:     "189513",
					ToAccount:       "222000000069",
					Status:          "1",
				},
			},
			wantErr: false,
		},
		{
			name: "test error GetListOrderType",
			doMock: func(mockData mockData) {
				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "test error GetList",
			doMock: func(mockData mockData) {
				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, nil)
				testHelper.mockFlagClient.EXPECT().IsEnabled(gomock.Any()).Return(false)
				testHelper.mockTrxRepository.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.TransactionFilterOptions{})).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "test error CountAllData",
			doMock: func(mockData mockData) {
				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, nil)
				testHelper.mockFlagClient.EXPECT().IsEnabled(gomock.Any()).Return(false)
				testHelper.mockTrxRepository.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.TransactionFilterOptions{})).Return(mockData.data, nil)
				testHelper.mockTrxRepository.EXPECT().CountAll(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.TransactionFilterOptions{})).Return(0, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.mockData)
			}
			gotData, gotTotal, err := testHelper.transactionService.GetAllTransaction(context.Background(), models.TransactionFilterOptions{})
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantResponse, gotData)
			assert.Equal(t, len(tt.wantResponse), gotTotal)
		})
	}
}

func TestService_DownloadTransactionFileCSV(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type mockData struct {
		csvBytes bytes.Buffer
		data     []models.Transaction
	}

	type args struct {
		ctx  context.Context
		opts models.DownloadTransactionRequest
	}
	tests := []struct {
		name         string
		args         args
		mockData     mockData
		doMock       func(args *args, mockData *mockData)
		wantResponse []byte
		wantErr      bool
	}{
		{
			name: "test success",
			args: args{
				ctx: context.Background(),
				opts: models.DownloadTransactionRequest{
					Options: models.TransactionFilterOptions{},
				},
			},
			mockData: mockData{
				data: []models.Transaction{
					{
						TransactionID:   "c172ca84-9ae2-489c-ae4f-8ef372a109ae",
						RefNumber:       "55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
						OrderType:       "TOPUP",
						Method:          "TOPUP.VA",
						TypeTransaction: "TOPUP",
						TransactionDate: time.Date(2023, 10, 23, 0, 0, 0, 0, time.UTC),
						FromAccount:     "189513",
						ToAccount:       "222000000069",
						Status:          "1",
					},
				},
			},
			doMock: func(args *args, m *mockData) {
				args.opts.Writer = bufio.NewWriter(&m.csvBytes)
				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, nil)
				testHelper.mockTrxRepository.EXPECT().GetStatusCount(args.ctx, models.DefaultThresholdStatusCountTransaction, args.opts.Options).Return(models.StatusCountTransaction{
					ExceedThreshold: false,
					Threshold:       models.DefaultThresholdStatusCountTransaction,
				}, nil)
				testHelper.mockTrxRepository.EXPECT().
					StreamAll(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
						chanTrx := make(chan models.TransactionStreamResult)
						go func() {
							defer close(chanTrx)
							for _, datum := range m.data {
								chanTrx <- models.TransactionStreamResult{
									Data: datum,
								}
							}
						}()
						return chanTrx
					})

			},
			wantResponse: []byte("Transaction ID,No Ref,Order Type Code,Order Type Name,Transaction Type Code,Transaction Type Name,Transaction Date,From Account Number,From Account Name,From Account Product Name,To Account Number,To Account Name,To Account Product Name,Amount,Status,Description,Method,Currency,Metadata\nc172ca84-9ae2-489c-ae4f-8ef372a109ae,55aa66bb-e6e0-4065-9f4a-64182e97e9d9,TOPUP,,TOPUP,,0001-01-01 07:07:12,189513,,,222000000069,,,0,1,,TOPUP.VA,,\n"),
			wantErr:      false,
		},
		{
			name: "failed to get data from master data gcs",
			args: args{
				ctx: context.Background(),
				opts: models.DownloadTransactionRequest{
					Options: models.TransactionFilterOptions{},
				},
			},
			mockData: mockData{
				data: []models.Transaction{
					{
						TransactionID:   "c172ca84-9ae2-489c-ae4f-8ef372a109ae",
						RefNumber:       "55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
						OrderType:       "TOPUP",
						Method:          "TOPUP.VA",
						TypeTransaction: "TOPUP",
						FromAccount:     "189513",
						ToAccount:       "222000000069",
						Status:          "1",
					},
				},
			},
			doMock: func(args *args, m *mockData) {
				args.opts.Writer = bufio.NewWriter(&m.csvBytes)
				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, nil)
				testHelper.mockTrxRepository.EXPECT().GetStatusCount(args.ctx, models.DefaultThresholdStatusCountTransaction, args.opts.Options).Return(models.StatusCountTransaction{}, assert.AnError)
			},
			wantResponse: []byte(nil),
			wantErr:      true,
		},
		{
			name: "failed to get data from repository",
			args: args{
				ctx: context.Background(),
				opts: models.DownloadTransactionRequest{
					Options: models.TransactionFilterOptions{},
				},
			},
			mockData: mockData{
				data: []models.Transaction{
					{
						TransactionID:   "c172ca84-9ae2-489c-ae4f-8ef372a109ae",
						RefNumber:       "55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
						OrderType:       "TOPUP",
						Method:          "TOPUP.VA",
						TypeTransaction: "TOPUP",
						FromAccount:     "189513",
						ToAccount:       "222000000069",
						Status:          "1",
					},
				},
			},
			doMock: func(args *args, m *mockData) {
				args.opts.Writer = bufio.NewWriter(&m.csvBytes)
				testHelper.mockMasterData.EXPECT().GetListOrderType(gomock.AssignableToTypeOf(context.Background()), models.FilterMasterData{}).Return([]models.OrderType{}, nil)
				testHelper.mockTrxRepository.EXPECT().GetStatusCount(args.ctx, models.DefaultThresholdStatusCountTransaction, args.opts.Options).Return(models.StatusCountTransaction{
					ExceedThreshold: false,
					Threshold:       models.DefaultThresholdStatusCountTransaction,
				}, nil)
				testHelper.mockTrxRepository.EXPECT().
					StreamAll(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
						chanTrx := make(chan models.TransactionStreamResult)
						go func() {
							defer close(chanTrx)
							chanTrx <- models.TransactionStreamResult{
								Err: assert.AnError,
							}
						}()
						return chanTrx
					})
			},
			wantResponse: []byte(nil),
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(&tt.args, &tt.mockData)
			}
			err := testHelper.transactionService.DownloadTransactionFileCSV(tt.args.ctx, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantResponse, tt.mockData.csvBytes.Bytes())
		})
	}
}

func TestService_GenerateTransactionReport(t *testing.T) {
	testHelper := serviceTestHelper(t)
	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "failed - error",
			doMock: func() {
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository)

				reportDate := *common.YesterdayTime()
				gcsPayload := &models.CloudStoragePayload{
					Filename: fmt.Sprintf("%d%02d%02d__1.csv", reportDate.Year(), reportDate.Month(), reportDate.Day()),
					Path:     fmt.Sprintf("%s/%d/%d", models.TransactionReportName, reportDate.Year(), reportDate.Month()),
				}

				testHelper.mockTrxRepository.EXPECT().
					StreamAll(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
						chanTrx := make(chan models.TransactionStreamResult)
						go func() {
							defer close(chanTrx)
							chanTrx <- models.TransactionStreamResult{Err: assert.AnError}
						}()
						return chanTrx
					})

				tempFile, _ := os.CreateTemp("", "test_mock_gcs")

				testHelper.mockGcs.EXPECT().NewWriter(gomock.Any(), gcsPayload).Return(tempFile)
				testHelper.mockGcs.EXPECT().GetURL(gcsPayload).Return("https://test.com")

				testHelper.mockGcs.EXPECT().DeleteFile(gomock.Any(), gcsPayload).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "success - GenerateTransactionReport",
			doMock: func() {
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository)

				reportDate := *common.YesterdayTime()
				gcsPayload := &models.CloudStoragePayload{
					Filename: fmt.Sprintf("%d%02d%02d__1.csv", reportDate.Year(), reportDate.Month(), reportDate.Day()),
					Path:     fmt.Sprintf("%s/%d/%d", models.TransactionReportName, reportDate.Year(), reportDate.Month()),
				}
				testHelper.mockTrxRepository.EXPECT().
					StreamAll(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
						chanTrx := make(chan models.TransactionStreamResult)
						go func() {
							defer close(chanTrx)
						}()
						return chanTrx
					})

				tempFile, _ := os.CreateTemp("", "test_mock_gcs")

				testHelper.mockGcs.EXPECT().NewWriter(gomock.Any(), gcsPayload).Return(tempFile)
				testHelper.mockGcs.EXPECT().GetURL(gcsPayload).Return("https://test.com")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}
			_, err := testHelper.transactionService.GenerateTransactionReport(context.Background())
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestService_StoreTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)
	type args struct {
		ctx              context.Context
		req              models.TransactionReq
		storeProcessType models.TransactionStoreProcessType
	}

	type mockData struct {
	}

	defaultReq := models.TransactionReq{
		TransactionID:   "b36bcd5a-6e59-4704-8a17-dfc4da0d30f5",
		FromAccount:     "111111111",
		ToAccount:       "222222222",
		TransactionDate: "2023-02-01",
		Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
		OrderType:       "DSB",
		TypeTransaction: "DSBAA",
		Description:     "transfer",
		RefNumber:       "TRX-REF-NUM",
		Status:          string(models.TransactionStatusSuccess),
	}

	defaultReserve := defaultReq
	defaultReserve.Status = string(models.TransactionStatusPending)

	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "success - create transaction",
			args: args{
				ctx:              context.Background(),
				req:              defaultReq,
				storeProcessType: models.TransactionStoreProcessNormal,
			},
			doMock: func(args args, mockData mockData) {
				req, err := args.req.ToRequest()
				assert.NoError(t, err)

				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"TRX-REF-NUM": false}, nil)

				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{args.req.FromAccount, args.req.ToAccount}).
					Return(map[string]bool{args.req.FromAccount: true, args.req.ToAccount: true}, nil)

				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						balances := map[string]models.Balance{
							args.req.FromAccount: models.NewBalance(decimal.NewFromInt(1000000), decimal.Zero),
							args.req.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
						}

						atomicAccRepo.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
							Return(balances, nil)

						ub1, ub2 := balances[args.req.FromAccount], balances[args.req.ToAccount]
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.FromAccount, gomock.Any()).
							Return(&ub1, nil)
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.ToAccount, gomock.Any()).
							Return(&ub2, nil)

						atomicTrxRepo.EXPECT().Store(gomock.Any(), &req).Return(nil)

						testHelper.mockCacheRepository.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: false,
		},
		{
			name: "failed - error publish notification",
			args: args{
				ctx:              context.Background(),
				req:              defaultReq,
				storeProcessType: models.TransactionStoreProcessNormal,
			},
			doMock: func(args args, mockData mockData) {
				req, err := args.req.ToRequest()
				assert.NoError(t, err)

				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"TRX-REF-NUM": false}, nil)

				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{args.req.FromAccount, args.req.ToAccount}).
					Return(map[string]bool{args.req.FromAccount: true, args.req.ToAccount: true}, nil)

				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						balances := map[string]models.Balance{
							args.req.FromAccount: models.NewBalance(decimal.NewFromInt(1000000), decimal.Zero),
							args.req.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
						}

						atomicAccRepo.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
							Return(balances, nil)

						ub1, ub2 := balances[args.req.FromAccount], balances[args.req.ToAccount]
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.FromAccount, gomock.Any()).
							Return(&ub1, nil)
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.ToAccount, gomock.Any()).
							Return(&ub2, nil)

						atomicTrxRepo.EXPECT().Store(gomock.Any(), &req).Return(nil)

						testHelper.mockCacheRepository.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "success - create transaction reserve",
			args: args{
				ctx:              context.Background(),
				req:              defaultReserve,
				storeProcessType: models.TransactionStoreProcessReserved,
			},
			doMock: func(args args, mockData mockData) {
				req, err := args.req.ToRequest()
				assert.NoError(t, err)

				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"TRX-REF-NUM": false}, nil)

				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{args.req.FromAccount, args.req.ToAccount}).
					Return(map[string]bool{args.req.FromAccount: true, args.req.ToAccount: true}, nil)

				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						balances := map[string]models.Balance{
							args.req.FromAccount: models.NewBalance(decimal.NewFromInt(1000000), decimal.Zero),
							args.req.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
						}

						atomicAccRepo.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
							Return(balances, nil)

						ub1, ub2 := balances[args.req.FromAccount], balances[args.req.ToAccount]
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.FromAccount, gomock.Any()).
							Return(&ub1, nil)
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.ToAccount, gomock.Any()).
							Return(&ub2, nil)

						atomicTrxRepo.EXPECT().Store(gomock.Any(), &req).Return(nil)

						testHelper.mockCacheRepository.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: false,
		},
		{
			name: "failed - duplicate transaction",
			args: args{
				ctx:              context.Background(),
				req:              defaultReq,
				storeProcessType: models.TransactionStoreProcessNormal,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"TRX-REF-NUM": true}, nil)
			},
			wantErr: true,
		},
		{
			name: "error - Store transaction",
			args: args{
				ctx:              context.Background(),
				req:              defaultReq,
				storeProcessType: models.TransactionStoreProcessNormal,
			},
			doMock: func(args args, mockData mockData) {
				req, err := args.req.ToRequest()
				assert.NoError(t, err)

				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"TRX-REF-NUM": false}, nil)

				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{args.req.FromAccount, args.req.ToAccount}).
					Return(map[string]bool{args.req.FromAccount: true, args.req.ToAccount: true}, nil)

				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						balances := map[string]models.Balance{
							args.req.FromAccount: models.NewBalance(decimal.NewFromInt(1000000), decimal.Zero),
							args.req.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
						}

						atomicAccRepo.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
							Return(balances, nil)

						ub1, ub2 := balances[args.req.FromAccount], balances[args.req.ToAccount]
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.FromAccount, gomock.Any()).
							Return(&ub1, nil)
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.ToAccount, gomock.Any()).
							Return(&ub2, nil)

						atomicTrxRepo.EXPECT().Store(gomock.Any(), &req).Return(assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "success - ensureAccountExists (success create account)",
			args: args{
				ctx:              context.Background(),
				req:              defaultReq,
				storeProcessType: models.TransactionStoreProcessNormal,
			},
			doMock: func(args args, mockData mockData) {
				req, err := args.req.ToRequest()
				assert.NoError(t, err)

				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"FT2303000001": false}, nil)

				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{args.req.FromAccount, args.req.ToAccount}).
					Return(map[string]bool{args.req.FromAccount: true, args.req.ToAccount: true}, nil)

				testHelper.mockAccRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						balances := map[string]models.Balance{
							args.req.FromAccount: models.NewBalance(decimal.NewFromInt(1000000), decimal.Zero),
							args.req.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
						}

						atomicAccRepo.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
							Return(balances, nil)

						ub1, ub2 := balances[args.req.FromAccount], balances[args.req.ToAccount]
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.FromAccount, gomock.Any()).
							Return(&ub1, nil)
						atomicAccRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), args.req.ToAccount, gomock.Any()).
							Return(&ub2, nil)

						atomicTrxRepo.EXPECT().Store(gomock.Any(), &req).Return(nil)

						testHelper.mockCacheRepository.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: false,
		},
		{
			name: "failed - error ensureAccountExists (failed create account)",
			args: args{
				ctx:              context.Background(),
				req:              defaultReq,
				storeProcessType: models.TransactionStoreProcessNormal,
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockMasterData.EXPECT().
					GetListOrderTypeCode(gomock.Any()).
					Return([]string{"DSB"}, nil)
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.Any()).
					Return([]string{"DSBAA"}, nil)

				testHelper.mockTrxRepository.EXPECT().
					CheckRefNumbers(gomock.Any(), args.req.RefNumber).
					Return(map[string]bool{"FT2303000001": false}, nil)

				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{args.req.FromAccount, args.req.ToAccount}).
					Return(map[string]bool{args.req.FromAccount: true, args.req.ToAccount: false}, nil)

				testHelper.mockAccRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}
			_, err := testHelper.transactionService.StoreTransaction(tt.args.ctx, tt.args.req, tt.args.storeProcessType, "")
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestService_GetByTransactionTypeAndRefNumber(t *testing.T) {
	testHelper := serviceTestHelper(t)

	tests := []struct {
		name    string
		doMock  func(arg *models.TransactionGetByTypeAndRefNumberRequest)
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func(arg *models.TransactionGetByTypeAndRefNumberRequest) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionTypeAndRefNumber(gomock.AssignableToTypeOf(context.Background()), arg).
					Return(&models.GetTransactionOut{}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - err repo",
			doMock: func(arg *models.TransactionGetByTypeAndRefNumberRequest) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionTypeAndRefNumber(gomock.AssignableToTypeOf(context.Background()), arg).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			param := &models.TransactionGetByTypeAndRefNumberRequest{}
			if tc.doMock != nil {
				tc.doMock(param)
			}
			_, err := testHelper.transactionService.GetByTransactionTypeAndRefNumber(context.Background(), param)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestService_CommitReservedStatus(t *testing.T) {
	testHelper := serviceTestHelper(t)

	tests := []struct {
		name    string
		doMock  func(trxID string)
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:        "0",
					TransactionID: trxID,
					ID:            1,
					Amount:        decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount:   "FromAccount",
					ToAccount:     "ToAccount",
				}

				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo).AnyTimes()
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo).AnyTimes()

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount, trx.ToAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{
								trx.FromAccount: models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10)),
								trx.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
							},
							nil,
						)

						ub := models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10))
						atomicAccRepo.EXPECT().UpdateAccountBalance(
							gomock.AssignableToTypeOf(context.Background()),
							gomock.AssignableToTypeOf(trx.FromAccount),
							gomock.AssignableToTypeOf(models.Balance{}), // TODO: Check balance value
						).Return(&ub, nil).Times(2)

						testHelper.mockCacheRepository.EXPECT().
							Del(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
							Return(nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: false,
		},
		{
			name: "failed - unable publish to notification",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:        "0",
					TransactionID: trxID,
					ID:            1,
					Amount:        decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount:   "FromAccount",
					ToAccount:     "ToAccount",
				}

				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo).AnyTimes()
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo).AnyTimes()

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount, trx.ToAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{
								trx.FromAccount: models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10)),
								trx.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
							},
							nil,
						)

						ub := models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10))
						atomicAccRepo.EXPECT().UpdateAccountBalance(
							gomock.AssignableToTypeOf(context.Background()),
							gomock.AssignableToTypeOf(trx.FromAccount),
							gomock.AssignableToTypeOf(models.Balance{}), // TODO: Check balance value
						).Return(&ub, nil).Times(2)

						testHelper.mockCacheRepository.EXPECT().
							Del(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
							Return(nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err get trx",
			doMock: func(trxID string) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "success - trx committed",
			doMock: func(trxID string) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(&models.Transaction{Status: "1"}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - trx not pending",
			doMock: func(trxID string) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(&models.Transaction{Status: "2"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - err update status",
			doMock: func(trxID string) {
				trx := &models.Transaction{Status: "0", ID: 1}
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(nil, assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - invalid trx amount",
			doMock: func(trxID string) {
				trx := &models.Transaction{Status: "0", ID: 1}
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(trx, nil)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err get account",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:      "0",
					ID:          1,
					Amount:      decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
				}
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount, trx.ToAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{},
							assert.AnError,
						)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err commit update balance",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:      "0",
					ID:          1,
					Amount:      decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
				}

				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo).AnyTimes()
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo).AnyTimes()

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount, trx.ToAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{
								trx.FromAccount: models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10)),
								trx.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
							},
							nil,
						)
						atomicAccRepo.EXPECT().UpdateAccountBalance(
							gomock.AssignableToTypeOf(context.Background()),
							gomock.AssignableToTypeOf(trx.FromAccount),
							gomock.AssignableToTypeOf(models.Balance{}),
						).Return(nil, assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err delete cache",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:      "0",
					ID:          1,
					Amount:      decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
				}

				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo).AnyTimes()
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo).AnyTimes()

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusSuccessNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount, trx.ToAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{
								trx.FromAccount: models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10)),
								trx.ToAccount:   models.NewBalance(decimal.Zero, decimal.Zero),
							},
							nil,
						)

						ub := models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10))
						atomicAccRepo.EXPECT().UpdateAccountBalance(
							gomock.AssignableToTypeOf(context.Background()),
							gomock.AssignableToTypeOf(trx.FromAccount),
							gomock.AssignableToTypeOf(models.Balance{}), // TODO: Check balance value
						).Return(&ub, nil).Times(2)

						testHelper.mockCacheRepository.EXPECT().
							Del(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
							Return(assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trxID := "b36bcd5a-6e59-4704-8a17-dfc4da0d30f5"
			if tc.doMock != nil {
				tc.doMock(trxID)
			}
			_, err := testHelper.transactionService.CommitReservedTransaction(context.Background(), trxID, "")
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestService_CancelReservedStatus(t *testing.T) {
	testHelper := serviceTestHelper(t)

	tests := []struct {
		name    string
		doMock  func(trxID string)
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:      "0",
					ID:          1,
					Amount:      decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
				}

				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo).AnyTimes()
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo).AnyTimes()

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusCancelNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{trx.FromAccount: models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10))},
							nil,
						)

						ub := models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10))
						atomicAccRepo.EXPECT().UpdateAccountBalance(
							gomock.AssignableToTypeOf(context.Background()),
							trx.FromAccount,
							gomock.AssignableToTypeOf(models.Balance{}),
						).Return(&ub, nil)

						return f(ctx, atomicRepo)
					})

			},
			wantErr: false,
		},
		{
			name: "failed - err get trx",
			doMock: func(trxID string) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "success - trx cancelled",
			doMock: func(trxID string) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(&models.Transaction{Status: "2"}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - trx not pending",
			doMock: func(trxID string) {
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(&models.Transaction{Status: "1"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - err update status",
			doMock: func(trxID string) {
				trx := &models.Transaction{Status: "0", ID: 1}
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusCancelNum,
						).Return(nil, assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - invalid trx amount",
			doMock: func(trxID string) {
				trx := &models.Transaction{Status: "0", ID: 1}
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusCancelNum,
						).Return(trx, nil)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err get account",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:      "0",
					ID:          1,
					Amount:      decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
				}
				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo)
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusCancelNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{},
							assert.AnError,
						)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err commit update balance",
			doMock: func(trxID string) {
				trx := &models.Transaction{
					Status:      "0",
					ID:          1,
					Amount:      decimal.NullDecimal{Decimal: decimal.NewFromFloat(10), Valid: true},
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
				}

				testHelper.mockTrxRepository.EXPECT().
					GetByTransactionID(gomock.AssignableToTypeOf(context.Background()), trxID).
					Return(trx, nil)
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicTrxRepo := mock.NewMockTransactionRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().DisableIndexScan(gomock.Any()).Return(nil)
						atomicRepo.EXPECT().GetTransactionRepository().Return(atomicTrxRepo).AnyTimes()
						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo).AnyTimes()

						atomicTrxRepo.EXPECT().UpdateStatus(
							gomock.AssignableToTypeOf(context.Background()),
							trx.ID,
							models.TransactionStatusCancelNum,
						).Return(trx, nil)
						atomicAccRepo.EXPECT().GetAccountBalances(
							gomock.AssignableToTypeOf(context.Background()),
							models.GetAccountBalanceRequest{
								AccountNumbers: []string{trx.FromAccount},
								ForUpdate:      true,
							},
						).Return(
							map[string]models.Balance{trx.FromAccount: models.NewBalance(decimal.NewFromInt(100), decimal.NewFromInt(10))},
							nil,
						)
						atomicAccRepo.EXPECT().UpdateAccountBalance(
							gomock.AssignableToTypeOf(context.Background()),
							trx.FromAccount,
							gomock.AssignableToTypeOf(models.Balance{}),
						).Return(nil, assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trxID := "test"
			if tc.doMock != nil {
				tc.doMock(trxID)
			}
			_, err := testHelper.transactionService.CancelReservedTransaction(context.Background(), trxID)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestService_GetStatusCount(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		threshold uint
		opts      models.TransactionFilterOptions
	}

	tests := []struct {
		name         string
		args         args
		doMock       func(a args)
		wantResponse models.StatusCountTransaction
		wantErr      bool
	}{
		{
			name: "test success",
			doMock: func(a args) {
				testHelper.mockTrxRepository.
					EXPECT().
					GetStatusCount(gomock.AssignableToTypeOf(context.Background()), a.threshold, a.opts).
					Return(models.StatusCountTransaction{
						ExceedThreshold: true,
						Threshold:       42069,
					}, nil)
			},
			wantResponse: models.StatusCountTransaction{
				ExceedThreshold: true,
				Threshold:       42069,
			},
			wantErr: false,
		},
		{
			name: "test error get status count",
			doMock: func(a args) {
				testHelper.mockTrxRepository.
					EXPECT().
					GetStatusCount(gomock.AssignableToTypeOf(context.Background()), a.threshold, a.opts).
					Return(models.StatusCountTransaction{}, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}
			gotData, err := testHelper.transactionService.GetStatusCount(context.Background(), tt.args.threshold, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantResponse, gotData)
		})
	}
}
