package services_test

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mock3 "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	mockAcuanClient "bitbucket.org/Amartha/go-fp-transaction/internal/common/acuanclient/mock"
	mockDDD "bitbucket.org/Amartha/go-fp-transaction/internal/common/ddd_notification/mock"
	mockIDGenerator "bitbucket.org/Amartha/go-fp-transaction/internal/common/idgenerator/mock"
	mockPublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/publisher/mock"
	mockQueueUnicorn "bitbucket.org/Amartha/go-fp-transaction/internal/common/queueunicorn/mock"
	mockKafkaRecon "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka_recon/mock"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	goAcuanLib "bitbucket.org/Amartha/go-acuan-lib/model"

	"github.com/Shopify/sarama"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type reconSUT struct {
	sut services.ReconService

	mockStorageRepo           *mockRepo.MockCloudStorageRepository
	mockSQLRepo               *mockRepo.MockSQLRepository
	mockABDRepo               *mockRepo.MockAccountBalanceDailyRepository
	mockAccRepo               *mockRepo.MockAccountRepository
	mockReconToolHistoryRepo  *mockRepo.MockReconToolHistoryRepository
	mockTransactionRepository *mockRepo.MockTransactionRepository

	mockFileRepo        *mockRepo.MockFileRepository
	mockDDDNotification *mockDDD.MockDDDNotification
	mockReconPub        *mockPublisher.MockPublisher
	mockBalanceHVTPub   *mockPublisher.MockPublisher

	mockConsumer *mockKafkaRecon.MockConsumer

	reportDate *time.Time
	gcsPayload *models.CloudStoragePayload
}

func initReconSUT(t *testing.T) reconSUT {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cfg := config.Config{}

	mockSQLRepository := mockRepo.NewMockSQLRepository(mockCtrl)
	mockCacheRepository := mockRepo.NewMockCacheRepository(mockCtrl)
	mockCloudStorageRepository := mockRepo.NewMockCloudStorageRepository(mockCtrl)
	mockConsumer := mockKafkaRecon.NewMockConsumer(mockCtrl)
	mockABCRepo := mockRepo.NewMockAccountBalanceDailyRepository(mockCtrl)
	mockAccRepo := mockRepo.NewMockAccountRepository(mockCtrl)
	mockReconToolHistoryRepo := mockRepo.NewMockReconToolHistoryRepository(mockCtrl)
	mockTransactionRepository := mockRepo.NewMockTransactionRepository(mockCtrl)
	mockAcuanClient := mockAcuanClient.NewMockAcuanClient(mockCtrl)
	mockAccountingClient := mock2.NewMockClient(mockCtrl)
	mockIDGenerator := mockIDGenerator.NewMockGenerator(mockCtrl)
	mockFileRepo := mockRepo.NewMockFileRepository(mockCtrl)
	mockMasterDataRepo := mockRepo.NewMockMasterDataRepository(mockCtrl)
	mockDDDNotification := mockDDD.NewMockDDDNotification(mockCtrl)
	mockReconPublisher := mockPublisher.NewMockPublisher(mockCtrl)
	mockBalanceHVTPublisher := mockPublisher.NewMockPublisher(mockCtrl)
	mockWalletTransactionAsync := mockPublisher.NewMockPublisher(mockCtrl)
	mockNotificationPublisher := mock.NewMockTransactionNotificationPublisher(mockCtrl)
	mockFlagClient := mock3.NewMockClient(mockCtrl)
	mockQueueUnicornClient := mockQueueUnicorn.NewMockClient(mockCtrl)

	mockSQLRepository.EXPECT().GetAccountBalanceDailyRepository().Return(mockABCRepo).AnyTimes()
	mockSQLRepository.EXPECT().GetAccountRepository().Return(mockAccRepo).AnyTimes()

	svc := services.New(
		cfg,
		mockSQLRepository,
		mockCacheRepository,
		mockCloudStorageRepository,
		mockConsumer,
		mockAcuanClient,
		mockIDGenerator,
		mockFileRepo,
		mockMasterDataRepo,
		mockDDDNotification,
		mockQueueUnicornClient,
		mockReconPublisher,
		mockBalanceHVTPublisher,
		mockNotificationPublisher,
		mockWalletTransactionAsync,
		mockAccountingClient,
		mockFlagClient,
		nil,
	)
	sut := services.NewReconBalanceService(svc)

	yesterday := common.YesterdayTime()
	gcsPayload := &models.CloudStoragePayload{
		Filename: fmt.Sprintf("%d%02d%02d.csv", yesterday.Year(), yesterday.Month(), yesterday.Day()),
		Path:     fmt.Sprintf("%s/%d/%d", models.BalanceReconReportName, yesterday.Year(), yesterday.Month()),
	}

	return reconSUT{
		sut: sut,

		mockStorageRepo:           mockCloudStorageRepository,
		mockSQLRepo:               mockSQLRepository,
		mockABDRepo:               mockABCRepo,
		mockAccRepo:               mockAccRepo,
		mockReconToolHistoryRepo:  mockReconToolHistoryRepo,
		mockTransactionRepository: mockTransactionRepository,

		mockFileRepo:        mockFileRepo,
		mockDDDNotification: mockDDDNotification,
		mockReconPub:        mockReconPublisher,

		mockConsumer: mockConsumer,

		reportDate: yesterday,
		gcsPayload: gcsPayload,
	}
}

func Test_ReconService_DoDailyBalance(t *testing.T) {
	reconSUT := initReconSUT(t)
	type ServiceArgs struct {
		ctx context.Context
	}
	type MockData struct {
		lastDailyBalance *models.AccountBalanceDaily
		dailyBalances    *[]models.AccountBalanceDaily
		accounts         *[]models.Account
	}

	defaultMockData := MockData{
		lastDailyBalance: &models.AccountBalanceDaily{
			AccountNumber: "",
			Date:          &time.Time{},
			Balance:       decimal.Decimal{},
		},
		dailyBalances: &[]models.AccountBalanceDaily{},
		accounts: &[]models.Account{{
			AccountNumber:  "[TEST] acc1",
			ActualBalance:  decimal.NewFromInt(1),
			PendingBalance: decimal.Zero,
		}},
	}
	tests := []struct {
		name     string
		args     ServiceArgs
		mockData MockData
		doMock   func(args ServiceArgs, data MockData)
		wantErr  bool
	}{
		{
			name: "failed - IsObjectExist exist",
			args: ServiceArgs{ctx: context.Background()},
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(true, "[TEST]")
			},
			wantErr: true,
		},
		{
			name: "failed - GetLast err",
			args: ServiceArgs{ctx: context.Background()},
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name:     "failed - ListByDate err",
			args:     ServiceArgs{ctx: context.Background()},
			mockData: defaultMockData,
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name:     "failed - GetAllWithoutPagination err",
			args:     ServiceArgs{ctx: context.Background()},
			mockData: defaultMockData,
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(data.dailyBalances, nil)
				reconSUT.mockAccRepo.EXPECT().GetAllWithoutPagination(gomock.AssignableToTypeOf(context.Background())).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name:     "failed - OffsetOldest - ABDCreate err",
			args:     ServiceArgs{ctx: context.Background()},
			mockData: defaultMockData,
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(data.dailyBalances, nil)
				reconSUT.mockAccRepo.EXPECT().GetAllWithoutPagination(gomock.AssignableToTypeOf(context.Background())).Return(data.accounts, nil)
				reconSUT.mockABDRepo.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(&[]models.AccountBalanceDaily{})).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:     "failed - OffsetOldest - no difference",
			args:     ServiceArgs{ctx: context.Background()},
			mockData: defaultMockData,
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(data.dailyBalances, nil)
				reconSUT.mockAccRepo.EXPECT().GetAllWithoutPagination(gomock.AssignableToTypeOf(context.Background())).Return(data.accounts, nil)
				reconSUT.mockABDRepo.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(&[]models.AccountBalanceDaily{})).Return(nil)
				reconSUT.mockConsumer.EXPECT().Consume(gomock.AssignableToTypeOf(time.Time{}), sarama.OffsetOldest, gomock.Any()).Return()
			},
			wantErr: false,
		},
		{
			name:     "failed - OffsetOldest - 1 difference - WriteStream error",
			args:     ServiceArgs{ctx: context.Background()},
			mockData: defaultMockData,
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(data.dailyBalances, nil)
				reconSUT.mockAccRepo.EXPECT().GetAllWithoutPagination(gomock.AssignableToTypeOf(context.Background())).Return(data.accounts, nil)
				reconSUT.mockABDRepo.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(&[]models.AccountBalanceDaily{})).Return(nil)
				reconSUT.mockConsumer.EXPECT().Consume(gomock.AssignableToTypeOf(time.Time{}), sarama.OffsetOldest, gomock.Any()).
					DoAndReturn(func(dateLimit time.Time, initialOffset int64, processor func(transactions []goAcuanLib.Transaction)) {
						account := *data.accounts
						reconSUT.sut.AppendAccountTransactions(account[0].AccountNumber, goAcuanLib.Transaction{SourceAccountId: account[0].AccountNumber})
					})
				reconSUT.mockStorageRepo.EXPECT().WriteStream(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload, gomock.Any()).
					DoAndReturn(func(ctx context.Context, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult {
						chanWrite := make(chan error)
						go func() {
							defer close(chanWrite)
							chanWrite <- assert.AnError
						}()
						writeResult := models.NewWriteStreamResult(chanWrite, "")
						return writeResult
					})
			},
			wantErr: true,
		},
		{
			name:     "success - OffsetOldest - 1 difference",
			args:     ServiceArgs{ctx: context.Background()},
			mockData: defaultMockData,
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(data.dailyBalances, nil)
				reconSUT.mockAccRepo.EXPECT().GetAllWithoutPagination(gomock.AssignableToTypeOf(context.Background())).Return(data.accounts, nil)
				reconSUT.mockABDRepo.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(&[]models.AccountBalanceDaily{})).Return(nil)
				reconSUT.mockConsumer.EXPECT().Consume(gomock.AssignableToTypeOf(time.Time{}), sarama.OffsetOldest, gomock.Any()).
					DoAndReturn(func(dateLimit time.Time, initialOffset int64, processor func(transactions []goAcuanLib.Transaction)) {
						account := *data.accounts
						reconSUT.sut.AppendAccountTransactions(account[0].AccountNumber, goAcuanLib.Transaction{SourceAccountId: account[0].AccountNumber})
					})
				reconSUT.mockStorageRepo.EXPECT().WriteStream(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload, gomock.Any()).
					DoAndReturn(func(ctx context.Context, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult {
						chanWrite := make(chan error)
						go func() {
							defer close(chanWrite)
						}()
						writeResult := models.NewWriteStreamResult(chanWrite, "")
						return writeResult
					})
			},
			wantErr: false,
		},
		{
			name: "success - OffsetNewest - 1 difference",
			args: ServiceArgs{ctx: context.Background()},
			mockData: MockData{
				lastDailyBalance: defaultMockData.lastDailyBalance,
				dailyBalances:    &[]models.AccountBalanceDaily{{AccountNumber: "[TEST] 1"}},
				accounts: &[]models.Account{{
					AccountNumber:  "[TEST] acc1",
					ActualBalance:  decimal.NewFromInt(1),
					PendingBalance: decimal.Zero,
				}},
			},
			doMock: func(args ServiceArgs, data MockData) {
				reconSUT.mockStorageRepo.EXPECT().
					IsObjectExist(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload).
					Return(false, "")
				reconSUT.mockABDRepo.EXPECT().GetLast(gomock.AssignableToTypeOf(context.Background())).Return(data.lastDailyBalance, nil)
				reconSUT.mockABDRepo.EXPECT().ListByDate(gomock.AssignableToTypeOf(context.Background()), *data.lastDailyBalance.Date).Return(data.dailyBalances, nil)
				reconSUT.mockAccRepo.EXPECT().GetAllWithoutPagination(gomock.AssignableToTypeOf(context.Background())).Return(data.accounts, nil)
				reconSUT.mockABDRepo.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(&[]models.AccountBalanceDaily{})).Return(nil)
				reconSUT.mockConsumer.EXPECT().
					Consume(gomock.AssignableToTypeOf(time.Time{}), sarama.OffsetNewest, gomock.Any()).
					DoAndReturn(func(dateLimit time.Time, initialOffset int64, processor func(transactions []goAcuanLib.Transaction)) {
						account := *data.accounts
						reconSUT.sut.AppendAccountTransactions(account[0].AccountNumber, goAcuanLib.Transaction{SourceAccountId: account[0].AccountNumber})
					})
				reconSUT.mockStorageRepo.EXPECT().WriteStream(gomock.AssignableToTypeOf(context.Background()), reconSUT.gcsPayload, gomock.Any()).
					DoAndReturn(func(ctx context.Context, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult {
						chanWrite := make(chan error)
						go func() {
							defer close(chanWrite)
						}()
						writeResult := models.NewWriteStreamResult(chanWrite, "")
						return writeResult
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			_, err := reconSUT.sut.DoDailyBalance(tt.args.ctx)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_ReconService_UploadReconTemplate(t *testing.T) {
	reconSUT := initReconSUT(t)
	reconSUT.mockSQLRepo.EXPECT().GetReconToolHistoryRepository().Return(reconSUT.mockReconToolHistoryRepo).AnyTimes()
	now := common.Now()
	gcsPayload := &models.CloudStoragePayload{
		Filename: fmt.Sprintf("%s.csv", now.Format(common.DateFormatYYYYMMDDHHMMSSWithoutDash)),
		Path:     fmt.Sprintf("%s/upload/%04d/%02d", models.ReconToolFolderName, now.Year(), now.Month()),
	}
	expectedCtx, cancel := context.WithCancel(context.Background())
	defer cancel() // Make sure to defer cancel to release resources

	type ServiceArgs struct {
		req *models.UploadReconFileRequest
	}
	tests := []struct {
		name    string
		args    ServiceArgs
		doMock  func(req *models.UploadReconFileRequest)
		wantErr bool
	}{
		{
			name: "happy path",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "identifier;amount;payment_date;remark"}
					}()
					return resultCh
				})
				tempFile, _ := os.CreateTemp("", "test_mock_gcs")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(tempFile)
				reconSUT.mockReconToolHistoryRepo.EXPECT().Create(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(&models.CreateReconToolHistoryIn{})).Return(&models.ReconToolHistory{}, nil)
				reconSUT.mockReconPub.EXPECT().Publish(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(models.ReconPublisher{})).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - err first line",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
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
			name: "failed - invalid template",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "test"}
					}()
					return resultCh
				})
			},
			wantErr: true,
		},
		{
			name: "failed - err read",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "identifier,amount,payment_date,remark"}
						resultCh <- repositories.StreamReadMultipartFileResult{Err: assert.AnError}
					}()
					return resultCh
				})
				tempFile, _ := os.CreateTemp("", "test_mock_gcs")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(tempFile)
			},
			wantErr: true,
		},
		{
			name: "failed - err create",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "identifier,amount,payment_date,remark"}
					}()
					return resultCh
				})
				tempFile, _ := os.CreateTemp("", "test_mock_gcs")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(tempFile)
				reconSUT.mockReconToolHistoryRepo.EXPECT().Create(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(&models.CreateReconToolHistoryIn{})).Return(nil, assert.AnError)
				reconSUT.mockStorageRepo.EXPECT().DeleteFile(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "failed - err create - err rollback",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "identifier,amount,payment_date,remark"}
					}()
					return resultCh
				})
				tempFile, _ := os.CreateTemp("", "test_mock_gcs")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(tempFile)
				reconSUT.mockReconToolHistoryRepo.EXPECT().Create(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(&models.CreateReconToolHistoryIn{})).Return(nil, assert.AnError)
				reconSUT.mockStorageRepo.EXPECT().DeleteFile(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - err publish",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "identifier,amount,payment_date,remark"}
					}()
					return resultCh
				})
				tempFile, _ := os.CreateTemp("", "test_mock_gcs")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(tempFile)
				reconSUT.mockReconToolHistoryRepo.EXPECT().Create(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(&models.CreateReconToolHistoryIn{})).Return(nil, assert.AnError)
				reconSUT.mockStorageRepo.EXPECT().DeleteFile(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "failed - err publish - err rollback",
			args: ServiceArgs{
				req: &models.UploadReconFileRequest{},
			},
			doMock: func(req *models.UploadReconFileRequest) {
				reconSUT.mockFileRepo.EXPECT().StreamReadMultipartFile(
					gomock.AssignableToTypeOf(expectedCtx),
					req.ReconFile,
				).DoAndReturn(func(ctx context.Context, file *multipart.FileHeader) <-chan repositories.StreamReadMultipartFileResult {
					resultCh := make(chan repositories.StreamReadMultipartFileResult)
					go func() {
						defer close(resultCh)
						resultCh <- repositories.StreamReadMultipartFileResult{Data: "identifier,amount,payment_date,remark"}
					}()
					return resultCh
				})
				created := &models.ReconToolHistory{ID: 99}
				tempFile, _ := os.CreateTemp("", "test_mock_gcs")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(tempFile)
				reconSUT.mockReconToolHistoryRepo.EXPECT().Create(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(&models.CreateReconToolHistoryIn{})).Return(created, nil)
				reconSUT.mockReconPub.EXPECT().Publish(gomock.AssignableToTypeOf(expectedCtx), gomock.AssignableToTypeOf(models.ReconPublisher{})).Return(assert.AnError)
				reconSUT.mockStorageRepo.EXPECT().DeleteFile(gomock.AssignableToTypeOf(expectedCtx), gcsPayload).Return(assert.AnError)
				reconSUT.mockReconToolHistoryRepo.EXPECT().DeleteByID(gomock.AssignableToTypeOf(expectedCtx), fmt.Sprint(created.ID)).Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args.req)
			}

			err := reconSUT.sut.UploadReconTemplate(context.Background(), tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_ReconService_GetListReconHistory(t *testing.T) {
	reconSUT := initReconSUT(t)
	reconSUT.mockSQLRepo.EXPECT().GetReconToolHistoryRepository().Return(reconSUT.mockReconToolHistoryRepo).AnyTimes()

	type args struct {
		ctx  context.Context
		opts models.ReconToolHistoryFilterOptions
	}

	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success get list history",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetList(args.ctx, args.opts).Return([]models.ReconToolHistory{}, nil)
				reconSUT.mockReconToolHistoryRepo.EXPECT().CountAll(args.ctx, args.opts).Return(0, nil)
			},
			wantErr: false,
		},
		{
			name: "failed get list history",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetList(args.ctx, args.opts).Return([]models.ReconToolHistory{}, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed count all history",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetList(args.ctx, args.opts).Return([]models.ReconToolHistory{}, nil)
				reconSUT.mockReconToolHistoryRepo.EXPECT().CountAll(args.ctx, args.opts).Return(0, assert.AnError)
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

			_, _, err := reconSUT.sut.GetListReconHistory(tt.args.ctx, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_ReconService_GetResultFileURL(t *testing.T) {
	reconSUT := initReconSUT(t)
	reconSUT.mockSQLRepo.EXPECT().GetReconToolHistoryRepository().Return(reconSUT.mockReconToolHistoryRepo).AnyTimes()

	type args struct {
		ctx context.Context
		id  uint64
	}

	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success get url file",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(&models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
				}, nil)
				reconSUT.mockStorageRepo.EXPECT().GetSignedURL("my_file.txt", gomock.Any()).Return("http://my_file.txt", nil)
			},
			wantErr: false,
		},
		{
			name: "failed get url file - error from gcs",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(&models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
				}, nil)
				reconSUT.mockStorageRepo.EXPECT().GetSignedURL("my_file.txt", gomock.Any()).Return("", assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed get url file from repo",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed get url file missing result",
			args: args{ctx: context.Background()},
			doMock: func(args args) {
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(&models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					ResultFilePath:   "",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
				}, nil)
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

			_, err := reconSUT.sut.GetResultFileURL(tt.args.ctx, tt.args.id)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
