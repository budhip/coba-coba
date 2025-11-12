package services_test

import (
	"context"
	"github.com/Unleash/unleash-client-go/v3/api"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_WalletTrxService_CreateTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)
	validAmount := decimal.NewFromFloat(10000)

	defaultAccountBalances := []models.AccountBalance{
		{
			AccountNumber: "111",
			Balance:       models.NewBalance(decimal.NewFromFloat(100000), decimal.Zero),
		},
		{
			AccountNumber: "222",
			Balance:       models.NewBalance(decimal.Zero, decimal.Zero),
		},
		{
			AccountNumber: testHelper.config.AccountConfig.SystemAccountNumber,
			Balance:       models.NewBalance(decimal.Zero, decimal.Zero, models.WithIgnoreBalanceSufficiency()),
		},
	}

	mockValidationBeforeCommit := func() {
		testHelper.mockMasterData.EXPECT().
			GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
			Return([]string{"TUPVA"}, nil)
	}

	mockAtomicHelper := func() *testServiceHelper {
		sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)
		accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
		balanceRepo := mockRepo.NewMockBalanceRepository(testHelper.mockCtrl)
		walletTrxRepo := mockRepo.NewMockWalletTransactionRepository(testHelper.mockCtrl)
		acuanRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)
		accConfigRepo := mockRepo.NewMockAccountConfigRepository(testHelper.mockCtrl)

		sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
		sqlRepo.EXPECT().GetWalletTransactionRepository().Return(walletTrxRepo).AnyTimes()
		sqlRepo.EXPECT().GetTransactionRepository().Return(acuanRepo).AnyTimes()
		sqlRepo.EXPECT().GetBalanceRepository().Return(balanceRepo).AnyTimes()
		sqlRepo.EXPECT().GetAccountConfigInternalRepository().Return(accConfigRepo).AnyTimes()
		sqlRepo.EXPECT().GetAccountConfigExternalRepository().Return(accConfigRepo).AnyTimes()

		return &testServiceHelper{
			mockMasterData:          testHelper.mockMasterData,
			mockAccRepository:       accRepo,
			mockBalanceRepository:   balanceRepo,
			mockWalletTrxRepository: walletTrxRepo,
			mockSQLRepository:       sqlRepo,
			mockTrxRepository:       acuanRepo,
		}
	}

	tests := []struct {
		name    string
		args    models.CreateWalletTransactionRequest
		doMock  func(args models.CreateWalletTransactionRequest)
		wantErr bool
	}{
		{
			name: "happy path",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow:          models.TransactionFlowTransfer,
				AccountNumber:            defaultAccountBalances[0].AccountNumber,
				DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
				RefNumber:                "333",
				TransactionTime:          time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				mockValidationBeforeCommit()

				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.LceRollout).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).
							Times(3)

						created := args.ToNewWalletTransaction().ToWalletTransaction()
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Create(gomock.Any(), gomock.AssignableToTypeOf(args.ToNewWalletTransaction())).
							Return(&created, nil)
						atomicHelper.mockTrxRepository.EXPECT().
							StoreBulkTransaction(gomock.Any(), gomock.Any()).
							Return(nil)

						testHelper.mockAccRepository.EXPECT().
							GetAccountNumberEntity(gomock.Any(), gomock.Any()).
							Return(map[string]string{}, nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: false,
		},
		{
			name: "happy path but data already created",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow:          models.TransactionFlowTransfer,
				AccountNumber:            defaultAccountBalances[0].AccountNumber,
				DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
				RefNumber:                "333",
				TransactionTime:          time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				mockValidationBeforeCommit()

				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPVA"]}`,
						},
						Enabled: false,
					})

				testHelper.mockWalletTrxRepository.EXPECT().CheckTransactionTypeAndReferenceNumber(gomock.Any(), args.TransactionType, args.RefNumber).Return(&models.WalletTransaction{
					ID:                       "string",
					Status:                   "string",
					AccountNumber:            "string",
					DestinationAccountNumber: "string",
					RefNumber:                "string",
					TransactionType:          "string",
					TransactionTime:          time.Time{},
					TransactionFlow:          "",
					NetAmount:                models.Amount{},
					Amounts:                  nil,
					Description:              "",
					Metadata:                 nil,
					CreatedAt:                time.Time{},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - validation - get master error",
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - validation - invalid transactionType",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "INVALID",
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - validation - invalid netAmount less than zero",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(decimal.Zero),
				},
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - validation - invalid amounts type",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "INVALID",
				}},
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - validation - invalid amounts value less than zero",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(decimal.Zero),
					},
				}},
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - validation - when transfer destinationAccountNumber not provided",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow: models.TransactionFlowTransfer,
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - validation - reserved transaction with cashin flow",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow: models.TransactionFlowCashIn,
				IsReserved:      true,
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})
				testHelper.mockMasterData.EXPECT().
					GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - atomic - err get balance",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow:          models.TransactionFlowTransfer,
				AccountNumber:            defaultAccountBalances[0].AccountNumber,
				DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
				RefNumber:                "333",
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				mockValidationBeforeCommit()

				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.LceRollout).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - atomic - err update balance",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow:          models.TransactionFlowTransfer,
				AccountNumber:            defaultAccountBalances[0].AccountNumber,
				DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
				RefNumber:                "333",
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				mockValidationBeforeCommit()

				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.LceRollout).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - atomic - err create wallet transaction",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow:          models.TransactionFlowTransfer,
				AccountNumber:            "111",
				DestinationAccountNumber: "222",
				RefNumber:                "333",
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				mockValidationBeforeCommit()

				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.LceRollout).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).
							Times(3)

						atomicHelper.mockWalletTrxRepository.EXPECT().
							Create(gomock.Any(), gomock.AssignableToTypeOf(args.ToNewWalletTransaction())).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - atomic - err publish",
			args: models.CreateWalletTransactionRequest{
				TransactionType: "TUPVA",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(validAmount),
				},
				Amounts: []models.AmountDetail{{
					Type: "TUPVA",
					Amount: &models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(validAmount),
					},
				}},
				TransactionFlow:          models.TransactionFlowTransfer,
				AccountNumber:            defaultAccountBalances[0].AccountNumber,
				DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
				RefNumber:                "333",
			},
			doMock: func(args models.CreateWalletTransactionRequest) {
				mockValidationBeforeCommit()

				testHelper.mockFlagClient.EXPECT().
					GetVariant(testHelper.config.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber).
					Return(&api.Variant{
						Name: "transactionType",
						Payload: api.Payload{
							Type:  "transactionType",
							Value: `{"transactionType": ["TUPQR"]}`,
						},
						Enabled: false,
					})

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.LceRollout).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).
							Times(3)

						created := args.ToNewWalletTransaction().ToWalletTransaction()
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Create(gomock.Any(), gomock.AssignableToTypeOf(args.ToNewWalletTransaction())).
							Return(&created, nil)
						atomicHelper.mockTrxRepository.EXPECT().
							StoreBulkTransaction(gomock.Any(), gomock.Any()).
							Return(nil)

						testHelper.mockTransactionNotification.EXPECT().
							Publish(gomock.Any(), gomock.Any()).
							Return(assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			_, err := testHelper.walletTrxService.CreateTransaction(context.Background(), tt.args)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_WalletTrxService_ProcessReservedTransaction_Commit(t *testing.T) {
	testHelper := serviceTestHelper(t)
	amount := decimal.NewFromFloat(10000)

	defaultAccountBalances := []models.AccountBalance{
		{
			AccountNumber: "111",
			Balance:       models.NewBalance(decimal.NewFromFloat(90000), decimal.NewFromFloat(10000)), // actual: 90000, pending: 10000 (reserved)
		},
		{
			AccountNumber: "222",
			Balance:       models.NewBalance(decimal.Zero, decimal.Zero),
		},
		{
			AccountNumber: testHelper.config.AccountConfig.SystemAccountNumber,
			Balance:       models.NewBalance(decimal.Zero, decimal.Zero, models.WithIgnoreBalanceSufficiency()),
		},
	}

	mockAtomicHelper := func() *testServiceHelper {
		sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)
		accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
		balanceRepo := mockRepo.NewMockBalanceRepository(testHelper.mockCtrl)
		walletTrxRepo := mockRepo.NewMockWalletTransactionRepository(testHelper.mockCtrl)
		acuanRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)
		accConfigRepo := mockRepo.NewMockAccountConfigRepository(testHelper.mockCtrl)

		sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
		sqlRepo.EXPECT().GetWalletTransactionRepository().Return(walletTrxRepo).AnyTimes()
		sqlRepo.EXPECT().GetTransactionRepository().Return(acuanRepo).AnyTimes()
		sqlRepo.EXPECT().GetBalanceRepository().Return(balanceRepo).AnyTimes()
		sqlRepo.EXPECT().GetAccountConfigInternalRepository().Return(accConfigRepo).AnyTimes()
		sqlRepo.EXPECT().GetAccountConfigExternalRepository().Return(accConfigRepo).AnyTimes()

		return &testServiceHelper{
			mockMasterData:          testHelper.mockMasterData,
			mockAccRepository:       accRepo,
			mockBalanceRepository:   balanceRepo,
			mockWalletTrxRepository: walletTrxRepo,
			mockSQLRepository:       sqlRepo,
			mockTrxRepository:       acuanRepo,
		}
	}

	currentTime := time.Now()

	argsCommit := models.UpdateStatusWalletTransactionRequest{
		TransactionId:      "123456",
		Action:             models.TransactionRequestCommitStatus,
		RawTransactionTime: currentTime.Format(time.RFC3339),
	}

	tests := []struct {
		name    string
		doMock  func(args models.UpdateStatusWalletTransactionRequest)
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusSuccess
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).
							Times(3)

						atomicHelper.mockTrxRepository.EXPECT().
							StoreBulkTransaction(gomock.Any(), gomock.Any()).
							Return(nil)

						testHelper.mockAccRepository.EXPECT().
							GetAccountNumberEntity(gomock.Any(), gomock.Any()).
							Return(map[string]string{}, nil).
							AnyTimes()

						testHelper.mockTransactionNotification.EXPECT().
							Publish(gomock.Any(), gomock.Any()).
							Return(nil)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: false,
		},
		{
			name: "success (trx already committed/success)",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(&models.WalletTransaction{
						Status: models.WalletTransactionStatusSuccess,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - trx not pending",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(&models.WalletTransaction{
						Status: models.WalletTransactionStatusCancel,
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - unable update status wallet transaction",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(&models.WalletTransaction{
						Status: models.WalletTransactionStatusPending,
					}, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - unable get balance",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}

				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusSuccess
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - unable update balance account",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}

				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusSuccess
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - unable insert child transaction",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}

				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusSuccess
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).
							Times(3)

						atomicHelper.mockTrxRepository.EXPECT().
							StoreBulkTransaction(gomock.Any(), gomock.Any()).
							Return(assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - unable publish notification",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusSuccess
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).
							Times(3)

						atomicHelper.mockTrxRepository.EXPECT().
							StoreBulkTransaction(gomock.Any(), gomock.Any()).
							Return(nil)

						testHelper.mockTransactionNotification.EXPECT().
							Publish(gomock.Any(), gomock.Any()).
							Return(assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(argsCommit)
			}

			_, err := testHelper.walletTrxService.ProcessReservedTransaction(context.Background(), argsCommit)
			t.Log(tt.name, "---", err)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_WalletTrxService_ProcessReservedTransaction_Cancel(t *testing.T) {
	testHelper := serviceTestHelper(t)
	amount := decimal.NewFromFloat(10000)

	defaultAccountBalances := []models.AccountBalance{
		{
			AccountNumber: "111",
			Balance:       models.NewBalance(decimal.Zero, decimal.NewFromInt(10000)),
		},
		{
			AccountNumber: "222",
			Balance:       models.NewBalance(decimal.Zero, decimal.Zero),
		},
		{
			AccountNumber: testHelper.config.AccountConfig.SystemAccountNumber,
			Balance:       models.NewBalance(decimal.Zero, decimal.Zero, models.WithIgnoreBalanceSufficiency()),
		},
	}

	mockAtomicHelper := func() *testServiceHelper {
		sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)
		accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
		balanceRepo := mockRepo.NewMockBalanceRepository(testHelper.mockCtrl)
		walletTrxRepo := mockRepo.NewMockWalletTransactionRepository(testHelper.mockCtrl)
		acuanRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)
		accConfigRepo := mockRepo.NewMockAccountConfigRepository(testHelper.mockCtrl)

		sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
		sqlRepo.EXPECT().GetWalletTransactionRepository().Return(walletTrxRepo).AnyTimes()
		sqlRepo.EXPECT().GetTransactionRepository().Return(acuanRepo).AnyTimes()
		sqlRepo.EXPECT().GetBalanceRepository().Return(balanceRepo).AnyTimes()
		sqlRepo.EXPECT().GetAccountConfigInternalRepository().Return(accConfigRepo).AnyTimes()
		sqlRepo.EXPECT().GetAccountConfigExternalRepository().Return(accConfigRepo).AnyTimes()

		return &testServiceHelper{
			mockMasterData:          testHelper.mockMasterData,
			mockAccRepository:       accRepo,
			mockBalanceRepository:   balanceRepo,
			mockWalletTrxRepository: walletTrxRepo,
			mockSQLRepository:       sqlRepo,
			mockTrxRepository:       acuanRepo,
		}
	}

	argsCancel := models.UpdateStatusWalletTransactionRequest{
		TransactionId:      "123456",
		Action:             models.TransactionRequestCancelStatus,
		RawTransactionTime: "",
	}

	tests := []struct {
		name    string
		doMock  func(args models.UpdateStatusWalletTransactionRequest)
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}

				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusCancel
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						testHelper.mockFlagClient.EXPECT().
							IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
							Return(false)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&models.Balance{}, nil).
							Times(3)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: false,
		},
		{
			name: "success (trx already cancelled)",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(&models.WalletTransaction{
						Status: models.WalletTransactionStatusCancel,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - trx not pending",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(&models.WalletTransaction{
						Status: models.WalletTransactionStatusSuccess,
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed - unable update status wallet transaction",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(&models.WalletTransaction{
						Status: models.WalletTransactionStatusPending,
					}, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - unable get balance",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}

				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusCancel
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)
						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - unable update balance account",
			doMock: func(args models.UpdateStatusWalletTransactionRequest) {
				walletTrx := &models.WalletTransaction{
					ID:                       args.TransactionId,
					TransactionType:          "ITRTF",
					AccountNumber:            defaultAccountBalances[0].AccountNumber,
					DestinationAccountNumber: defaultAccountBalances[1].AccountNumber,
					Status:                   models.WalletTransactionStatusPending,
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(amount),
					},
					TransactionFlow: models.TransactionFlowTransfer,
				}

				testHelper.mockWalletTrxRepository.EXPECT().
					GetById(gomock.Any(), args.TransactionId).
					Return(walletTrx, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicHelper := mockAtomicHelper()

						walletTrx.Status = models.WalletTransactionStatusCancel
						atomicHelper.mockWalletTrxRepository.EXPECT().
							Update(gomock.Any(), args.TransactionId, gomock.Any()).
							Return(walletTrx, nil)

						atomicHelper.mockBalanceRepository.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						atomicHelper.mockAccRepository.EXPECT().
							UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)

						return steps(ctx, atomicHelper.mockSQLRepository)
					})
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(argsCancel)
			}

			_, err := testHelper.walletTrxService.ProcessReservedTransaction(context.Background(), argsCancel)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_WalletTrxService_List(t *testing.T) {
	testHelper := serviceTestHelper(t)

	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func() {
				testHelper.mockWalletTrxRepository.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return([]models.WalletTransaction{}, nil)
				testHelper.mockWalletTrxRepository.EXPECT().CountAll(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return(0, nil)
			},
			wantErr: false,
		},
		{
			name: "err list",
			doMock: func() {
				testHelper.mockWalletTrxRepository.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return([]models.WalletTransaction{}, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "err count",
			doMock: func() {
				testHelper.mockWalletTrxRepository.EXPECT().List(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return([]models.WalletTransaction{}, nil)
				testHelper.mockWalletTrxRepository.EXPECT().CountAll(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(models.WalletTrxFilterOptions{})).Return(0, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			_, _, err := testHelper.walletTrxService.List(context.Background(), models.WalletTrxFilterOptions{})
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
