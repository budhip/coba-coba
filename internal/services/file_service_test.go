package services_test

import (
	"context"
	"fmt"
	"mime/multipart"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/matcher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestFileService_Upload(t *testing.T) {
	testHelper := serviceTestHelper(t)

	nowDate := common.Now().Format(common.DateFormatDDMMYYYYWithoutDash)
	cacheKey := fmt.Sprintf("TRX-MANUAL-%s", nowDate)

	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: "26-Sep-2023,INSURANCE,100000,IDR,123CIH,IDR1208000011000,loan_account,,Insurance Premi"}
						}()
						return resultCh
					})
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().CheckAccountNumbers(gomock.Any(), []string{"123CIH", "IDR1208000011000"}).
					Return(map[string]bool{"123CIH": true, "IDR1208000011000": true}, nil)
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - get cache",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "err - parse cache",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("abc", nil)
			},
			wantErr: true,
		},
		{
			name: "error - stream",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", common.ErrDataNotFound)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 1, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Err: assert.AnError}
						}()
						return resultCh
					})
			},
			wantErr: true,
		},
		{
			name: "error - set cache",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", common.ErrDataNotFound)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 1, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(common.ErrDataNotFound)
			},
			wantErr: true,
		},
		{
			name: "happy path - account doesn't exists but still continue",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: "26-Sep-2023,INSURANCE,100000,IDR,123CIH,IDR1208000011000,loan_account,,Insurance Premi"}
						}()
						return resultCh
					})
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().CheckAccountNumbers(gomock.Any(), []string{"123CIH", "IDR1208000011000"}).
					Return(map[string]bool{"123CIH": false, "IDR1208000011000": false}, nil)
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "happy path - unable to query check account but still continue",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: "26-Sep-2023,INSURANCE,100000,IDR,123CIH,IDR1208000011000,loan_account,,Insurance Premi"}
						}()
						return resultCh
					})
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100"}, nil)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"100001"}, nil)
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().CheckAccountNumbers(gomock.Any(), []string{"123CIH", "IDR1208000011000"}).
					Return(nil, assert.AnError)
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "happy path - publish err but continue",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", common.ErrDataNotFound)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 1, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: "26-Sep-2023,INSURANCE,100000,IDR,123CIH,IDR1208000011000,loan_account,,Insurance Premi"}
						}()
						return resultCh
					})
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(assert.AnError)

				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return(nil, assert.AnError)
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return(nil, assert.AnError)

				// No mock expectations needed - using structured error logging
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			fileHeader := &multipart.FileHeader{Filename: "test.csv"}
			err := testHelper.fileService.Upload(context.Background(), fileHeader)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestFileService_UploadWalletTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)

	nowDate := common.Now().Format(common.DateFormatDDMMYYYYWithoutDash)
	cacheKey := fmt.Sprintf("%s-%s", models.WalletTransactionIDManualPrefix, nowDate)

	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func() {
				accountNumber := "211001000000691"
				defaultAccountBalances := []models.AccountBalance{
					{
						AccountNumber: accountNumber,
						Balance:       models.NewBalance(decimal.NewFromFloat(100000), decimal.Zero),
					},
					{
						AccountNumber: testHelper.config.AccountConfig.SystemAccountNumber,
						Balance:       models.NewBalance(decimal.Zero, decimal.Zero, models.WithIgnoreBalanceSufficiency()),
					},
				}

				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: `28082024,111,cashout,TUPVA,211001000000691,10000,,,"{""entity"":""AFA""}",TUPVA;TUPVA,200;500`}
						}()
						return resultCh
					})
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)

				testHelper.mockAccRepository.EXPECT().
					GetOneByLegacyId(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(&models.Account{AccountNumber: "211001000000691"}, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(matcher.ContextWithTimeoutRange(7*time.Second, 9*time.Second), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)
						accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
						balanceRepo := mockRepo.NewMockBalanceRepository(testHelper.mockCtrl)
						walletTrxRepo := mockRepo.NewMockWalletTransactionRepository(testHelper.mockCtrl)
						acuanRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)
						featureRepo := mockRepo.NewMockFeatureRepository(testHelper.mockCtrl)

						sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
						sqlRepo.EXPECT().GetWalletTransactionRepository().Return(walletTrxRepo).AnyTimes()
						sqlRepo.EXPECT().GetTransactionRepository().Return(acuanRepo).AnyTimes()
						sqlRepo.EXPECT().GetFeatureRepository().Return(featureRepo).AnyTimes()
						sqlRepo.EXPECT().GetBalanceRepository().Return(balanceRepo).AnyTimes()

						balanceRepo.EXPECT().
							GetMany(gomock.Any(), gomock.Any()).
							Return(defaultAccountBalances, nil)

						accRepo.EXPECT().UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(&defaultAccountBalances[0].Balance, nil).Times(3)

						newWalletTrx := models.NewWalletTransaction{
							AccountNumber:            accountNumber,
							RefNumber:                "111",
							TransactionType:          "TUPVA",
							TransactionFlow:          "",
							NetAmount:                models.Amount{},
							Amounts:                  []models.AmountDetail{},
							Status:                   "",
							DestinationAccountNumber: "",
							Description:              "",
							Metadata:                 map[string]any{},
						}
						created := newWalletTrx.ToWalletTransaction()
						walletTrxRepo.EXPECT().
							Create(gomock.AssignableToTypeOf(ctx), gomock.AssignableToTypeOf(newWalletTrx)).
							Return(&created, nil)
						acuanRepo.EXPECT().StoreBulkTransaction(gomock.Any(), gomock.Any()).
							Return(nil)
						testHelper.mockCacheRepository.EXPECT().Del(gomock.AssignableToTypeOf(ctx), gomock.AssignableToTypeOf([]string{})).
							Return(nil)
						testHelper.mockAccRepository.EXPECT().GetAccountNumberEntity(gomock.Any(), gomock.Any()).Return(map[string]string{}, nil)

						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

						return steps(ctx, sqlRepo)
					})
			},
			wantErr: false,
		},
		{
			name: "error - get cache",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "err - parse cache",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("abc", nil)
			},
			wantErr: true,
		},
		{
			name: "error - stream failed",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", common.ErrDataNotFound)
				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 1, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)

				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Err: assert.AnError}
						}()
						return resultCh
					})
			},
			wantErr: true,
		},
		{
			name: "error - set cache",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("", common.ErrDataNotFound)
				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 1, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(common.ErrDataNotFound)
			},
			wantErr: true,
		},
		{
			name: "error - empty file",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().
					Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().
					Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{}
						}()
						return resultCh
					})
			},
			wantErr: false,
		},
		{
			name: "error - invalid format data",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: "ddmmyyyy"}
						}()
						return resultCh
					})
				testHelper.mockDDDNotification.EXPECT().SendEmail(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - get master transaction type",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: `09072024,111,cashout,TUPVA,211001000000691,10000,,,"{""entity"":""AFA""}",TUPVA;TUPVA,200;500`}
						}()
						return resultCh
					})
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{""}, assert.AnError)
				testHelper.mockDDDNotification.EXPECT().SendEmail(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - atomic",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: `09072024,111,cashout,TUPVA,211001000000691,10000,,,"{""entity"":""AFA""}",TUPVA;TUPVA,200;500`}
						}()
						return resultCh
					})
				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
					Return([]string{"TUPVA"}, nil)

				testHelper.mockAccRepository.EXPECT().
					GetOneByLegacyId(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(&models.Account{AccountNumber: "211001000000691"}, nil)

				testHelper.mockFlagClient.EXPECT().
					IsEnabled(testHelper.config.FeatureFlagKeyLookup.UseAccountConfigFromExternal).
					Return(false)

				testHelper.mockSQLRepository.EXPECT().Atomic(matcher.ContextWithTimeoutRange(7*time.Second, 9*time.Second), gomock.Any()).
					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
						sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)
						accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
						balanceRepo := mockRepo.NewMockBalanceRepository(testHelper.mockCtrl)
						walletTrxRepo := mockRepo.NewMockWalletTransactionRepository(testHelper.mockCtrl)
						acuanRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)
						featureRepo := mockRepo.NewMockFeatureRepository(testHelper.mockCtrl)

						sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
						sqlRepo.EXPECT().GetWalletTransactionRepository().Return(walletTrxRepo).AnyTimes()
						sqlRepo.EXPECT().GetTransactionRepository().Return(acuanRepo).AnyTimes()
						sqlRepo.EXPECT().GetFeatureRepository().Return(featureRepo).AnyTimes()
						sqlRepo.EXPECT().GetBalanceRepository().Return(balanceRepo).AnyTimes()

						balanceRepo.EXPECT().GetMany(gomock.Any(), gomock.Any()).
							Return(nil, assert.AnError)
						return steps(ctx, sqlRepo)
					})

				testHelper.mockDDDNotification.EXPECT().SendEmail(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - send email message",
			doMock: func() {
				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
					Return("1", nil)
				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
					Return(nil)
				testHelper.mockFileRepo.EXPECT().StreamReadMultipartFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
						resultCh := make(chan repositories.StreamReadMultipartFileResult)
						go func() {
							defer close(resultCh)
							resultCh <- repositories.StreamReadMultipartFileResult{Data: "ddmmyyyy"}
						}()
						return resultCh
					})
				testHelper.mockDDDNotification.EXPECT().SendEmail(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			fileHeader := &multipart.FileHeader{Filename: "test.csv"}
			err := testHelper.fileService.UploadWalletTransaction(context.Background(), fileHeader, "test@gmail.com", "")
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

//func TestFileService_UploadWalletTransactionFromGCS(t *testing.T) {
//	testHelper := serviceTestHelper(t)
//
//	nowDate := time.Now().Format(common.DateFormatDDMMYYYYWithoutDash)
//	cacheKey := fmt.Sprintf("%s-%s", models.WalletTransactionIDManualPrefix, nowDate)
//	preset := models.DefaultPresetWalletFeature
//	allowedNegativeBalance := true
//	negativeBalanceLimit := decimal.NewFromInt(100000)
//	defaultWalletFeature := models.WalletFeature{
//		Preset:                 &preset,
//		AllowedNegativeBalance: &allowedNegativeBalance,
//		NegativeBalanceLimit:   &negativeBalanceLimit,
//	}
//
//	gcsPayload := &models.CloudStoragePayload{
//		Filename: fmt.Sprintf("test.csv"),
//		Path:     fmt.Sprintf("error"),
//	}
//
//	type args struct {
//		ctx            context.Context
//		filePath       string
//		bucketName     string
//		isPublishAcuan bool
//		gcsPayload     *models.CloudStoragePayload
//	}
//
//	type mockData struct {
//		resultFile *os.File
//		inputFile  *os.File
//	}
//
//	tests := []struct {
//		name     string
//		doMock   func(md *mockData)
//		args     args
//		mockData mockData
//		wantErr  bool
//	}{
//		{
//			name: "success case",
//			args: args{
//				isPublishAcuan: true,
//				gcsPayload:     gcsPayload,
//			},
//			doMock: func(md *mockData) {
//				accountNumber := "211001000000691"
//				balances := map[string]models.Balance{
//					accountNumber: models.NewBalance(decimal.NewFromFloat(100000), decimal.Zero),
//					testHelper.config.AccountConfig.SystemAccountNumber: models.NewBalance(decimal.Zero, decimal.Zero, models.WithIgnoreBalanceSufficiency()),
//				}
//				accountFeature := models.MapAccountFeature{
//					accountNumber:    defaultWalletFeature,
//					"00000100000000": defaultWalletFeature,
//				}
//
//				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
//					Return("1", nil)
//				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
//					Return(nil)
//
//				md.inputFile, _ = os.CreateTemp("", "test_file_recon_csv_input")
//				md.inputFile.Write([]byte("123456,100000,01-Jan-2023,this is remark\n"))
//
//				testHelper.mockGcs.EXPECT().NewReaderBucketCustom(gomock.Any(), gomock.Any(), gomock.Any()).Return(md.inputFile, nil)
//
//				testHelper.mockFileRepo.EXPECT().StreamReadCSVFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
//					DoAndReturn(func(ctx context.Context, file io.ReadCloser) <-chan repositories.StreamReadCSVFileResult {
//						resultCh := make(chan repositories.StreamReadCSVFileResult)
//						go func() {
//							defer close(resultCh)
//							resultCh <- repositories.StreamReadCSVFileResult{Data: []string{
//								`25062024`,
//								``,
//								`cashin`,
//								`TUPBM`,
//								`211001000000122`,
//								`12000`,
//								``,
//								`Test`,
//								``,
//								``,
//								``,
//							}}
//						}()
//						return resultCh
//					})
//
//				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
//					Return([]string{"TUPBM"}, nil)
//
//				testHelper.mockAccRepository.EXPECT().
//					GetOneByLegacyId(gomock.Any(), gomock.Any()).
//					Return(&models.Account{AccountNumber: "211001000000691"}, nil)
//
//				testHelper.mockGcs.EXPECT().WriteStreamCustomBucket(gomock.AssignableToTypeOf(context.Background()), "test", gcsPayload, gomock.Any()).
//					DoAndReturn(func(ctx context.Context, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult {
//						chanWrite := make(chan error)
//						go func() {
//							defer close(chanWrite)
//							chanWrite <- assert.AnError
//						}()
//						writeResult := models.NewWriteStreamResult(chanWrite, "")
//						return writeResult
//					})
//
//				testHelper.mockSQLRepository.EXPECT().Atomic(gomock.Any(), gomock.Any()).
//					DoAndReturn(func(ctx context.Context, steps func(ctx context.Context, r repositories.SQLRepository) error) error {
//						sqlRepo := mockRepo.NewMockSQLRepository(testHelper.mockCtrl)
//						accRepo := mockRepo.NewMockAccountRepository(testHelper.mockCtrl)
//						walletTrxRepo := mockRepo.NewMockWalletTransactionRepository(testHelper.mockCtrl)
//						acuanRepo := mockRepo.NewMockTransactionRepository(testHelper.mockCtrl)
//						featureRepo := mockRepo.NewMockFeatureRepository(testHelper.mockCtrl)
//
//						sqlRepo.EXPECT().GetAccountRepository().Return(accRepo).AnyTimes()
//						sqlRepo.EXPECT().GetWalletTransactionRepository().Return(walletTrxRepo).AnyTimes()
//						sqlRepo.EXPECT().GetTransactionRepository().Return(acuanRepo).AnyTimes()
//						sqlRepo.EXPECT().GetFeatureRepository().Return(featureRepo).AnyTimes()
//
//						accRepo.EXPECT().GetAccountBalances(gomock.Any(), gomock.Any()).Return(balances, nil)
//						featureRepo.EXPECT().GetFeatureByAccountNumbers(gomock.Any(), gomock.Any()).Return(accountFeature, nil)
//
//						ub := balances[accountNumber]
//						accRepo.EXPECT().UpdateAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
//							Return(&ub, nil).Times(3)
//
//						newWalletTrx := models.NewWalletTransaction{
//							AccountNumber:            accountNumber,
//							RefNumber:                "111",
//							TransactionType:          "TUPVA",
//							TransactionFlow:          "",
//							TransactionTime:          time.Time{},
//							NetAmount:                models.Amount{},
//							Amounts:                  []models.AmountDetail{},
//							Status:                   "",
//							DestinationAccountNumber: "",
//							Description:              "",
//							Metadata:                 map[string]any{},
//						}
//						created := newWalletTrx.ToWalletTransaction()
//						walletTrxRepo.EXPECT().
//							Create(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(newWalletTrx)).
//							Return(&created, nil)
//						acuanRepo.EXPECT().StoreBulkTransaction(gomock.Any(), gomock.Any()).
//							Return(nil)
//						testHelper.mockCacheRepository.EXPECT().Del(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf([]string{})).
//							Return(nil)
//
//						testHelper.mockTransactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
//
//						return steps(ctx, sqlRepo)
//					})
//			},
//		},
//		{
//			name: "error - get master transaction type",
//			doMock: func(md *mockData) {
//				md.inputFile, _ = os.CreateTemp("", "test_file_recon_csv_input")
//				md.inputFile.Write([]byte("123456,100000,01-Jan-2023,this is remark\n"))
//
//				testHelper.mockGcs.EXPECT().NewReaderBucketCustom(gomock.Any(), gomock.Any(), gomock.Any()).Return(md.inputFile, nil)
//
//				testHelper.mockCacheRepository.EXPECT().Get(gomock.AssignableToTypeOf(context.Background()), cacheKey).
//					Return("1", nil)
//				testHelper.mockCacheRepository.EXPECT().Set(gomock.AssignableToTypeOf(context.Background()), cacheKey, 2, gomock.AssignableToTypeOf(time.Until(common.NowEndOfDay()))).
//					Return(nil)
//
//				testHelper.mockFileRepo.EXPECT().StreamReadCSVFile(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
//					DoAndReturn(func(ctx context.Context, file io.ReadCloser) <-chan repositories.StreamReadCSVFileResult {
//						resultCh := make(chan repositories.StreamReadCSVFileResult)
//						go func() {
//							defer close(resultCh)
//							resultCh <- repositories.StreamReadCSVFileResult{Data: []string{
//								`25062024`,
//								``,
//								`cashin`,
//								`TUPBM`,
//								`211001000000122`,
//								`12000`,
//								``,
//								`Test`,
//								``,
//								``,
//								``,
//							}}
//						}()
//						return resultCh
//					})
//
//				testHelper.mockMasterData.EXPECT().GetListTransactionTypeCode(gomock.AssignableToTypeOf(context.Background())).
//					Return([]string{""}, assert.AnError)
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tc := range tests {
//		t.Run(tc.name, func(t *testing.T) {
//			if tc.doMock != nil {
//				tc.doMock(&tc.mockData)
//			}
//
//			err := testHelper.fileService.UploadWalletTransactionFromGCS(context.Background(), tc.args.filePath, tc.args.bucketName, tc.args.isPublishAcuan)
//			assert.Equal(t, tc.wantErr, err != nil)
//		})
//	}
//}
