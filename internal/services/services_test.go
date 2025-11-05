package services_test

import (
	"os"
	"testing"

	mock3 "bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mock4 "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	mock5 "bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics/mock"
	mockQueueUnicorn "bitbucket.org/Amartha/go-fp-transaction/internal/common/queueunicorn/mock"
	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	mockAcuanClient "bitbucket.org/Amartha/go-fp-transaction/internal/common/acuanclient/mock"
	mockDDD "bitbucket.org/Amartha/go-fp-transaction/internal/common/ddd_notification/mock"
	mockIDGenerator "bitbucket.org/Amartha/go-fp-transaction/internal/common/idgenerator/mock"
	mockPublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/publisher/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

type testServiceHelper struct {
	mockCtrl                      *gomock.Controller
	config                        config.Config
	mockSQLRepository             *mock.MockSQLRepository
	mockAccRepository             *mock.MockAccountRepository
	mockBalanceRepository         *mock.MockBalanceRepository
	mockAccBalanceDailyRepository *mock.MockAccountBalanceDailyRepository
	mockCategoryRepository        *mock.MockCategoryRepository
	mockEntityRepository          *mock.MockEntityRepository
	mockSubCategoryRepository     *mock.MockSubCategoryRepository
	mockTrxRepository             *mock.MockTransactionRepository
	mockFeatureRepository         *mock.MockFeatureRepository
	mockWalletTrxRepository       *mock.MockWalletTransactionRepository
	mockCacheRepository           *mock.MockCacheRepository
	mockGcs                       *mock.MockCloudStorageRepository
	mockAcuanClient               *mockAcuanClient.MockAcuanClient
	mockAccountingClient          *mock3.MockClient
	mockIDGenerator               *mockIDGenerator.MockGenerator
	mockFileRepo                  *mock.MockFileRepository
	mockMasterData                *mock.MockMasterDataRepository
	mockDDDNotification           *mockDDD.MockDDDNotification

	mockQueueUnicornClient      *mockQueueUnicorn.MockClient
	mockFlagClient              *mock4.MockClient
	mockTransactionNotification *mock2.MockTransactionNotificationPublisher

	transactionService   services.TransactionService
	accountService       services.AccountService
	balanceService       services.BalanceService
	storageService       services.StorageService
	entityService        services.EntityService
	categoryService      services.CategoryService
	subCategoryService   services.SubCategoryService
	fileService          services.FileService
	dlqProcessorService  services.DLQProcessorService
	masterDataService    services.MasterDataService
	walletAccountService services.WalletAccountService
	walletTrxService     services.WalletTrxService
}

func serviceTestHelper(t *testing.T) testServiceHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSQLRepository := mock.NewMockSQLRepository(mockCtrl)
	mockAccountRepository := mock.NewMockAccountRepository(mockCtrl)
	mockBalanceRepository := mock.NewMockBalanceRepository(mockCtrl)
	mockAccountBalanceDailyRepository := mock.NewMockAccountBalanceDailyRepository(mockCtrl)
	mockCategoryRepository := mock.NewMockCategoryRepository(mockCtrl)
	mockEntityRepository := mock.NewMockEntityRepository(mockCtrl)
	mockSubCategoryRepository := mock.NewMockSubCategoryRepository(mockCtrl)
	mockTransactionRepository := mock.NewMockTransactionRepository(mockCtrl)
	mockWalletTransactionRepository := mock.NewMockWalletTransactionRepository(mockCtrl)
	mockFeatureRepository := mock.NewMockFeatureRepository(mockCtrl)
	mockAccountConfigRepository := mock.NewMockAccountConfigRepository(mockCtrl)

	mockCacheRepository := mock.NewMockCacheRepository(mockCtrl)
	mockCloudStorageRepository := mock.NewMockCloudStorageRepository(mockCtrl)
	mockAcuanClient := mockAcuanClient.NewMockAcuanClient(mockCtrl)
	mockIDGenerator := mockIDGenerator.NewMockGenerator(mockCtrl)
	mockFileRepo := mock.NewMockFileRepository(mockCtrl)
	mockMasterDataRepo := mock.NewMockMasterDataRepository(mockCtrl)
	mockDDDNotification := mockDDD.NewMockDDDNotification(mockCtrl)
	mockQueueUnicornClient := mockQueueUnicorn.NewMockClient(mockCtrl)
	mockReconPublisher := mockPublisher.NewMockPublisher(mockCtrl)
	mockBalanceHVTPub := mockPublisher.NewMockPublisher(mockCtrl)
	mockWalletTransaction := mockPublisher.NewMockPublisher(mockCtrl)
	mockNotificationPublisher := mock2.NewMockTransactionNotificationPublisher(mockCtrl)
	mockAccountingClient := mock3.NewMockClient(mockCtrl)
	mockFlagClient := mock4.NewMockClient(mockCtrl)

	mockMetrics := mock5.NewMockMetrics(mockCtrl)
	mockMetrics.EXPECT().GetBalancePrometheus().Return(nil).AnyTimes()

	mockSQLRepository.EXPECT().GetAccountRepository().Return(mockAccountRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetBalanceRepository().Return(mockBalanceRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetAccountBalanceDailyRepository().Return(mockAccountBalanceDailyRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetCategoryRepository().Return(mockCategoryRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetEntityRepository().Return(mockEntityRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetSubCategoryRepository().Return(mockSubCategoryRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetTransactionRepository().Return(mockTransactionRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetWalletTransactionRepository().Return(mockWalletTransactionRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetFeatureRepository().Return(mockFeatureRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetAccountConfigExternalRepository().Return(mockAccountConfigRepository).AnyTimes()
	mockSQLRepository.EXPECT().GetAccountConfigInternalRepository().Return(mockAccountConfigRepository).AnyTimes()

	conf := config.Config{
		TransactionConfig: config.TransactionConfig{
			BatchSize: 1000,
		},
		AccountConfig: config.AccountConfig{
			AccountNumberPadWidth: 8,
			HVTSubCategoryCodes:   []string{"21103"},
			SystemAccountNumber:   "00000100000000",
		},
		TransactionValidationConfig: config.TransactionValidationConfig{
			AcceptedOrderType:       []string{"INSURANCE", "TOPUP"},
			AcceptedTransactionType: []string{"Insurance Premi", "ACRF"},
		},
		FeatureFlag: config.FeatureFlag{
			EnableConsumerValidationReject: true,
		},
		AccountFeatureConfig: map[string]config.FeatureConfig{
			"customer": {
				BalanceRangeMin:        2000000,
				AllowedNegativeTrxType: []string{"DSBAA", "RPYAA", "TUPVM", "TUPVA"},
				AllowedTrxType:         []string{"DSBAA", "RPYAA", "TUPVM", "TUPVA"},
				NegativeBalanceAllowed: true,
				NegativeLimit:          500000,
			},
			"pocket": {
				BalanceRangeMax:        30000, // Default max balance for pocket accounts
				BalanceRangeMin:        0,
				AllowedNegativeTrxType: []string{},
				AllowedTrxType:         []string{"TUPVA"},
				NegativeBalanceAllowed: false,
				NegativeLimit:          0,
			},
		},
		FeatureFlagKeyLookup: config.FeatureFlagKeyLookup{
			BalanceLimitToggle: "balance_limit_toggle",
		},
	}
	serv := services.New(
		conf,
		mockSQLRepository,
		mockCacheRepository,
		mockCloudStorageRepository,
		nil, // Removed mockConsumer - not needed
		mockAcuanClient,
		mockIDGenerator,
		mockFileRepo,
		mockMasterDataRepo,
		mockDDDNotification,
		mockQueueUnicornClient,
		mockReconPublisher,
		mockBalanceHVTPub,
		mockNotificationPublisher,
		mockWalletTransaction,
		mockAccountingClient,
		mockFlagClient,
		mockMetrics,
	)

	return testServiceHelper{
		mockCtrl:                      mockCtrl,
		config:                        conf,
		mockSQLRepository:             mockSQLRepository,
		mockAccRepository:             mockAccountRepository,
		mockBalanceRepository:         mockBalanceRepository,
		mockAccBalanceDailyRepository: mockAccountBalanceDailyRepository,
		mockCategoryRepository:        mockCategoryRepository,
		mockEntityRepository:          mockEntityRepository,
		mockSubCategoryRepository:     mockSubCategoryRepository,
		mockTrxRepository:             mockTransactionRepository,
		mockWalletTrxRepository:       mockWalletTransactionRepository,
		mockFeatureRepository:         mockFeatureRepository,
		mockFileRepo:                  mockFileRepo,

		mockMasterData:              mockMasterDataRepo,
		mockCacheRepository:         mockCacheRepository,
		mockGcs:                     mockCloudStorageRepository,
		mockAcuanClient:             mockAcuanClient,
		mockAccountingClient:        mockAccountingClient,
		mockIDGenerator:             mockIDGenerator,
		mockDDDNotification:         mockDDDNotification,
		mockQueueUnicornClient:      mockQueueUnicornClient,
		mockFlagClient:              mockFlagClient,
		mockTransactionNotification: mockNotificationPublisher,

		transactionService:   serv.Transaction,
		accountService:       serv.Account,
		balanceService:       serv.Balance,
		storageService:       serv.Storage,
		entityService:        serv.Entity,
		categoryService:      serv.Category,
		subCategoryService:   serv.SubCategory,
		fileService:          serv.File,
		dlqProcessorService:  serv.DLQProcessor,
		masterDataService:    serv.MasterData,
		walletAccountService: serv.WalletAccount,
		walletTrxService:     serv.WalletTrx,
	}
}
