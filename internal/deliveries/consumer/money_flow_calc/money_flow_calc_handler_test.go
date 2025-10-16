package money_flow_calc

import (
	"context"
	"testing"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	mock4 "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type moneyFlowCalcHandlerHelper struct {
	mockCtrl *gomock.Controller
	mfs      *mock.MockMoneyFlowService
	dlq      *mock4.MockPublisher

	payload []byte
}

func newMoneyFlowCalcHandlerHelper(t *testing.T) moneyFlowCalcHandlerHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	mfs := mock.NewMockMoneyFlowService(mockCtrl)
	dlq := mock4.NewMockPublisher(mockCtrl)

	payload := []byte(`{
		"identifier": "test-ref-123",
		"status": "SUCCESS",
		"acuanData": {
			"headers": {
				"sourceSystem": "go-core-topup"
			},
			"body": {
				"data": {
					"order": {
						"orderTime": "2023-06-14T11:13:24+07:00",
						"orderType": "TOPUP",
						"refNumber": "trx080989999",
						"transactions": [
							{
								"id": "e931d4d8-d554-4ba3-acef-df7dde8936a8",
								"amount": "420.69",
								"currency": "IDR",
								"sourceAccountId": "666",
								"destinationAccountId": "777",
								"description": "TEST TOPUP",
								"method": "TOPUP",
								"transactionType": "",
								"transactionTime": "2023-06-14T11:13:24+07:00",
								"status": 1
							}
						]
					}
				}
			}
		}
	}`)

	return moneyFlowCalcHandlerHelper{
		mockCtrl: mockCtrl,
		mfs:      mfs,
		dlq:      dlq,
		payload:  payload,
	}
}

func TestNewMoneyFlowCalcHandler(t *testing.T) {
	th := newMoneyFlowCalcHandlerHelper(t)
	defer th.mockCtrl.Finish()

	cfg := &config.Config{}

	type args struct {
		cfg *config.Config
		mfs services.MoneyFlowService
		dlq dlqpublisher.Publisher
	}
	tests := []struct {
		name string
		args args
		want *MoneyFlowCalcHandler
	}{
		{
			name: "success init MoneyFlowCalcHandler",
			args: args{
				cfg: cfg,
				mfs: th.mfs,
				dlq: th.dlq,
			},
			want: &MoneyFlowCalcHandler{
				mfs: th.mfs,
				dlq: th.dlq,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewMoneyFlowCalcHandler("", tt.args.mfs, tt.args.dlq, *tt.args.cfg, nil), "NewMoneyFlowCalcHandler(%v)", tt.args.mfs)
		})
	}
}

func TestMoneyFlowCalcHandler_Cleanup(t *testing.T) {
	th := newMoneyFlowCalcHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		mfs services.MoneyFlowService
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
			name: "Success Cleanup",
			fields: fields{
				mfs: th.mfs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := MoneyFlowCalcHandler{
				mfs: tt.fields.mfs,
			}
			err := th.Cleanup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestMoneyFlowCalcHandler_processMessage(t *testing.T) {
	th := newMoneyFlowCalcHandlerHelper(t)
	defer th.mockCtrl.Finish()

	tests := []struct {
		name    string
		message *sarama.ConsumerMessage
		doMock  func()
		wantErr bool
	}{
		{
			name:    "happy path",
			message: &sarama.ConsumerMessage{Value: th.payload},
			doMock: func() {
				th.mfs.EXPECT().ProcessTransactionNotification(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "error marshall message",
			message: &sarama.ConsumerMessage{Value: []byte("{__INVALID_JSON_HERE")},
			wantErr: true,
		},
		{
			name:    "error service",
			message: &sarama.ConsumerMessage{Value: th.payload},
			doMock: func() {
				th.mfs.EXPECT().ProcessTransactionNotification(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			h := MoneyFlowCalcHandler{
				mfs: th.mfs,
			}
			err := h.processMessage(context.Background(), tt.message)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestMoneyFlowCalcHandler_Setup(t *testing.T) {
	th := newMoneyFlowCalcHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		mfs services.MoneyFlowService
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
				mfs: th.mfs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := MoneyFlowCalcHandler{
				mfs: tt.fields.mfs,
			}
			err := th.Setup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestMoneyFlowCalcHandler_ConsumeClaim(t *testing.T) {
	th := newMoneyFlowCalcHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		mfs       *mock.MockMoneyFlowService
		dlq       *mock4.MockPublisher
		ctx       context.Context
		ctxCancel context.CancelFunc
		msg       chan *sarama.ConsumerMessage
	}

	type args struct {
		session *mock2.MockConsumerGroupSession
		claim   *mock2.MockConsumerGroupClaim
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
				mfs: mock.NewMockMoneyFlowService(th.mockCtrl),
				dlq: mock4.NewMockPublisher(th.mockCtrl),
				msg: make(chan *sarama.ConsumerMessage, 1),
			},
			args: args{
				session: mock2.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   mock2.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: th.payload,
				}

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
				f.mfs.EXPECT().ProcessTransactionNotification(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed consume message and send to DLQ",
			fields: fields{
				mfs: mock.NewMockMoneyFlowService(th.mockCtrl),
				dlq: mock4.NewMockPublisher(th.mockCtrl),
				msg: make(chan *sarama.ConsumerMessage, 1),
			},
			args: args{
				session: mock2.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   mock2.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: th.payload,
				}

				f.mfs.EXPECT().ProcessTransactionNotification(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()

				f.dlq.EXPECT().Publish(gomock.Any()).Return(nil).AnyTimes()

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

			th := MoneyFlowCalcHandler{
				mfs: tt.fields.mfs,
				dlq: tt.fields.dlq,
			}

			err := th.ConsumeClaim(tt.args.session, tt.args.claim)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
