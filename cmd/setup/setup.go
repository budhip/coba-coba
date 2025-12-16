package setup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"time"

	"golang.org/x/exp/slices"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/acuanclient"
	genericCache "bitbucket.org/Amartha/go-fp-transaction/internal/common/cache"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/ddd_notification"
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/idgenerator"
	cMetrics "bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/queueunicorn"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	kafkaRecon "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka_recon"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	confLoader "bitbucket.org/Amartha/go-config-loader-library"
	xlog "bitbucket.org/Amartha/go-x/log"

	"cloud.google.com/go/compute/metadata"
	"github.com/newrelic/go-agent/v3/integrations/nrzap"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/go-redis/v9"

	_ "github.com/newrelic/go-agent/v3/integrations/nrpgx"
)

type Setup struct {
	Config           config.Config
	NewRelic         *newrelic.Application
	WriteDB          *sql.DB
	ReadDB           *sql.DB
	Cache            *redis.Client
	RepoCache        repositories.CacheRepository
	RepoCloudStorage repositories.CloudStorageRepository
	Service          *services.Services
	PublisherClient  *PublisherClient
	Metrics          cMetrics.Metrics
}

func Init(command string) (setup *Setup, stopper []graceful.ProcessStopper, err error) {
	ctx := context.Background()

	var cfg config.Config
	l := confLoader.New(
		"GO_FP_TRANSACTION",
		"",
		os.Getenv(""),
		confLoader.WithConfigFileName("config"),
		confLoader.WithConfigFileSearchPaths("/config", "."),
		confLoader.WithConfigFileSearchPaths("./config"), // tambahin ini, tapi jangan di commit ke master.
	)
	err = l.Load(&cfg)
	if err != nil {
		return
	}

	setup = &Setup{
		Config: cfg,
	}

	logLevel := xlog.DebugLogLevel()
	excludedDebugLevelOnEnvs := []config.Environment{
		config.DEV_ENV,
		config.UAT_ENV,
		config.PROD_ENV,
	}

	if slices.Contains(excludedDebugLevelOnEnvs, config.StringToEnvironment(cfg.App.Env)) {
		logLevel = xlog.InfoLogLevel()
	}

	xlog.Init(cfg.App.Name,
		xlog.WithLogToOption(cfg.App.LogOption),
		xlog.WithLogEnvOption(cfg.App.Env),
		xlog.WithCaller(true),
		xlog.AddCallerSkip(2),
		logLevel)

	stopper = append(stopper, func(ctx context.Context) error {
		xlog.Sync()
		return nil
	})

	projectID := cfg.GcloudProjectID
	if projectID == "" {
		projectID, _ = metadata.ProjectID()
		cfg.GcloudProjectID = projectID
		xlog.Info(ctx, "can not determine google cloud project, for local use set the gcloud_project_id in config yaml")
	}

	newRelic := setupNR(ctx, cfg)

	// metrics
	mtc := cMetrics.New()

	// connect to db master
	writeDB, readDB, err := setupPostgres(cfg)
	if err != nil {
		err = fmt.Errorf("failed connect to database: %w", err)
		return
	}
	stopper = append(stopper, func(ctx context.Context) error {
		var errs error

		if writeDB != nil {
			if err := writeDB.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to close writeDB: %w", err))
			}
		}

		if readDB != nil {
			if err := readDB.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to close readDB: %w", err))
			}
		}

		return errs
	})

	// connect to redis
	cache := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Db,
	})
	_, err = cache.Ping(ctx).Result()
	if err != nil {
		return
	}
	stopper = append(stopper, func(ctx context.Context) error { return cache.Close() })

	flagClient, err := flag.New(&cfg)
	if err != nil {
		err = fmt.Errorf("failed to create flag client: %w", err)
		return
	}
	stopper = append(stopper, func(ctx context.Context) error { return flagClient.Close() })

	if mtc != nil {
		// register DB write stat prometheus metrics
		err = mtc.RegisterDB(writeDB, cfg.App.Name+"-"+command+"-write", cfg.Postgres.Write.DbName)
		if err != nil {
			err = fmt.Errorf("failed register DB stat prometheus: %w", err)
			return
		}
		// register DB read stat prometheus metrics
		err = mtc.RegisterDB(readDB, cfg.App.Name+"-"+command+"-read", cfg.Postgres.Read.DbName)
		if err != nil {
			err = fmt.Errorf("failed register DB stat prometheus: %w", err)
			return
		}

		// register redis prometheus metrics
		err = mtc.RegisterRedis(cache, cfg.App.Name, command)
		if err != nil {
			err = fmt.Errorf("failed register redis prometheus: %w", err)
			return
		}
	}

	cacheAccounting := genericCache.NewInMemoryClient[string]()
	stopper = append(stopper, func(ctx context.Context) error {
		cacheAccounting.Close()
		return nil
	})

	cacheListAccounting := genericCache.NewInMemoryClient[accounting.ResponseGetListAccountNumber]()
	stopper = append(stopper, func(ctx context.Context) error {
		cacheListAccounting.Close()
		return nil
	})

	accountingClient := accounting.New(cfg.GoAccounting, mtc, cacheAccounting, cacheListAccounting)

	// register repository
	sqlRepo := repositories.NewSQLRepository(writeDB, readDB, cfg, flagClient, accountingClient)
	cacheRepo := repositories.NewCacheRepository(cache)

	masterDataRepo, err := repositories.NewGCSMasterDataRepository(&cfg)
	if err != nil {
		err = fmt.Errorf("failed connect to gcs master data: %w", err)
		return
	}

	masterDataRepo.RefreshDataPeriodically(ctx, time.Minute)

	cloudStorageRepo, err := repositories.NewCloudStorageRepository(&cfg)
	if err != nil {
		return
	}
	stopper = append(stopper, func(ctx context.Context) error { return cloudStorageRepo.Close() })

	consumer, err := kafkaRecon.NewBalanceReconConsumer(context.Background(), cfg)
	if err != nil {
		err = fmt.Errorf("unable to create BalanceReconConsumer: %w", err)
		xlog.Errorf(ctx, "unable to create BalanceReconConsumer: %v", err)
		return
	}

	acuanClient, err := acuanclient.New(cfg)
	if err != nil {
		return
	}

	idGenerator := idgenerator.New()
	fileRepo := repositories.NewFileRepository()

	dddNotification := ddd_notification.New(cfg)
	queueUnicornClient, err := queueunicorn.New(cfg)
	if err != nil {
		err = fmt.Errorf("unable to create client go_queue_unicorn: %w", err)
		return
	}

	producer, err := publisher.NewKafkaSyncProducer(cfg.MessageBroker.KafkaConsumer.Brokers)
	if err != nil {
		err = fmt.Errorf("unable to create create client kafka sync producer: %w", err)
		return
	}
	stopper = append(stopper, func(ctx context.Context) error { return producer.Close() })

	reconPub := publisher.NewPublisher(producer, cfg.MessageBroker.KafkaConsumer.TopicRecon)

	balanceProducer, err := publisher.NewKafkaSyncProducer(
		cfg.MessageBroker.KafkaConsumer.Brokers,
		publisher.WithCustomHasher(fnv.New32a),
	)
	if err != nil {
		err = fmt.Errorf("unable to create create client kafka sync producer: %w", err)
		return
	}
	stopper = append(stopper, func(ctx context.Context) error { return balanceProducer.Close() })

	balanceHVTPub := publisher.NewPublisher(balanceProducer, cfg.MessageBroker.KafkaConsumer.TopicBalanceHVT)

	walletTransactionAsync := publisher.NewPublisher(producer, cfg.MessageBroker.KafkaConsumer.TopicProcessWalletTransaction)

	publisherClient := PublisherClient{
		TransactionNotification: transaction_notification.NewTransactionNotificationPublisher(
			cfg,
			producer,
			balanceProducer,
			mtc),
		TransactionDQL: dlqpublisher.New(producer, cfg.MessageBroker.KafkaConsumer.TopicDLQ, mtc),
	}

	// register service
	srv := services.New(
		cfg,
		sqlRepo,
		cacheRepo,
		cloudStorageRepo,
		consumer,
		acuanClient,
		idGenerator,
		fileRepo,
		masterDataRepo,
		dddNotification,
		queueUnicornClient,
		reconPub,
		balanceHVTPub,
		publisherClient.TransactionNotification,
		walletTransactionAsync,
		accountingClient,
		flagClient,
		mtc,
	)

	return &Setup{
		Config:           cfg,
		NewRelic:         newRelic,
		WriteDB:          writeDB,
		ReadDB:           readDB,
		Cache:            cache,
		Service:          srv,
		RepoCache:        cacheRepo,
		RepoCloudStorage: cloudStorageRepo,
		PublisherClient:  &publisherClient,
		Metrics:          mtc,
	}, stopper, nil
}

func setupPostgres(conf config.Config) (*sql.DB, *sql.DB, error) {
	writeDB, err := initDB(conf.Postgres.Write)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init write DB: %w", err)
	}

	readDB, err := initDB(conf.Postgres.Read)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init read DB: %w", err)
	}

	return writeDB, readDB, nil
}

func initDB(pgConf config.Database) (*sql.DB, error) {
	const (
		DefaultMaxOpen     = 10
		DefaultMaxIdle     = 10
		DefaultMaxLifetime = 3 // minutes
	)

	dsName := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s search_path=%s sslmode=disable",
		pgConf.DbHost, pgConf.DbPort, pgConf.DbUser, pgConf.DbPass, pgConf.DbName, pgConf.DbSchema,
	)

	db, err := sql.Open("nrpgx", dsName)
	if err != nil {
		return nil, err
	}

	if pgConf.MaxOpenConnection > 0 {
		db.SetMaxOpenConns(pgConf.MaxOpenConnection)
	} else {
		db.SetMaxOpenConns(DefaultMaxOpen)
	}

	if pgConf.MaxIdleConnection > 0 {
		db.SetMaxIdleConns(pgConf.MaxIdleConnection)
	} else {
		db.SetMaxIdleConns(DefaultMaxIdle)
	}

	if pgConf.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(pgConf.ConnMaxLifetime) * time.Minute)
	} else {
		db.SetConnMaxLifetime(time.Duration(DefaultMaxLifetime) * time.Minute)
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func setupNR(ctx context.Context, cfg config.Config) *newrelic.Application {
	if env := config.StringToEnvironment(cfg.App.Env); env == config.PROD_ENV {
		logger, ok := xlog.Loggers.Load(xlog.DefaultLogger)
		if !ok {
			return nil
		}
		app, err := newrelic.NewApplication(
			newrelic.ConfigAppName(cfg.App.Name),
			newrelic.ConfigLicense(cfg.NewRelicLicenseKey),
			func(config *newrelic.Config) {
				config.Logger = nrzap.Transform(logger)
			},
			newrelic.ConfigDistributedTracerEnabled(true),
		)
		if err != nil {
			xlog.Errorf(ctx, "setupNR.NewApplication - %v", err)
		}
		if err = app.WaitForConnection(15 * time.Second); nil != err {
			xlog.Errorf(ctx, "setupNR.WaitForConnection - %v", err)
		}
		return app
	}
	return nil
}
