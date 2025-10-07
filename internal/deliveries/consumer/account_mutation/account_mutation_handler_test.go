package account_mutation

import (
	"context"
	"testing"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	mock4 "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type accountMutationHandlerHelper struct {
	mockCtrl *gomock.Controller
	as       *mock.MockAccountService
	dlq      *mock4.MockPublisher

	payload                  []byte
	migrationDatabasePayload []byte
}

func newAccountMutationHandlerHelper(t *testing.T) accountMutationHandlerHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	as := mock.NewMockAccountService(mockCtrl)
	dlq := mock4.NewMockPublisher(mockCtrl)

	payload := []byte(`{
	    "headers": {
	        "sourceSystem": "go-accounting"
	    },
	    "body": {
	        "data": {
	            "account": {
					"type": "account_created",
					"kind": "account",
					"accountNumber": "21101100000001",
					"name": "Lender Yang Baik",
					"ownerId": "12345",
					"categoryCode": "211",
					"subCategoryCode": "100",
					"entityCode": "001",
					"currency": "IDR",
					"altId": "534534534555353523523423423",
					"t24Id": "111000035909",
					"metadata": null
				}
	        }
	    }
	}`)

	return accountMutationHandlerHelper{
		mockCtrl: mockCtrl,
		as:       as,
		dlq:      dlq,
		payload:  payload,
		migrationDatabasePayload: []byte(`{
			"headers": {
				"sourceSystem": "go-accounting"
			},
			"body": {
				"data": {
					"account": {
						"type": "migration_database",
						"kind": "account",
						"accountNumber": "21101100000001",
						"name": "Lender Yang Baik",
						"ownerId": "12345",
						"categoryCode": "211",
						"subCategoryCode": "100",
						"entityCode": "001",
						"currency": "IDR",
						"altId": "534534534555353523523423423",
						"t24Id": "111000035909",
						"metadata": null
					}
				}
			}
		}`),
	}
}

func TestNewAccountMutationHandler(t *testing.T) {
	th := newAccountMutationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	cfg := &config.Config{}

	type args struct {
		cfg *config.Config
		as  services.AccountService
		dlq dlqpublisher.Publisher
	}
	tests := []struct {
		name string
		args args
		want *AccountMutationHandler
	}{
		{
			name: "success init NewAccountMutationHandler",
			args: args{
				cfg: cfg,
				as:  th.as,
				dlq: th.dlq,
			},
			want: &AccountMutationHandler{
				as:  th.as,
				dlq: th.dlq,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewAccountMutationHandler("", tt.args.as, tt.args.dlq, *tt.args.cfg, nil), "NewTransactionHandler(%v)", tt.args.as)
		})
	}
}

func TestAccountMutationHandler_Cleanup(t *testing.T) {
	th := newAccountMutationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		as services.AccountService
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
				as: th.as,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := AccountMutationHandler{
				as: tt.fields.as,
			}
			err := th.Cleanup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountMutationHandler_processMessage(t *testing.T) {
	th := newAccountMutationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	tests := []struct {
		name                                   string
		enablePreventSameAccountMutationActing bool
		message                                *sarama.ConsumerMessage
		doMock                                 func()
		wantErr                                bool
	}{
		{
			name:    "upsert - happy path",
			message: &sarama.ConsumerMessage{Value: th.payload},
			doMock: func() {
				th.as.EXPECT().Upsert(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "upsert with migration_database - happy path",
			message: &sarama.ConsumerMessage{Value: th.migrationDatabasePayload},
			doMock: func() {
				th.as.EXPECT().RemoveDuplicateAccountMigration(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf("")).Return(nil)
				th.as.EXPECT().Upsert(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:                                   "insert - happy path",
			message:                                &sarama.ConsumerMessage{Value: th.payload},
			enablePreventSameAccountMutationActing: true,
			doMock: func() {
				th.as.EXPECT().GetOneByAccountNumber(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(models.GetAccountOut{}, assert.AnError)
				th.as.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(models.CreateAccount{}, nil)
			},
			wantErr: false,
		},
		{
			name:    "error marshall message",
			message: &sarama.ConsumerMessage{Value: []byte("{__INVALID_JSON_HERE")},
			wantErr: true,
		},
		{
			name:    "upsert with migration_database - err service",
			message: &sarama.ConsumerMessage{Value: th.migrationDatabasePayload},
			doMock: func() {
				th.as.EXPECT().RemoveDuplicateAccountMigration(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf("")).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:    "upsert - err upsert",
			message: &sarama.ConsumerMessage{Value: th.payload},
			doMock: func() {
				th.as.EXPECT().Upsert(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:                                   "insert - account exist",
			message:                                &sarama.ConsumerMessage{Value: th.payload},
			enablePreventSameAccountMutationActing: true,
			doMock: func() {
				th.as.EXPECT().GetOneByAccountNumber(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(models.GetAccountOut{}, nil)
			},
			wantErr: true,
		},
		{
			name:                                   "insert - error insert",
			message:                                &sarama.ConsumerMessage{Value: th.payload},
			enablePreventSameAccountMutationActing: true,
			doMock: func() {
				th.as.EXPECT().GetOneByAccountNumber(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(models.GetAccountOut{}, assert.AnError)
				th.as.EXPECT().Create(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).Return(models.CreateAccount{}, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			h := AccountMutationHandler{
				as: th.as,
				cfg: config.Config{
					FeatureFlag: config.FeatureFlag{
						EnablePreventSameAccountMutationActing: tt.enablePreventSameAccountMutationActing,
					},
				},
			}
			err := h.processMessage(context.Background(), tt.message)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestAccountMutationHandler_Setup(t *testing.T) {
	th := newAccountMutationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		as services.AccountService
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
				as: th.as,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := AccountMutationHandler{
				as: tt.fields.as,
			}
			err := th.Setup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountMutationHandler_ConsumeClaim(t *testing.T) {
	th := newAccountMutationHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		as        *mock.MockAccountService
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
				as:  mock.NewMockAccountService(th.mockCtrl),
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
				f.as.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed consume message",
			fields: fields{
				as:  mock.NewMockAccountService(th.mockCtrl),
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

				f.as.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()

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

			th := AccountMutationHandler{
				as:  tt.fields.as,
				dlq: tt.fields.dlq,
			}

			err := th.ConsumeClaim(tt.args.session, tt.args.claim)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
