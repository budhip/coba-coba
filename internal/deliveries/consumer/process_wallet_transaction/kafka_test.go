package process_wallet_transaction

import (
	"context"
	"os"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"

	publisherMock "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	messagingMock "bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging/mock"
	kafkaMock "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"
	repositoryMock "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"
	serviceMock "bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

type kafkaTestHelper struct {
	mockCtrl      *gomock.Controller
	group         string
	topic         string
	broker        *sarama.MockBroker
	defaultConfig config.Config

	dlq                      dlqpublisher.Publisher
	cacheRepo                repositories.CacheRepository
	walletTransactionService services.WalletTrxService

	cg *kafkaMock.MockConsumerGroup
}

func (th kafkaTestHelper) close() {
	th.broker.Close()
	th.mockCtrl.Finish()
}

func newKafkaTestHelper(t *testing.T) kafkaTestHelper {
	t.Helper()
	t.Parallel()

	var (
		group = "go-fp-transaction"
		topic = "test"
	)

	mockCtrl := gomock.NewController(t)
	broker := messagingMock.NewMockBroker(t, group, topic)

	return kafkaTestHelper{
		mockCtrl: mockCtrl,
		group:    group,
		topic:    topic,
		broker:   broker,
		defaultConfig: config.Config{
			App: config.App{
				Env:  "test",
				Name: "go-fp-transaction",
			},
			MessageBroker: config.MessageBroker{
				KafkaConsumer: config.ConsumerConfig{
					Brokers:       []string{broker.Addr()},
					Topic:         topic,
					ConsumerGroup: group,
				},
			},
		},
		dlq:                      publisherMock.NewMockPublisher(mockCtrl),
		cacheRepo:                repositoryMock.NewMockCacheRepository(mockCtrl),
		walletTransactionService: serviceMock.NewMockWalletTrxService(mockCtrl),
		cg:                       kafkaMock.NewMockConsumerGroup(mockCtrl),
	}
}

func TestNew(t *testing.T) {

	th := newKafkaTestHelper(t)
	defer th.close()

	type args struct {
		ctx                      context.Context
		cfg                      config.Config
		metrics                  metrics.Metrics
		dlq                      dlqpublisher.Publisher
		cacheRepo                repositories.CacheRepository
		walletTransactionService services.WalletTrxService
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success new client",
			args: args{
				cfg: config.Config{
					App: config.App{
						Env:  "test",
						Name: "go-fp-transaction",
					},
					MessageBroker: config.MessageBroker{
						KafkaConsumer: config.ConsumerConfig{
							Brokers:       []string{th.broker.Addr()},
							Topic:         th.topic,
							ConsumerGroup: th.group,
						},
					},
				},
				dlq:                      th.dlq,
				cacheRepo:                th.cacheRepo,
				walletTransactionService: th.walletTransactionService,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(
				tt.args.ctx,
				tt.args.cfg,
				nil,
				tt.args.cacheRepo,
				tt.args.walletTransactionService,
				tt.args.dlq,
			)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestConsumer_Start(t *testing.T) {
	th := newKafkaTestHelper(t)
	defer th.close()

	type fields struct {
		ctx         context.Context
		ctxCancel   context.CancelFunc
		cfg         config.Config
		consumerCfg config.ConsumerConfig
		cg          *kafkaMock.MockConsumerGroup

		cacheRepo                repositories.CacheRepository
		walletTransactionService services.WalletTrxService
	}

	tests := []struct {
		name   string
		fields fields
		doMock func(f fields)
	}{
		{
			name: "success start",
			fields: fields{
				cfg:                      th.defaultConfig,
				consumerCfg:              th.defaultConfig.MessageBroker.KafkaConsumer,
				cg:                       kafkaMock.NewMockConsumerGroup(th.mockCtrl),
				cacheRepo:                th.cacheRepo,
				walletTransactionService: th.walletTransactionService,
			},
			doMock: func(f fields) {
				chanErr := make(chan error)
				f.cg.EXPECT().Errors().Return(chanErr).AnyTimes()
				f.cg.EXPECT().Consume(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
		},
		{
			name: "failed preStart() error config",
			fields: fields{
				cfg:                      th.defaultConfig,
				consumerCfg:              config.ConsumerConfig{},
				cg:                       kafkaMock.NewMockConsumerGroup(th.mockCtrl),
				cacheRepo:                th.cacheRepo,
				walletTransactionService: th.walletTransactionService,
			},
			doMock: func(f fields) {
			},
		},
		{
			name: "error consume message",
			fields: fields{
				cfg:                      th.defaultConfig,
				consumerCfg:              th.defaultConfig.MessageBroker.KafkaConsumer,
				cg:                       kafkaMock.NewMockConsumerGroup(th.mockCtrl),
				cacheRepo:                th.cacheRepo,
				walletTransactionService: th.walletTransactionService,
			},
			doMock: func(f fields) {
				chanErr := make(chan error, 1)
				chanErr <- assert.AnError
				f.cg.EXPECT().Errors().Return(chanErr).AnyTimes()
				f.cg.EXPECT().Consume(gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.ctx, tt.fields.ctxCancel = context.WithTimeout(context.Background(), 1*time.Second)
			defer tt.fields.ctxCancel()

			if tt.doMock != nil {
				tt.doMock(tt.fields)
			}

			consumer := &Consumer{
				ctx:                      tt.fields.ctx,
				clientID:                 th.group,
				cfg:                      tt.fields.cfg,
				cg:                       tt.fields.cg,
				cacheRepo:                tt.fields.cacheRepo,
				walletTransactionService: th.walletTransactionService,
			}

			consumer.Start()
		})
	}
}

func TestConsumer_Stop(t *testing.T) {
	th := newKafkaTestHelper(t)
	defer th.close()

	type fields struct {
		ctx       context.Context
		ctxCancel context.CancelFunc
		cg        *kafkaMock.MockConsumerGroup
	}

	tests := []struct {
		name   string
		fields fields
		doMock func(f fields)
	}{
		{
			name: "success stop consumer",
			fields: fields{
				cg: kafkaMock.NewMockConsumerGroup(th.mockCtrl),
			},
		},
		{
			name: "error stop consumer",
			fields: fields{
				cg: kafkaMock.NewMockConsumerGroup(th.mockCtrl),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.ctx, tt.fields.ctxCancel = context.WithTimeout(context.Background(), 1*time.Second)
			defer tt.fields.ctxCancel()

			if tt.doMock != nil {
				tt.doMock(tt.fields)
			}

			consumer := &Consumer{
				ctx:      tt.fields.ctx,
				clientID: th.group,
				cg:       tt.fields.cg,
			}

			consumer.Stop()
		})
	}
}
