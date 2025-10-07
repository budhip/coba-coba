package hvtbalanceupdate

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	publisherMock "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	kafkaMock "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	repositoryMock "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type hvtBalanceUpdateHandlerHelper struct {
	mockCtrl *gomock.Controller
	bs       *mock.MockBalanceService

	cacheRepo *repositoryMock.MockCacheRepository
	dlq       *publisherMock.MockPublisher
	payload   []byte
}

func newHvtBalanceUpdateHandlerHelper(t *testing.T) hvtBalanceUpdateHandlerHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	bs := mock.NewMockBalanceService(mockCtrl)
	cacheRepo := repositoryMock.NewMockCacheRepository(mockCtrl)
	dlq := publisherMock.NewMockPublisher(mockCtrl)
	payload := []byte(`{
	"accountNumber": "211001000331186",
	"updateAmount": {
		"value": 9999.999,
		"currency": "IDR"
		}
	}`)

	return hvtBalanceUpdateHandlerHelper{
		mockCtrl:  mockCtrl,
		bs:        bs,
		cacheRepo: cacheRepo,
		payload:   payload,
		dlq:       dlq,
	}
}

func TestNewHvtBalanceUpdateHandler(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	cfg := &config.Config{
		FeatureFlag: config.FeatureFlag{},
	}

	type args struct {
		cfg       *config.Config
		bs        services.BalanceService
		cacheRepo repositories.CacheRepository
		dlq       dlqpublisher.Publisher
	}

	tests := []struct {
		name string
		args args
		want *HvtBalanceHandler
	}{
		{
			name: "success init HvtBalanceHandler",
			args: args{
				bs:        hh.bs,
				cfg:       cfg,
				cacheRepo: hh.cacheRepo,
				dlq:       hh.dlq,
			},
			want: &HvtBalanceHandler{
				clientId:        "",
				bs:              hh.bs,
				cacheRepo:       hh.cacheRepo,
				dlq:             hh.dlq,
				featureFlag:     cfg.FeatureFlag,
				consumerMetrics: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t,
				tt.want,
				NewHvtBalanceHandler(
					"",
					tt.args.bs,
					tt.args.cacheRepo,
					tt.args.dlq,
					tt.args.cfg.FeatureFlag,
					nil),
				"NewHvtBalanceHandler(%v)", tt.args.bs)
		})
	}
}

func TestHvtBalanceUpdateHandler_Cleanup(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	type fields struct {
		bs  services.BalanceService
		dlq dlqpublisher.Publisher
	}
	type args struct {
		session sarama.ConsumerGroupSession
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success cleanup",
			fields: fields{
				bs:  hh.bs,
				dlq: hh.dlq,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hh := HvtBalanceHandler{
				bs: tt.fields.bs,
			}
			err := hh.Cleanup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}

}

func TestHvtBalanceUpdateHandler_processMessage(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	type fields struct {
		bs services.BalanceService
	}
	type args struct {
		message *sarama.ConsumerMessage
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				bs: hh.bs,
			},
			args: args{
				message: &sarama.ConsumerMessage{Value: hh.payload},
			},
			doMock: func() {
				hh.bs.EXPECT().AdjustAccountBalance(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.Any(),
					gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - err service",
			fields: fields{
				bs: hh.bs,
			},
			args: args{
				message: &sarama.ConsumerMessage{Value: hh.payload},
			},
			doMock: func() {
				hh.bs.EXPECT().AdjustAccountBalance(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.Any(),
					gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			var payload models.UpdateBalanceHVTPayload
			err := json.Unmarshal(tt.args.message.Value, &payload)
			assert.NoError(t, err)

			h := HvtBalanceHandler{
				bs: tt.fields.bs,
			}
			err = h.processMessage(context.Background(), tt.args.message, &payload)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestHvtBalanceUpdateHandler_Setup(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	type fields struct {
		bs services.BalanceService
	}
	type args struct {
		session sarama.ConsumerGroupSession
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success Setup",
			fields: fields{
				bs: hh.bs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := HvtBalanceHandler{
				bs: tt.fields.bs,
			}
			err := th.Setup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestHvtBalanceUpdateHandler_Ack(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	mockSess := kafkaMock.NewMockConsumerGroupSession(hh.mockCtrl)

	type fields struct {
		bs services.BalanceService
	}
	type args struct {
		ctx     context.Context
		session sarama.ConsumerGroupSession
		payload *ackPayload
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		doMock func()
	}{
		{
			name: "success Ack message",
			fields: fields{
				bs: hh.bs,
			},
			args: args{
				ctx:     context.TODO(),
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
			},
			doMock: func() {
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			th := HvtBalanceHandler{
				clientId:        "",
				bs:              tt.fields.bs,
				consumerMetrics: nil,
			}
			th.Ack(tt.args.session, tt.args.payload)
		})
	}
}

func TestHvtBalanceUpdateHandler_Nack(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	mockSess := kafkaMock.NewMockConsumerGroupSession(hh.mockCtrl)

	type fields struct {
		bs          services.BalanceService
		dlq         dlqpublisher.Publisher
		cacheRepo   repositories.CacheRepository
		featureFlag config.FeatureFlag
	}
	type args struct {
		ctx        context.Context
		session    sarama.ConsumerGroupSession
		payload    *ackPayload
		consumeErr error
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		doMock func(a args)
	}{
		{
			name: "success Nack message - dlq enabled",
			fields: fields{
				bs:        hh.bs,
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
				consumeErr: assert.AnError,
			},
			doMock: func(a args) {
				hh.cacheRepo.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)
				hh.dlq.EXPECT().Publish(gomock.Any()).Return(nil)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "success Nack message - dlq disabled",
			fields: fields{
				bs:        hh.bs,
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: false,
				},
			},
			args: args{
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
				consumeErr: assert.AnError,
			},
			doMock: func(a args) {
				hh.cacheRepo.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "Success Nack message - no send to DLQ (idempotency key existed)",
			fields: fields{
				bs:        hh.bs,
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
				consumeErr: common.ErrRequestBeingProcessed,
			},
			doMock: func(a args) {
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "Success Nack message but error while deleting idempotency key",
			fields: fields{
				bs:        hh.bs,
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
				consumeErr: assert.AnError,
			},
			doMock: func(a args) {
				hh.cacheRepo.EXPECT().Del(gomock.Any(), gomock.Any()).Return(assert.AnError)
				hh.dlq.EXPECT().Publish(gomock.Any()).Return(nil)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "Success Nack message but publish dlq failed",
			fields: fields{
				bs:        hh.bs,
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
				consumeErr: assert.AnError,
			},
			doMock: func(a args) {
				hh.cacheRepo.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil)
				hh.dlq.EXPECT().Publish(gomock.Any()).Return(assert.AnError)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			th := HvtBalanceHandler{
				bs:          tt.fields.bs,
				cacheRepo:   tt.fields.cacheRepo,
				dlq:         tt.fields.dlq,
				featureFlag: tt.fields.featureFlag,
			}
			th.Nack(tt.args.ctx, tt.args.session, tt.args.payload, tt.args.consumeErr)
		})
	}
}

func TestHvtBalanceUpdateHandler_ConsumeClaim(t *testing.T) {
	hh := newHvtBalanceUpdateHandlerHelper(t)
	defer hh.mockCtrl.Finish()

	type fields struct {
		bs          *mock.MockBalanceService
		ctx         context.Context
		ctxCancel   context.CancelFunc
		msg         chan *sarama.ConsumerMessage
		dlq         *publisherMock.MockPublisher
		cacheRepo   *repositoryMock.MockCacheRepository
		featureFlag config.FeatureFlag
	}

	type args struct {
		session *kafkaMock.MockConsumerGroupSession
		claim   *kafkaMock.MockConsumerGroupClaim
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMock  func(a args, f fields)
		wantErr bool
	}{
		{
			name: "success consume message",
			fields: fields{
				bs:        hh.bs,
				msg:       make(chan *sarama.ConsumerMessage, 1),
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(hh.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(hh.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: hh.payload,
				}

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()
				f.cacheRepo.EXPECT().SetIfNotExists(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				f.bs.EXPECT().AdjustAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed - err Unmarshal",
			fields: fields{
				bs:        hh.bs,
				msg:       make(chan *sarama.ConsumerMessage, 1),
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(hh.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(hh.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: []byte("{__INVALID_JSON_HERE"),
				}

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()

				f.cacheRepo.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				f.dlq.EXPECT().Publish(gomock.Any()).Return(nil).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed consume message and send to DLQ",
			fields: fields{
				bs:        hh.bs,
				msg:       make(chan *sarama.ConsumerMessage, 1),
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(hh.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(hh.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: hh.payload,
				}

				str := ""

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()

				f.cacheRepo.EXPECT().SetIfNotExists(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
				f.bs.EXPECT().AdjustAccountBalance(gomock.Any(), gomock.AssignableToTypeOf(str), gomock.Any()).Return(assert.AnError).AnyTimes()

				f.cacheRepo.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				f.dlq.EXPECT().Publish(gomock.Any()).Return(nil).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Idempotency key already exist",
			fields: fields{
				bs:        hh.bs,
				msg:       make(chan *sarama.ConsumerMessage, 1),
				dlq:       hh.dlq,
				cacheRepo: hh.cacheRepo,
				featureFlag: config.FeatureFlag{
					EnablePublishHvtBalanceDLQ: true,
				},
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(hh.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(hh.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: hh.payload,
				}

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()

				f.cacheRepo.EXPECT().SetIfNotExists(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.ctx, tt.fields.ctxCancel = context.WithTimeout(context.Background(), 1*time.Second)
			defer tt.fields.ctxCancel()

			if tt.doMock != nil {
				tt.doMock(tt.args, tt.fields)
			}

			hh := HvtBalanceHandler{
				clientId:  "",
				bs:        hh.bs,
				cacheRepo: hh.cacheRepo,
				dlq:       hh.dlq,
			}

			err := hh.ConsumeClaim(tt.args.session, tt.args.claim)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}

}
