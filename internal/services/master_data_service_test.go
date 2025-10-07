package services_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
)

func Test_masterData_CreateOrderType(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx context.Context
		ot  models.OrderType
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success create order type",
			args: args{
				ctx: context.Background(),
				ot: models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(args.ctx).Return([]string{}, nil)
				testHelper.mockMasterData.EXPECT().UpsertOrderType(args.ctx, args.ot).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed create order type, code already exists",
			args: args{
				ctx: context.Background(),
				ot: models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(args.ctx).Return([]string{"SOMETHING"}, nil)
			},
			wantErr: true,
		},
		{
			name: "failed create order type, failed insert repo",
			args: args{
				ctx: context.Background(),
				ot: models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(args.ctx).Return([]string{"ABC"}, nil)
				testHelper.mockMasterData.EXPECT().UpsertOrderType(args.ctx, args.ot).Return(assert.AnError)

			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			err := testHelper.masterDataService.CreateOrderType(tt.args.ctx, tt.args.ot)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_masterData_GetAllOrderType(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx    context.Context
		filter models.FilterMasterData
	}
	tests := []struct {
		name     string
		args     args
		doMock   func(args args)
		wantData []models.OrderType
		wantErr  bool
	}{
		{
			name: "success get all order type",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				res := []models.OrderType{
					{
						OrderTypeCode: "SOMETHING",
						OrderTypeName: "SOMETHING",
					},
				}
				testHelper.mockMasterData.EXPECT().GetListOrderType(args.ctx, args.filter).Return(res, nil)
			},
			wantData: []models.OrderType{
				{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			wantErr: false,
		},
		{
			name: "failed get all order type",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderType(args.ctx, args.filter).Return(nil, assert.AnError)
			},
			wantData: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			data, err := testHelper.masterDataService.GetAllOrderType(tt.args.ctx, tt.args.filter)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func Test_masterData_GetAllTransactionType(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx    context.Context
		filter models.FilterMasterData
	}
	tests := []struct {
		name     string
		args     args
		doMock   func(args args)
		wantData []models.TransactionType
		wantErr  bool
	}{
		{
			name: "success get all transaction type",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				res := []models.TransactionType{
					{
						TransactionTypeCode: "SOMETHING",
						TransactionTypeName: "SOMETHING",
					},
				}
				testHelper.mockMasterData.EXPECT().GetListTransactionType(args.ctx, args.filter).Return(res, nil)
			},
			wantData: []models.TransactionType{
				{
					TransactionTypeCode: "SOMETHING",
					TransactionTypeName: "SOMETHING",
				},
			},
			wantErr: false,
		},
		{
			name: "failed get all transaction type",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListTransactionType(args.ctx, args.filter).Return(nil, assert.AnError)
			},
			wantData: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			data, err := testHelper.masterDataService.GetAllTransactionType(tt.args.ctx, tt.args.filter)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func Test_masterData_UpdateOrderType(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx context.Context
		ot  models.OrderType
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success update order type",
			args: args{
				ctx: context.Background(),
				ot: models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(args.ctx).Return([]string{"SOMETHING"}, nil)
				testHelper.mockMasterData.EXPECT().UpsertOrderType(args.ctx, args.ot).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed update order type, code not exists",
			args: args{
				ctx: context.Background(),
				ot: models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(args.ctx).Return([]string{}, nil)
				testHelper.mockMasterData.EXPECT().UpsertOrderType(args.ctx, args.ot).Return(assert.AnError)

			},
			wantErr: true,
		},
		{
			name: "failed update order type, failed insert repo",
			args: args{
				ctx: context.Background(),
				ot: models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetListOrderTypeCode(args.ctx).Return([]string{"SOMETHING"}, nil)
				testHelper.mockMasterData.EXPECT().UpsertOrderType(args.ctx, args.ot).Return(assert.AnError)

			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			err := testHelper.masterDataService.UpdateOrderType(tt.args.ctx, tt.args.ot)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_masterData_GetOneOrderType(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx           context.Context
		orderTypeCode string
	}
	tests := []struct {
		name     string
		args     args
		doMock   func(args args)
		wantData *models.OrderType
		wantErr  bool
	}{
		{
			name: "success",
			args: args{
				ctx:           context.Background(),
				orderTypeCode: "1001",
			},
			doMock: func(args args) {
				res := &models.OrderType{
					OrderTypeCode: "SOMETHING",
					OrderTypeName: "SOMETHING",
				}
				testHelper.mockMasterData.EXPECT().GetOrderType(args.ctx, args.orderTypeCode).Return(res, nil)
			},
			wantData: &models.OrderType{
				OrderTypeCode: "SOMETHING",
				OrderTypeName: "SOMETHING",
			},
			wantErr: false,
		},
		{
			name: "failed - err repo",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetOrderType(args.ctx, args.orderTypeCode).Return(nil, assert.AnError)
			},
			wantData: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			data, err := testHelper.masterDataService.GetOneOrderType(tt.args.ctx, tt.args.orderTypeCode)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func Test_masterData_GetOneTransactionType(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx                 context.Context
		TransactionTypeCode string
	}
	tests := []struct {
		name     string
		args     args
		doMock   func(args args)
		wantData *models.TransactionType
		wantErr  bool
	}{
		{
			name: "success",
			args: args{
				ctx:                 context.Background(),
				TransactionTypeCode: "1001",
			},
			doMock: func(args args) {
				res := &models.TransactionType{
					TransactionTypeCode: "SOMETHING",
					TransactionTypeName: "SOMETHING",
				}
				testHelper.mockMasterData.EXPECT().GetTransactionType(args.ctx, args.TransactionTypeCode).Return(res, nil)
			},
			wantData: &models.TransactionType{
				TransactionTypeCode: "SOMETHING",
				TransactionTypeName: "SOMETHING",
			},
			wantErr: false,
		},
		{
			name: "failed - err repo",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().GetTransactionType(args.ctx, args.TransactionTypeCode).Return(nil, assert.AnError)
			},
			wantData: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			data, err := testHelper.masterDataService.GetOneTransactionType(tt.args.ctx, tt.args.TransactionTypeCode)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func Test_masterData_GetAllVATConfig(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		args     args
		doMock   func(args args)
		wantData []models.ConfigVatRevenue
		wantErr  bool
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				res := []models.ConfigVatRevenue{
					{
						Percentage: decimal.NewFromFloat(0.11),
					},
				}
				testHelper.mockMasterData.EXPECT().
					GetConfigVATRevenue(args.ctx).
					Return(res, nil)
			},
			wantData: []models.ConfigVatRevenue{
				{
					Percentage: decimal.NewFromFloat(0.11),
				},
			},
			wantErr: false,
		},
		{
			name: "failed - err repo",
			args: args{
				ctx: context.Background(),
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().
					GetConfigVATRevenue(args.ctx).
					Return(nil, assert.AnError)
			},
			wantData: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			data, err := testHelper.masterDataService.GetAllVATConfig(tt.args.ctx)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func Test_masterData_UpsertVATConfig(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx   context.Context
		input []models.ConfigVatRevenue
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success update vat config",
			args: args{
				ctx: context.Background(),
				input: []models.ConfigVatRevenue{
					{
						Percentage: decimal.NewFromFloat(0.11),
					},
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().
					UpsertConfigVATRevenue(args.ctx, args.input).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed update vat config, failed update from repo",
			args: args{
				ctx: context.Background(),
				input: []models.ConfigVatRevenue{
					{
						Percentage: decimal.NewFromFloat(0.11),
					},
				},
			},
			doMock: func(args args) {
				testHelper.mockMasterData.EXPECT().
					UpsertConfigVATRevenue(args.ctx, args.input).
					Return(assert.AnError)

			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			err := testHelper.masterDataService.UpsertVATConfig(tt.args.ctx, tt.args.input)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
