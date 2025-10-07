package services

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/acuanclient"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/ddd_notification"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/idgenerator"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/mapper"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/queueunicorn"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	kafkaRecon "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka_recon"
)

type service struct {
	srv *Services
}

type Services struct {
	conf config.Config

	sqlRepo        repositories.SQLRepository
	cacheRepo      repositories.CacheRepository
	cloudStorage   repositories.CloudStorageRepository
	fileRepo       repositories.FileRepository
	masterDataRepo repositories.MasterDataRepository
	featureRepo    repositories.FeatureRepository

	reconPub                publisher.Publisher
	balanceHVTPub           publisher.Publisher
	transactionNotification transaction_notification.TransactionNotificationPublisher
	walletTransactionAsync  publisher.Publisher

	consumerRecon      kafkaRecon.Consumer
	acuanClient        acuanclient.AcuanClient
	idgenerator        idgenerator.Generator
	dddNotification    ddd_notification.DDDNotification
	queueUnicornClient queueunicorn.Client
	accountingClient   accounting.Client
	accountMapper      mapper.AccountMapper
	flag               flag.Client
	metrics            metrics.Metrics

	common service

	Account       *account
	Balance       *balance
	MoneyFlowCalc *moneyFlowCalc
	Transaction   *transaction
	Storage       *storage
	Entity        *entity
	Category      *category
	SubCategory   *subCategory
	File          *file
	DLQProcessor  *dlqProcessor
	MasterData    *masterData
	Recon         *reconService
	WalletAccount *walletAccount
	WalletTrx     *walletTrx
}

func New(
	conf config.Config,
	sqlRepo repositories.SQLRepository,
	cacheRepo repositories.CacheRepository,
	cloudStorage repositories.CloudStorageRepository,
	consumerRecon kafkaRecon.Consumer,
	acuanClient acuanclient.AcuanClient,
	idgenerator idgenerator.Generator,
	fileRepo repositories.FileRepository,
	masterDataRepo repositories.MasterDataRepository,
	dddNotification ddd_notification.DDDNotification,
	queueUnicornClient queueunicorn.Client,
	reconPub publisher.Publisher,
	balanceHVTPub publisher.Publisher,
	transactionNotification transaction_notification.TransactionNotificationPublisher,
	walletTransactionAsync publisher.Publisher,
	accountingClient accounting.Client,
	flag flag.Client,
	metrics metrics.Metrics,
) *Services {
	srv := &Services{
		conf:                    conf,
		sqlRepo:                 sqlRepo,
		cacheRepo:               cacheRepo,
		cloudStorage:            cloudStorage,
		consumerRecon:           consumerRecon,
		acuanClient:             acuanClient,
		idgenerator:             idgenerator,
		fileRepo:                fileRepo,
		masterDataRepo:          masterDataRepo,
		dddNotification:         dddNotification,
		queueUnicornClient:      queueUnicornClient,
		accountingClient:        accountingClient,
		reconPub:                reconPub,
		balanceHVTPub:           balanceHVTPub,
		transactionNotification: transactionNotification,
		walletTransactionAsync:  walletTransactionAsync,
		flag:                    flag,
		metrics:                 metrics,
	}
	srv.common.srv = srv
	srv.Account = (*account)(&srv.common)
	srv.Balance = (*balance)(&srv.common)
	srv.MoneyFlowCalc = (*moneyFlowCalc)(&srv.common)
	srv.Transaction = (*transaction)(&srv.common)
	srv.Storage = (*storage)(&srv.common)
	srv.Entity = (*entity)(&srv.common)
	srv.Category = (*category)(&srv.common)
	srv.SubCategory = (*subCategory)(&srv.common)
	srv.File = (*file)(&srv.common)
	srv.DLQProcessor = (*dlqProcessor)(&srv.common)
	srv.MasterData = (*masterData)(&srv.common)
	srv.WalletAccount = (*walletAccount)(&srv.common)
	srv.WalletTrx = (*walletTrx)(&srv.common)

	return srv
}
