package kafkaconsumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification"
	mock3 "bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	retryMock "bitbucket.org/Amartha/go-fp-transaction/internal/common/retry/mock"
	kafkaMock "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka/mock"

	acuanLibModel "bitbucket.org/Amartha/go-acuan-lib/model"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type transactionHandlerHelper struct {
	mockCtrl                *gomock.Controller
	ts                      *mock.MockTransactionService
	accountService          *mock.MockAccountService
	dlq                     *mock2.MockPublisher
	journal                 *kafkaMock.MockJournalPublisher
	transactionNotification *mock3.MockTransactionNotificationPublisher
	ebRetry                 *retryMock.MockRetryer

	payload []byte
}

func newTransactionHandlerHelper(t *testing.T) transactionHandlerHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	ts := mock.NewMockTransactionService(mockCtrl)
	accSvc := mock.NewMockAccountService(mockCtrl)
	mockDlq := mock2.NewMockPublisher(mockCtrl)
	mockJournal := kafkaMock.NewMockJournalPublisher(mockCtrl)
	mockTransactionNotification := mock3.NewMockTransactionNotificationPublisher(mockCtrl)

	payload := []byte(`{
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
	                        "status": 1,
	                        "meta": {
	                            "fromNarrative": "A_666",
	                            "toNarrative": "B_777",
	                            "vaData": {
	                                "source": "MANDIRI",
	                                "virtualAccountId": "777",
	                                "virtualAccountNo": "666088888888888"
	                            },
	                            "eWalletData": null,
	                            "institutionData": null
	                        }
	                    }
	                ]
	            }
	        }
	    }
	}`)

	return transactionHandlerHelper{
		mockCtrl:                mockCtrl,
		ts:                      ts,
		accountService:          accSvc,
		payload:                 payload,
		dlq:                     mockDlq,
		journal:                 mockJournal,
		transactionNotification: mockTransactionNotification,
		ebRetry:                 retryMock.NewMockRetryer(mockCtrl),
	}
}

func TestNewTransactionHandler(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	cfg := &config.Config{
		FeatureFlag: config.FeatureFlag{},
	}

	type args struct {
		cfg                     *config.Config
		ts                      services.TransactionService
		accountSvc              services.AccountService
		dlq                     dlqpublisher.Publisher
		journal                 JournalPublisher
		transactionNotification transaction_notification.TransactionNotificationPublisher
	}
	tests := []struct {
		name string
		args args
		want *TransactionHandler
	}{
		{
			name: "success init TransactionHandler",
			args: args{
				cfg:                     cfg,
				ts:                      th.ts,
				accountSvc:              th.accountService,
				dlq:                     th.dlq,
				journal:                 th.journal,
				transactionNotification: th.transactionNotification,
			},
			want: &TransactionHandler{
				ebRetry:                 th.ebRetry,
				ts:                      th.ts,
				accountSvc:              th.accountService,
				dlq:                     th.dlq,
				journal:                 th.journal,
				transactionNotification: th.transactionNotification,
				featureFlag:             cfg.FeatureFlag,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(
				t,
				tt.want,
				NewTransactionHandler(
					"",
					tt.args.ts,
					tt.args.dlq,
					th.ebRetry,
					tt.args.journal,
					tt.args.accountSvc,
					tt.args.cfg.FeatureFlag,
					tt.args.transactionNotification,
					nil),
				"NewTransactionHandler(%v)", tt.args.ts)
		})
	}
}

func TestTransactionHandler_Cleanup(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		ts  services.TransactionService
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
			name: "Success Cleanup",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := TransactionHandler{
				ts: tt.fields.ts,
			}
			err := th.Cleanup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTransactionHandler_processMessage(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		ts  *mock.MockTransactionService
		dlq dlqpublisher.Publisher
	}

	type args struct {
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
			name: "happy path",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
			},
			args: args{
				message: &sarama.ConsumerMessage{Value: th.payload},
			},
			doMock: func(a args) {
				th.ts.EXPECT().NewStoreBulkTransaction(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.AssignableToTypeOf([]models.TransactionReq{}),
				).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - err service",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
			},
			args: args{
				message: &sarama.ConsumerMessage{Value: th.payload},
			},
			doMock: func(a args) {
				th.ts.EXPECT().NewStoreBulkTransaction(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.AssignableToTypeOf([]models.TransactionReq{}),
				).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			var payload acuanLibModel.Payload[acuanLibModel.DataOrder]
			err := json.Unmarshal(tt.args.message.Value, &payload)
			assert.NoError(t, err)

			h := TransactionHandler{
				ts:  tt.fields.ts,
				dlq: tt.fields.dlq,
			}
			err = h.processMessage(context.Background(), tt.args.message, &payload)
			assert.Equal(t, tt.wantErr, err != nil, err)
		})
	}
}

func TestTransactionHandler_Setup(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		ts  services.TransactionService
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
			name: "Success Setup",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := TransactionHandler{
				ts: tt.fields.ts,
			}
			err := th.Setup(tt.args.session)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTransactionHandler_Ack(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	mockSess := kafkaMock.NewMockConsumerGroupSession(th.mockCtrl)

	type fields struct {
		ts  services.TransactionService
		dlq dlqpublisher.Publisher
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
		doMock func(a args)
	}{
		{
			name: "success Ack message",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
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
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "success Ack message but publish transaction notification failed",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
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
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			th := TransactionHandler{
				ts:                      tt.fields.ts,
				dlq:                     tt.fields.dlq,
				transactionNotification: th.transactionNotification,
				featureFlag:             config.FeatureFlag{EnablePublishTransactionNotification: true},
			}
			th.Ack(tt.args.ctx, tt.args.session, tt.args.payload)
		})
	}
}

func TestTransactionHandler_Nack(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	mockSess := kafkaMock.NewMockConsumerGroupSession(th.mockCtrl)

	type fields struct {
		ts  services.TransactionService
		dlq dlqpublisher.Publisher
		tn  transaction_notification.TransactionNotificationPublisher
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
			name: "success Nack message",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
				tn:  th.transactionNotification,
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
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
				th.dlq.EXPECT().Publish(models.FailedMessage{
					Payload:    a.payload.message.Value,
					Timestamp:  a.payload.message.Timestamp,
					CauseError: a.consumeErr,
				})
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "success Nack message - no send to DLQ (order already exists)",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
				tn:  th.transactionNotification,
			},
			args: args{
				session: mockSess,
				payload: &ackPayload{
					message: &sarama.ConsumerMessage{
						Value: []byte(`payload_here`),
						Topic: "test",
					},
				},
				consumeErr: common.ErrOrderAlreadyExists,
			},
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "success Nack message but publish transaction notification failed",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
				tn:  th.transactionNotification,
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
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)
				th.dlq.EXPECT().Publish(models.FailedMessage{
					Payload:    a.payload.message.Value,
					Timestamp:  a.payload.message.Timestamp,
					CauseError: a.consumeErr,
				})
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
			},
		},
		{
			name: "success Nack message but publish dlq failed",
			fields: fields{
				ts:  th.ts,
				dlq: th.dlq,
				tn:  th.transactionNotification,
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
				mockSess.EXPECT().MarkMessage(gomock.Any(), "")
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
				th.dlq.EXPECT().Publish(models.FailedMessage{
					Payload:    a.payload.message.Value,
					Timestamp:  a.payload.message.Timestamp,
					CauseError: a.consumeErr,
				}).Return(assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			th := TransactionHandler{
				ts:                      tt.fields.ts,
				dlq:                     tt.fields.dlq,
				transactionNotification: tt.fields.tn,
				featureFlag: config.FeatureFlag{
					EnablePublishTransactionNotification: true,
				},
			}
			th.Nack(tt.args.ctx, tt.args.session, tt.args.payload, tt.args.consumeErr)
		})
	}
}

func TestTransactionHandler_ConsumeClaim(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type fields struct {
		ts        *mock.MockTransactionService
		accSvc    *mock.MockAccountService
		dlq       *mock2.MockPublisher
		journal   *kafkaMock.MockJournalPublisher
		ctx       context.Context
		ctxCancel context.CancelFunc
		msg       chan *sarama.ConsumerMessage
		ebRetry   retryMock.MockRetryer
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
				ts:      mock.NewMockTransactionService(th.mockCtrl),
				accSvc:  mock.NewMockAccountService(th.mockCtrl),
				dlq:     th.dlq,
				journal: th.journal,
				msg:     make(chan *sarama.ConsumerMessage, 1),
				ebRetry: *th.ebRetry,
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: th.payload,
				}

				str := ""

				f.ebRetry.EXPECT().Retry(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
				f.journal.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				f.accSvc.EXPECT().GetACuanAccountNumber(gomock.Any(), gomock.AssignableToTypeOf(str)).Return(str, nil).AnyTimes()
				f.ts.EXPECT().NewStoreBulkTransaction(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed - err Unmarshal",
			fields: fields{
				ts:      mock.NewMockTransactionService(th.mockCtrl),
				accSvc:  mock.NewMockAccountService(th.mockCtrl),
				dlq:     th.dlq,
				journal: th.journal,
				msg:     make(chan *sarama.ConsumerMessage, 1),
				ebRetry: *th.ebRetry,
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: []byte("{__INVALID_JSON_HERE"),
				}

				f.dlq.EXPECT().Publish(gomock.Any()).Return(nil).AnyTimes()

				a.claim.EXPECT().Messages().Return(f.msg).AnyTimes()
				a.session.EXPECT().Context().Return(f.ctx).AnyTimes()
				a.session.EXPECT().MarkMessage(gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "failed consume message and send to DLQ",
			fields: fields{
				ts:      mock.NewMockTransactionService(th.mockCtrl),
				accSvc:  mock.NewMockAccountService(th.mockCtrl),
				dlq:     th.dlq,
				journal: th.journal,
				msg:     make(chan *sarama.ConsumerMessage, 1),
				ebRetry: *th.ebRetry,
			},
			args: args{
				session: kafkaMock.NewMockConsumerGroupSession(th.mockCtrl),
				claim:   kafkaMock.NewMockConsumerGroupClaim(th.mockCtrl),
			},
			doMock: func(a args, f fields) {
				f.msg <- &sarama.ConsumerMessage{
					Value: th.payload,
				}

				str := ""

				f.ebRetry.EXPECT().Retry(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				f.accSvc.EXPECT().GetACuanAccountNumber(gomock.Any(), gomock.AssignableToTypeOf(str)).Return(str, nil).AnyTimes()
				f.ts.EXPECT().NewStoreBulkTransaction(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()
				f.dlq.EXPECT().Publish(gomock.Any()).Return(nil).AnyTimes()

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

			th := TransactionHandler{
				ts:         tt.fields.ts,
				accountSvc: tt.fields.accSvc,
				dlq:        tt.fields.dlq,
				journal:    tt.fields.journal,
				ebRetry:    th.ebRetry,
				featureFlag: config.FeatureFlag{
					EnableCheckAccountTransaction: true,
				},
			}

			err := th.ConsumeClaim(tt.args.session, tt.args.claim)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestTransactionHandler_parseMessageToAcuanOrder(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	type MockData struct {
		accSvc *mock.MockAccountService
	}

	tests := []struct {
		name                            string
		payload                         []byte
		isEnableCheckAccountTransaction bool
		mockData                        MockData
		doMock                          func(mockData MockData)
		wantErr                         bool
	}{
		{
			name:                            "happy path",
			payload:                         th.payload,
			mockData:                        MockData{accSvc: mock.NewMockAccountService(th.mockCtrl)},
			isEnableCheckAccountTransaction: true,
			doMock: func(mockData MockData) {
				accountNumber := ""
				newAccountNumber := "NEW"
				mockData.accSvc.EXPECT().GetACuanAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.AssignableToTypeOf(accountNumber),
				).Return(newAccountNumber, nil).Times(2)
			},
			wantErr: false,
		},
		{
			name:                            "success - disable check account feature flag",
			isEnableCheckAccountTransaction: false,
			payload:                         th.payload,
			wantErr:                         false,
		},
		{
			name:                            "failed - amount is negative",
			isEnableCheckAccountTransaction: true,
			payload: []byte(`{
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
									"amount": "-1",
									"currency": "IDR",
									"sourceAccountId": "666",
									"destinationAccountId": "777",
									"description": "TEST TOPUP",
									"method": "TOPUP",
									"transactionType": "",
									"transactionTime": "2023-06-14T11:13:24+07:00",
									"status": 1,
									"meta": {
										"fromNarrative": "A_666",
										"toNarrative": "B_777",
										"vaData": {
											"source": "MANDIRI",
											"virtualAccountId": "777",
											"virtualAccountNo": "666088888888888"
										},
										"eWalletData": null,
										"institutionData": null
									}
								}
							]
						}
					}
				}
			}`),
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.mockData)
			}

			handler := TransactionHandler{accountSvc: tc.mockData.accSvc}
			handler.featureFlag = config.FeatureFlag{
				EnableCheckAccountTransaction:  tc.isEnableCheckAccountTransaction,
				EnableConsumerValidationReject: true,
			}
			_, err := handler.parseMessageToAcuanOrder(context.Background(), &sarama.ConsumerMessage{Value: tc.payload})
			assert.Equal(t, tc.wantErr, err != nil, err)
		})
	}
}

func TestTransactionHandler_publishTransactionNotification(t *testing.T) {
	th := newTransactionHandlerHelper(t)
	defer th.mockCtrl.Finish()

	acuanOrder := acuanLibModel.Payload[acuanLibModel.DataOrder]{}
	err := json.Unmarshal(th.payload, &acuanOrder)
	assert.NoError(t, err)

	type fields struct {
		transactionNotification transaction_notification.TransactionNotificationPublisher
	}
	type args struct {
		ctx          context.Context
		acuanOrder   acuanLibModel.Payload[acuanLibModel.DataOrder]
		operationErr error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMock  func(a args)
		wantErr bool
	}{
		{
			name: "success publish transaction notification",
			fields: fields{
				transactionNotification: th.transactionNotification,
			},
			args: args{
				ctx:          context.TODO(),
				acuanOrder:   acuanOrder,
				operationErr: nil,
			},
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success publish transaction notification with general error value",
			fields: fields{
				transactionNotification: th.transactionNotification,
			},
			args: args{
				ctx:          context.TODO(),
				acuanOrder:   acuanOrder,
				operationErr: assert.AnError,
			},
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success publish transaction notification with specific error value (order already exists)",
			fields: fields{
				transactionNotification: th.transactionNotification,
			},
			args: args{
				ctx:          context.TODO(),
				acuanOrder:   acuanOrder,
				operationErr: common.ErrOrderAlreadyExists,
			},
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed publish transaction notification with error value",
			fields: fields{
				transactionNotification: th.transactionNotification,
			},
			args: args{
				ctx:          context.TODO(),
				acuanOrder:   acuanOrder,
				operationErr: assert.AnError,
			},
			doMock: func(a args) {
				th.transactionNotification.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			th := TransactionHandler{
				transactionNotification: tc.fields.transactionNotification,
			}
			err := th.publishTransactionNotification(tc.args.ctx, acuanOrder, tc.args.operationErr)
			assert.Equal(t, tc.wantErr, err != nil, err)
		})
	}
}
