package report

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_reportHandler_GenerateTransactionReport(t *testing.T) {
	testHelper := reportTestHelper(t)

	type args struct {
		ctx  context.Context
		date time.Time
		flag flag.Job
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
		// TODO: Add test cases.
		{
			name: "success GenerateTransactionReport",
			args: args{
				ctx:  context.TODO(),
				date: common.Now(),
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTransactionService.EXPECT().GenerateTransactionReport(gomock.AssignableToTypeOf(args.ctx)).Return([]string{""}, nil)
			},
			wantErr: false,
		},
		{
			name: "error GenerateTransactionReport",
			args: args{
				ctx:  context.TODO(),
				date: common.Now(),
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockTransactionService.EXPECT().GenerateTransactionReport(gomock.AssignableToTypeOf(args.ctx)).Return([]string{""}, assert.AnError)
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
			rh := &reportHandler{
				transactionSrv: testHelper.mockTransactionService,
				reconSrv:       testHelper.mockReconService,
			}
			err := rh.GenerateTransactionReport(tt.args.ctx, tt.args.date, tt.args.flag)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_reportHandler_DoBalanceReconDaily(t *testing.T) {
	testHelper := reportTestHelper(t)

	type args struct {
		ctx  context.Context
		date time.Time
		flag flag.Job
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
		// TODO: Add test cases.
		{
			name: "success DoDailyBalance",
			args: args{
				ctx:  context.TODO(),
				date: common.Now(),
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockReconService.EXPECT().DoDailyBalance(gomock.AssignableToTypeOf(args.ctx)).Return("", nil)
			},
			wantErr: false,
		},
		{
			name: "error DoDailyBalance",
			args: args{
				ctx:  context.TODO(),
				date: common.Now(),
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockReconService.EXPECT().DoDailyBalance(gomock.AssignableToTypeOf(args.ctx)).Return("", assert.AnError)
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
			rh := &reportHandler{
				transactionSrv: testHelper.mockTransactionService,
				reconSrv:       testHelper.mockReconService,
			}
			err := rh.DoBalanceReconDaily(tt.args.ctx, tt.args.date, tt.args.flag)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
