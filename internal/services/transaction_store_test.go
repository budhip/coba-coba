package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_NewStoreBulkTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)
	type args struct {
		ctx context.Context
		req []models.TransactionReq
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
			name: "success - create transaction",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
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
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = false
				}

				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), refNumbers).Return(trxExists, nil)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.BalanceLimitToggle).
					Return(true)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.BalanceLimitToggle).
					Return(true)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.AutoCreateAccountIfNotExists).
					Return(true)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.ExcludeConsumeTransactionFromSpecificSubCategory).
					Return(false)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{trxReq[0].FromAccount, trxReq[0].ToAccount}).
					Return(map[string]bool{trxReq[0].FromAccount: true, trxReq[0].ToAccount: false}, nil)
				testHelper.mockAccRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

				testHelper.mockSQLRepository.
					EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)

						accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
						trxRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)

						sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
						sqlRepo.EXPECT().GetTransactionRepository().Return(trxRepo).AnyTimes()

						balances := map[string]models.Balance{
							"1202517699": models.NewBalance(decimal.NewFromInt(20000), decimal.Zero),
							"123233333":  models.NewBalance(decimal.Zero, decimal.Zero),
						}

						testHelper.mockAccRepository.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
							Return(balances, nil)

						testHelper.mockCacheRepository.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)
						trxRepo.EXPECT().StoreBulkTransaction(gomock.Any(), trxReq).Return(nil)

						ub := models.NewBalance(decimal.NewFromInt(20000), decimal.Zero)
						accRepo.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&ub, nil).
							Times(2)
						return steps(ctx, sqlRepo)
					})

			},
			wantErr: false,
		},
		{
			name: "failed - get account error",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
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
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = false
				}

				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), refNumbers).Return(trxExists, nil)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{trxReq[0].FromAccount, trxReq[0].ToAccount}).
					Return(map[string]bool{trxReq[0].FromAccount: true, trxReq[0].ToAccount: false}, nil)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.AutoCreateAccountIfNotExists).
					Return(true)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.ExcludeConsumeTransactionFromSpecificSubCategory).
					Return(false)
				testHelper.mockAccRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
					testHelper.mockAccRepository.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).
						Return(map[string]models.Balance{}, assert.AnError)
					return steps(ctx, testHelper.mockSQLRepository)
				})

			},
			wantErr: true,
		},
		{
			name: "success - but transactions already exists on db",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
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
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository).AnyTimes()
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository).AnyTimes()

				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = true
				}
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), refNumbers).Return(trxExists, nil)
			},
			wantErr: true,
		},
		{
			name: "error - invalid order type and transaction type",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
						TransactionID:   "TRX1678947359NAVTaI2QQK6AyxkR5GLIGw",
						FromAccount:     "1202517699",
						ToAccount:       "123233333",
						FromNarrative:   "TOPUP.TRX",
						ToNarrative:     "TOPUP",
						TransactionDate: "2023-02-01",
						Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
						Status:          "",
						Method:          "TOPUP",
						TypeTransaction: "THIS SHOULD NOT BE ACCEPTED",
						Description:     "TOP UP",
						RefNumber:       "FT2303000001",
						OrderType:       "THIS SHOULD NOT BE ACCEPTED",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository).AnyTimes()
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository).AnyTimes()

				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = true
				}
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
			},
			wantErr: true,
		},
		{
			name: "error - ToRequest",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
						TransactionID:   "TRX1678947359NAVTaI2QQK6AyxkR5GLIGw",
						FromAccount:     "1202517699",
						ToAccount:       "123233333",
						FromNarrative:   "TOPUP.TRX",
						ToNarrative:     "TOPUP",
						TransactionDate: "2023-02-012",
						Amount:          decimal.NewNullDecimal(decimal.NewFromInt(20000)),
						Status:          "",
						Method:          "TOPUP",
						TypeTransaction: "ACRF",
						Description:     "TOP UP",
						RefNumber:       "FT2303000001",
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
			},
			wantErr: true,
		},
		{
			name: "error CheckAccountNumbers",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
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
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository).AnyTimes()
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository).AnyTimes()

				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = false
				}

				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), refNumbers).Return(trxExists, nil)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.AutoCreateAccountIfNotExists).
					Return(true)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{trxReq[0].FromAccount, trxReq[0].ToAccount}).
					Return(map[string]bool{trxReq[0].FromAccount: true, trxReq[0].ToAccount: false}, common.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "error Create account",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
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
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository).AnyTimes()
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository).AnyTimes()

				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = false
				}

				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), refNumbers).Return(trxExists, nil)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.AutoCreateAccountIfNotExists).
					Return(true)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{trxReq[0].FromAccount, trxReq[0].ToAccount}).
					Return(map[string]bool{trxReq[0].FromAccount: true, trxReq[0].ToAccount: false}, nil)
				testHelper.mockAccRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(common.ErrNoRowsAffected)
			},
			wantErr: true,
		},
		{
			name: "error Atomic",
			args: args{
				ctx: context.Background(),
				req: []models.TransactionReq{
					{
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
						OrderType:       "TOPUP",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository).AnyTimes()
				testHelper.mockSQLRepository.EXPECT().GetTransactionRepository().Return(testHelper.mockTrxRepository).AnyTimes()

				var (
					trxReq     []*models.Transaction
					refNumbers []string
					trxExists  = make(map[string]bool)
				)
				for _, testData := range args.req {
					req, err := testData.ToRequest()
					assert.NoError(t, err)
					trxReq = append(trxReq, &req)
					refNumbers = append(refNumbers, req.RefNumber)
					trxExists[req.RefNumber] = false
				}

				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockTrxRepository.EXPECT().CheckRefNumbers(gomock.Any(), refNumbers).Return(trxExists, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.AutoCreateAccountIfNotExists).
					Return(true)
				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.ExcludeConsumeTransactionFromSpecificSubCategory).
					Return(false)
				testHelper.mockAccRepository.EXPECT().
					CheckAccountNumbers(gomock.Any(), []string{trxReq[0].FromAccount, trxReq[0].ToAccount}).
					Return(map[string]bool{trxReq[0].FromAccount: true, trxReq[0].ToAccount: false}, nil)
				testHelper.mockAccRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).Return(common.ErrNoRowsAffected)
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
			err := testHelper.transactionService.NewStoreBulkTransaction(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
