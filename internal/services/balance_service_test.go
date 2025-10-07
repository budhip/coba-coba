package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBalanceService_Get(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx           context.Context
		accountNumber string
	}

	tests := []struct {
		name    string
		args    args
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
			},
			doMock: func() {
				testHelper.mockBalanceRepository.
					EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(models.AccountBalance{}, nil)
			},
			wantErr: false,
		},
		{
			name: "error repository",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
			},
			doMock: func() {
				testHelper.mockBalanceRepository.
					EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(models.AccountBalance{}, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			_, err := testHelper.balanceService.Get(tc.args.ctx, tc.args.accountNumber)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestBalanceService_AdjustAccountBalance(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx           context.Context
		accountNumber string
		updateAmount  models.Decimal
	}

	tests := []struct {
		name    string
		args    args
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
				updateAmount:  models.NewDecimalFromExternal(decimal.NewFromFloat(123.2)),
			},
			doMock: func() {
				testHelper.mockBalanceRepository.
					EXPECT().
					AdjustAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error repository",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
			},
			doMock: func() {
				testHelper.mockBalanceRepository.
					EXPECT().
					AdjustAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			err := testHelper.balanceService.AdjustAccountBalance(tc.args.ctx, tc.args.accountNumber, tc.args.updateAmount)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}
