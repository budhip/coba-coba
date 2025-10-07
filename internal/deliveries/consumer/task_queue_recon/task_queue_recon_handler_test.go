package queuerecon

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type taskQueueReconHandlerHelper struct {
	mockCtrl *gomock.Controller
	rs       *mock.MockReconService

	payload []byte
}

func newTaskQueueReconHandlerHelper(t *testing.T) taskQueueReconHandlerHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	rs := mock.NewMockReconService(mockCtrl)

	payload := []byte(`{"id":"1","task":"RECON_FILE"}`)

	return taskQueueReconHandlerHelper{
		mockCtrl: mockCtrl,
		rs:       rs,
		payload:  payload,
	}
}

func TestNewTaskQueueReconHandler(t *testing.T) {
	th := newTaskQueueReconHandlerHelper(t)
	defer th.mockCtrl.Finish()

	cfg := &config.Config{}

	type args struct {
		cfg *config.Config
		rs  services.ReconService
	}
	tests := []struct {
		name string
		args args
		want *TaskQueueReconHandler
	}{
		{
			name: "success init TaskQueueReconHandler",
			args: args{
				cfg: cfg,
				rs:  th.rs,
			},
			want: &TaskQueueReconHandler{
				rs: th.rs,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewTaskQueueReconHandler("", tt.args.rs), "NewTransactionHandler(%v)", tt.args.rs)
		})
	}
}

func TestTaskQueueReconHandler_Cleanup(t *testing.T) {
	th := newTaskQueueReconHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		rs services.ReconService
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
				rs: th.rs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := TaskQueueReconHandler{
				rs: tt.fields.rs,
			}
			err := th.Cleanup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTaskQueueReconHandler_processMessage(t *testing.T) {
	th := newTaskQueueReconHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		rs *mock.MockReconService
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
			name: "success handle message",
			fields: fields{
				rs: th.rs,
			},
			args: args{
				ctx: context.Background(),
				message: &sarama.ConsumerMessage{
					Value: th.payload,
				},
			},
			doMock: func(a args) {
				th.rs.EXPECT().ProcessReconTaskQueue(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error marshall message",
			fields: fields{
				rs: th.rs,
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
			name: "error process recon task",
			fields: fields{
				rs: th.rs,
			},
			args: args{
				ctx: context.Background(),
				message: &sarama.ConsumerMessage{
					Value: th.payload,
				},
			},
			doMock: func(a args) {
				th.rs.EXPECT().ProcessReconTaskQueue(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			h := TaskQueueReconHandler{
				rs: tt.fields.rs,
			}
			err := h.processMessage(tt.args.ctx, tt.args.message)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestTaskQueueReconHandler_Setup(t *testing.T) {
	th := newTaskQueueReconHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		rs services.ReconService
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
				rs: th.rs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := TaskQueueReconHandler{
				rs: tt.fields.rs,
			}
			err := th.Setup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTaskQueueReconHandler_ConsumeClaim(t *testing.T) {
	th := newTaskQueueReconHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		rs        *mock.MockReconService
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
				rs:  mock.NewMockReconService(th.mockCtrl),
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
				f.rs.EXPECT().ProcessReconTaskQueue(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed consume message",
			fields: fields{
				rs:  mock.NewMockReconService(th.mockCtrl),
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

				f.rs.EXPECT().ProcessReconTaskQueue(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()

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

			th := TaskQueueReconHandler{
				rs: tt.fields.rs,
			}

			err := th.ConsumeClaim(tt.args.session, tt.args.claim)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
