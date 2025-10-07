package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_transaction_PublishTransaction(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx context.Context
		req models.DoPublishTransactionRequest
	}
	type mockData struct {
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "success - publish transaction",
			args: args{
				ctx: context.Background(),
				req: models.DoPublishTransactionRequest{
					FromAccount:     "666",
					ToAccount:       "777",
					Amount:          "420000.69",
					Method:          "BANK_TRANSFER",
					TransactionType: "DISBNORMBPEBSA",
					TransactionDate: "2006-01-02",
					OrderType:       "DISBURSEMENT",
					RefNumber:       "12345abcd",
					Description:     "TEST DISBURSEMENT",
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(args.ctx, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - publish transaction",
			args: args{
				ctx: context.Background(),
				req: models.DoPublishTransactionRequest{
					FromAccount:     "666",
					ToAccount:       "777",
					Amount:          "420000.69",
					Method:          "BANK_TRANSFER",
					TransactionType: "DISBNORMBPEBSA",
					TransactionDate: "2006-01-02",
					OrderType:       "DISBURSEMENT",
					Description:     "TEST DISBURSEMENT",
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockIDGenerator.EXPECT().Generate(models.TransactionIDManualPrefix).Return(uuid.New().String()).AnyTimes()
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(args.ctx, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - amount invalid",
			args: args{
				ctx: context.Background(),
				req: models.DoPublishTransactionRequest{
					Amount:          "420000,69",
					TransactionDate: "2006-01-02",
				},
			},
			doMock: func(args args, mockData mockData) {
			},
			wantErr: true,
		},
		{
			name: "error - transactionDate invalid",
			args: args{
				ctx: context.Background(),
				req: models.DoPublishTransactionRequest{
					Amount:          "420000.69",
					TransactionDate: "2006-01-026",
				},
			},
			doMock: func(args args, mockData mockData) {
			},
			wantErr: true,
		},
		{
			name: "error - publish transaction",
			args: args{
				ctx: context.Background(),
				req: models.DoPublishTransactionRequest{
					FromAccount:     "666",
					ToAccount:       "777",
					Amount:          "420000.69",
					Method:          "BANK_TRANSFER",
					TransactionType: "DISBNORMBPEBSA",
					TransactionDate: "2006-01-02",
					OrderType:       "DISBURSEMENT",
					RefNumber:       "12345abcd",
					Description:     "TEST DISBURSEMENT",
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAcuanClient.EXPECT().PublishTransaction(args.ctx, gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			_, err := testHelper.transactionService.PublishTransaction(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
