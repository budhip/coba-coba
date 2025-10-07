package dlq_notification

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"
)

type transactionHandlerHelper struct {
	mockCtrl    *gomock.Controller
	dp          *mock.MockDLQProcessorService
	consumerCfg config.ConsumerConfig

	payload []byte
}

func newDlqNotificationHandlerHelper(t *testing.T) transactionHandlerHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	dp := mock.NewMockDLQProcessorService(mockCtrl)

	payload := []byte(`{
	"payload": "eyJoZWFkZXJzIjp7InNvdXJjZVN5c3RlbSI6ImdvLWNvcmUtY2FzaC1vdXQifSwiYm9keSI6eyJkYXRhIjp7Im9yZGVyIjp7Im9yZGVyVGltZSI6IjIwMjMtMTAtMTBUMTE6NDE6MTZaIiwib3JkZXJUeXBlIjoiQ0FTSE9VVCIsInJlZk51bWJlciI6IjllMmRkMTE5LTBjNjItNDIyMC04ZDRjLTYwZjZhMDQ3YjA1MyIsInRyYW5zYWN0aW9ucyI6W3siaWQiOiI0NDA3ZmRhNS00NzliLTRmMWQtODY1NC0zYTkwZDY3OTY3MTkiLCJhbW91bnQiOiIxMDAwMDAwIiwiY3VycmVuY3kiOiJJRFIiLCJzb3VyY2VBY2NvdW50SWQiOiIxMTMwMDAwMDAwNTUiLCJkZXN0aW5hdGlvbkFjY291bnRJZCI6IklEUjE0MTcxMDAwMTEwMDAiLCJkZXNjcmlwdGlvbiI6IkNBU0hPVVQuTEVOREVSIiwibWV0aG9kIjoiIiwidHJhbnNhY3Rpb25UeXBlIjoiQ0lIX0RFRFVDVEVEIiwidHJhbnNhY3Rpb25UaW1lIjoiMjAyMy0xMC0xMFQxMTo0MToxNloiLCJzdGF0dXMiOjEsIm1ldGEiOnsiZnJvbU5hcnJhdGl2ZSI6IkNBU0hPVVQgQ0hBTkVMIiwidG9OYXJyYXRpdmUiOiJMRU5ERVIiLCJzb3VyY2UiOiJuZy1taXMud2ViIiwicmVtYXJrcyI6IiJ9fV19fX19",
	"timestamp": "2023-10-10T11:41:21.35Z",
	"error": "An error has occurred from "
}`)

	return transactionHandlerHelper{
		mockCtrl: mockCtrl,
		dp:       dp,
		payload:  payload,
		consumerCfg: config.ConsumerConfig{
			TopicDLQ:                "TopicDLQ",
			TopicAccountMutationDLQ: "TopicAccountMutationDLQ",
		},
	}
}

func TestNewDLQNotificationHandler(t *testing.T) {
	th := newDlqNotificationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	cfg := &config.Config{}

	type args struct {
		cfg *config.Config
		dp  services.DLQProcessorService
	}
	tests := []struct {
		name string
		args args
		want *DLQNotificationHandler
	}{
		{
			name: "success init DLQNotificationHandler",
			args: args{
				cfg: cfg,
				dp:  th.dp,
			},
			want: &DLQNotificationHandler{
				dp: th.dp,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewNotificationHandler("", tt.args.dp, tt.args.cfg.MessageBroker.KafkaConsumer, nil), "NewTransactionHandler(%v)", tt.args.dp)
		})
	}
}

func TestTransactionHandler_Cleanup(t *testing.T) {
	th := newDlqNotificationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		dp services.DLQProcessorService
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
				dp: th.dp,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := DLQNotificationHandler{
				dp: tt.fields.dp,
			}
			err := th.Cleanup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTransactionHandler_processMessage(t *testing.T) {
	th := newDlqNotificationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		dp *mock.MockDLQProcessorService
	}

	type args struct {
		ctx     context.Context
		message *sarama.ConsumerMessage
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		doMock  func(a args)
		wantErr bool
	}{
		{
			name: "success handle message - order",
			fields: fields{
				dp: th.dp,
			},
			args: args{
				ctx: context.Background(),
				message: &sarama.ConsumerMessage{
					Value: th.payload,
					Topic: th.consumerCfg.TopicDLQ,
				},
			},
			doMock: func(a args) {
				th.dp.EXPECT().SendNotificationOrderFailure(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success handle message - account",
			fields: fields{
				dp: th.dp,
			},
			args: args{
				ctx: context.Background(),
				message: &sarama.ConsumerMessage{
					Value: th.payload,
					Topic: th.consumerCfg.TopicAccountMutationDLQ,
				},
			},
			doMock: func(a args) {
				th.dp.EXPECT().SendNotificationAccountFailure(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error marshall message",
			fields: fields{
				dp: th.dp,
			},
			args: args{
				ctx: context.Background(),
				message: &sarama.ConsumerMessage{
					Value: []byte("{__INVALID_JSON_HERE"),
				},
			},
			wantErr: true,
		},
		{
			name: "error service",
			fields: fields{
				dp: th.dp,
			},
			args: args{
				ctx: context.Background(),
				message: &sarama.ConsumerMessage{
					Value: th.payload,
					Topic: th.consumerCfg.TopicDLQ,
				},
			},
			doMock: func(a args) {
				th.dp.EXPECT().SendNotificationOrderFailure(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			h := DLQNotificationHandler{
				dp:          tt.fields.dp,
				consumerCfg: th.consumerCfg,
			}
			err := h.processMessage(tt.args.ctx, tt.args.message)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestTransactionHandler_Setup(t *testing.T) {
	th := newDlqNotificationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		dp services.DLQProcessorService
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
				dp: th.dp,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := DLQNotificationHandler{
				dp: tt.fields.dp,
			}
			err := th.Setup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTransactionHandler_ConsumeClaim(t *testing.T) {
	th := newDlqNotificationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		dp        *mock.MockDLQProcessorService
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
				dp:  mock.NewMockDLQProcessorService(th.mockCtrl),
				msg: make(chan *sarama.ConsumerMessage, 1),
			},
			args: args{
				session: mock2.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   mock2.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: th.payload,
					Topic: th.consumerCfg.TopicDLQ,
				}

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
				f.dp.EXPECT().SendNotificationOrderFailure(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed consume message and send to DLQ",
			fields: fields{
				dp:  mock.NewMockDLQProcessorService(th.mockCtrl),
				msg: make(chan *sarama.ConsumerMessage, 1),
			},
			args: args{
				session: mock2.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   mock2.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: th.payload,
					Topic: th.consumerCfg.TopicDLQ,
				}

				f.dp.EXPECT().SendNotificationOrderFailure(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()
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

			th := DLQNotificationHandler{
				dp: tt.fields.dp,
			}

			err := th.ConsumeClaim(tt.args.session, tt.args.claim)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
