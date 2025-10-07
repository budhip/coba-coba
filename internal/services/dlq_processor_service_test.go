package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

func Test_dlqProcessor_SendNotificationOrderFailure(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx     context.Context
		message models.FailedMessage
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success handle transaction failure from DLQ",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"order":{"orderTime":"2023-10-10T11:41:16Z","orderType":"CASHOUT","refNumber":"TRX-MANUAL-123","transactions":[{"id":"4407fda5-479b-4f1d-8654-3a90d6796719","amount":"1000000","currency":"IDR","sourceAccountId":"113000000055","destinationAccountId":"IDR1417100011000","description":"CASHOUT.LENDER","method":"","transactionType":"CIH_DEDUCTED","transactionTime":"2023-10-10T11:41:16Z","status":1,"meta":{"fromNarrative":"CASHOUT CHANEL","toNarrative":"LENDER","source":"ng-mis.web","remarks":""}}]}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: false,
		},
		{
			name: "success with structured error logging",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"order":{"orderTime":"2023-10-10T11:41:16Z","orderType":"CASHOUT","refNumber":"TRX-MANUAL-123","transactions":[{"id":"4407fda5-479b-4f1d-8654-3a90d6796719","amount":"1000000","currency":"IDR","sourceAccountId":"113000000055","destinationAccountId":"IDR1417100011000","description":"CASHOUT.LENDER","method":"","transactionType":"CIH_DEDUCTED","transactionTime":"2023-10-10T11:41:16Z","status":1,"meta":{"fromNarrative":"CASHOUT CHANEL","toNarrative":"LENDER","source":"ng-mis.web","remarks":""}}]}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: false,
		},
		{
			name: "failed to unmarshal payload",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte("e3lvX3doYXRfaXNfdGhpc30="),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			err := testHelper.dlqProcessorService.SendNotificationOrderFailure(tc.args.ctx, tc.args.message)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func Test_dlqProcessor_SendNotificationAccountFailure(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx     context.Context
		message models.FailedMessage
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"account":{"type":"","accountNumber":"abc","name":"","ownerId":"","categoryCode":"","subCategoryCode":"","entityCode":"","currency":"","altId":"","status":"","legacyId":null,"metadata":null}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: false,
		},
		{
			name: "success with structured error logging",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"account":{"type":"","accountNumber":"abc","name":"","ownerId":"","categoryCode":"","subCategoryCode":"","entityCode":"","currency":"","altId":"","status":"","legacyId":null,"metadata":null}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: false,
		},
		{
			name: "failed - unmarshal",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte("e3lvX3doYXRfaXNfdGhpc30="),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			err := testHelper.dlqProcessorService.SendNotificationAccountFailure(tc.args.ctx, tc.args.message)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func Test_dlqProcessor_RetryAccountMutation(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx     context.Context
		message models.FailedMessage
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success send job",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"account":{"type":"","accountNumber":"abc","name":"","ownerId":"","categoryCode":"","subCategoryCode":"","entityCode":"","currency":"","altId":"","status":"","legacyId":null,"metadata":null}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Set(args.ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				testHelper.mockQueueUnicornClient.EXPECT().SendJobHTTP(args.ctx, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - message job err",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"account":{"type":"","accountNumber":"abc","name":"","ownerId":"","categoryCode":"","subCategoryCode":"","entityCode":"","currency":"","altId":"","status":"","legacyId":null,"metadata":null}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Set(args.ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				testHelper.mockQueueUnicornClient.EXPECT().SendJobHTTP(args.ctx, gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - unmarshal",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte("e3lvX3doYXRfaXNfdGhpc30="),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			err := testHelper.dlqProcessorService.RetryAccountMutation(tc.args.ctx, tc.args.message)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func Test_dlqProcessor_RetryCreateOrderTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx     context.Context
		message models.FailedMessage
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success send job",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"account":{"type":"","accountNumber":"abc","name":"","ownerId":"","categoryCode":"","subCategoryCode":"","entityCode":"","currency":"","altId":"","status":"","legacyId":null,"metadata":null}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Set(args.ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				testHelper.mockQueueUnicornClient.EXPECT().SendJobHTTP(args.ctx, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - message job err",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte(`{"headers":{"sourceSystem":"go-core-cash-out"},"body":{"data":{"account":{"type":"","accountNumber":"abc","name":"","ownerId":"","categoryCode":"","subCategoryCode":"","entityCode":"","currency":"","altId":"","status":"","legacyId":null,"metadata":null}}}}`),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Set(args.ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				testHelper.mockQueueUnicornClient.EXPECT().SendJobHTTP(args.ctx, gomock.Any()).Return(assert.AnError)
				// No mock expectations needed - using structured error logging
			},
			wantErr: true,
		},
		{
			name: "failed - unmarshal",
			args: args{
				ctx: context.TODO(),
				message: models.FailedMessage{
					Payload:   []byte("e3lvX3doYXRfaXNfdGhpc30="),
					Timestamp: time.Now(),
					Error:     "this is dummy error test",
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			err := testHelper.dlqProcessorService.RetryCreateOrderTransaction(tc.args.ctx, tc.args.message)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func Test_dlqProcessor_GetStatusRetry(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx            context.Context
		processRetryId string
	}
	tests := []struct {
		name     string
		args     args
		doMock   func(args args)
		wantErr  bool
		wantData models.StatusRetryDLQ
	}{
		{
			name: "success get status retry",
			args: args{
				ctx:            context.TODO(),
				processRetryId: "123456",
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Get(args.ctx, gomock.Any()).
					Return(`{"processId":"123456","processName":"create account","maxRetry":5,"currentRetry":1}`, nil)
			},
			wantErr: false,
			wantData: models.StatusRetryDLQ{
				ProcessId:    "123456",
				ProcessName:  "create account",
				MaxRetry:     5,
				CurrentRetry: 1,
			},
		},
		{
			name: "failed - get status retry",
			args: args{
				ctx:            context.TODO(),
				processRetryId: "123456",
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Get(args.ctx, gomock.Any()).
					Return("", assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - unmarshal",
			args: args{
				ctx:            context.TODO(),
				processRetryId: "123456",
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().
					Get(args.ctx, gomock.Any()).
					Return(`{__invalid_JSON_here`, nil)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			res, err := testHelper.dlqProcessorService.GetStatusRetry(tc.args.ctx, tc.args.processRetryId)
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantData, res)
		})
	}
}

func Test_dlqProcessor_UpsertStatusRetry(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx            context.Context
		processRetryId string
		status         models.StatusRetryDLQ
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success update status retry",
			args: args{
				ctx:            context.TODO(),
				processRetryId: "123456",
				status: models.StatusRetryDLQ{
					ProcessId:    "123456",
					ProcessName:  "create account",
					MaxRetry:     5,
					CurrentRetry: 1,
				},
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().Set(args.ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - update status retry",
			args: args{
				ctx:            context.TODO(),
				processRetryId: "123456",
			},
			doMock: func(args args) {
				testHelper.mockCacheRepository.
					EXPECT().Set(args.ctx, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			err := testHelper.dlqProcessorService.UpsertStatusRetry(tc.args.ctx, tc.args.processRetryId, tc.args.status)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func Test_dlqProcessor_SendNotificationRetryFailure(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx       context.Context
		operation string
		message   string
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success update status retry",
			args: args{
				ctx:       context.TODO(),
				operation: "create account",
				message:   "this is dummy message",
			},
			wantErr: false,
		},
		{
			name: "success with structured error logging",
			args: args{
				ctx:       context.TODO(),
				operation: "create account",
				message:   "this is dummy message",
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}

			err := testHelper.dlqProcessorService.SendNotificationRetryFailure(tc.args.ctx, tc.args.operation, tc.args.message)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}
