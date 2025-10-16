package money_flow_calc

import (
	"context"
	"os"
	"testing"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	mock4 "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	mock3 "bitbucket.org/Amartha/go-fp-transaction/internal/common/messaging/mock"
	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

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

	mfs services.MoneyFlowService
	dlq dlqpublisher.Publisher
	cg  *mock.MockConsumerGroup
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

	broker := mock3.NewMockBroker(t, group, topic)
	cg := mock.NewMockConsumerGroup(mockCtrl)
	mfs := mock2.NewMockMoneyFlowService(mockCtrl)
	dlq := mock4.NewMockPublisher(mockCtrl)

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
		mfs: mfs,
		dlq: dlq,
		cg:  cg,
	}
}

func TestNew(t *testing.T) {
	th := newKafkaTestHelper(t)
	defer th.close()

	type args struct {
		ctx context.Context
		cfg config.Config
		mfs services.MoneyFlowService
		dlq dlqpublisher.Publisher
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
				mfs: th.mfs,
				dlq: th.dlq,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.ctx, tt.args.cfg, tt.args.mfs, tt.args.dlq, nil)
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
		cg          *mock.MockConsumerGroup
		mfs         services.MoneyFlowService
		dlq         dlqpublisher.Publisher
	}

	tests := []struct {
		name   string
		fields fields
		doMock func(f fields)
	}{
		{
			name: "success start",
			fields: fields{
				cfg:         th.defaultConfig,
				consumerCfg: th.defaultConfig.MessageBroker.KafkaConsumer,
				cg:          mock.NewMockConsumerGroup(th.mockCtrl),
				mfs:         th.mfs,
				dlq:         th.dlq,
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
				cfg:         th.defaultConfig,
				consumerCfg: config.ConsumerConfig{},
				cg:          mock.NewMockConsumerGroup(th.mockCtrl),
				mfs:         th.mfs,
				dlq:         th.dlq,
			},
			doMock: func(f fields) {
			},
		},
		{
			name: "error consume message",
			fields: fields{
				cfg:         th.defaultConfig,
				consumerCfg: th.defaultConfig.MessageBroker.KafkaConsumer,
				cg:          mock.NewMockConsumerGroup(th.mockCtrl),
				mfs:         th.mfs,
				dlq:         th.dlq,
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
				ctx:      tt.fields.ctx,
				clientID: th.group,
				cfg:      tt.fields.cfg,
				cg:       tt.fields.cg,
				mfs:      tt.fields.mfs,
				dlq:      tt.fields.dlq,
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
		cg        *mock.MockConsumerGroup
	}

	tests := []struct {
		name   string
		fields fields
		doMock func(f fields)
	}{
		{
			name: "success stop consumer",
			fields: fields{
				cg: mock.NewMockConsumerGroup(th.mockCtrl),
			},
		},
		{
			name: "error stop consumer",
			fields: fields{
				cg: mock.NewMockConsumerGroup(th.mockCtrl),
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
