package transformer

import (
	"context"
	"reflect"
	"testing"
	"time"

	mock2 "bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mock3 "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	"go.uber.org/mock/gomock"
)

func TestMapTransformer_GetTransformer(t *testing.T) {
	type args struct {
		transactionType string
	}
	tests := []struct {
		name    string
		m       MapTransformer
		args    args
		want    Transformer
		wantErr bool
	}{
		{
			name: "success get transformer",
			m: MapTransformer{
				"ITRTF": &itrtfTransformer{},
			},
			args: args{
				transactionType: "ITRTF",
			},
			want: &itrtfTransformer{},
		},
		{
			name: "failed get transformer",
			m: MapTransformer{
				"ITRTF": &itrtfTransformer{},
			},
			args: args{
				transactionType: "INVALID_TRANSACTION_TYPE",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.m.GetTransformer(tt.args.transactionType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransformer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransformer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapTransformer_Transform(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	mockMasterDataRepo := mock.NewMockMasterDataRepository(mockCtrl)
	mockAccountingClient := mock2.NewMockClient(mockCtrl)
	mockAccountRepo := mock.NewMockAccountRepository(mockCtrl)
	mockTransactionRepo := mock.NewMockTransactionRepository(mockCtrl)
	mockAccountConfigRepo := mock.NewMockAccountConfigRepository(mockCtrl)
	mockWalletTransactionRepo := mock.NewMockWalletTransactionRepository(mockCtrl)
	mockFlag := mock3.NewMockClient(mockCtrl)

	cfg := config.Config{
		AccountConfig: config.AccountConfig{
			SystemAccountNumber: "00000100000000",
			OperationalReceivableAccountNumberByEntity: map[string]string{
				"AMF": "00000000000002",
				"AFA": "00000000000003",
			},
		},
	}

	mt := NewMapTransformer(
		cfg,
		mockMasterDataRepo,
		mockAccountingClient,
		mockAccountRepo,
		mockTransactionRepo,
		mockAccountConfigRepo,
		mockWalletTransactionRepo,
		mockFlag,
	)

	ct := time.Now()

	type args struct {
		in models.WalletTransaction
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success transform",
			args: args{
				in: models.WalletTransaction{
					TransactionType: "ITRTF",
					TransactionTime: ct,
					Status:          models.WalletTransactionStatusSuccess,
					CreatedAt:       ct,
				},
			},
		},
		{
			name: "failed transform",
			args: args{
				in: models.WalletTransaction{
					TransactionType: "INVALID_TRANSACTION_TYPE",
					TransactionTime: ct,
					Status:          models.WalletTransactionStatusSuccess,
					CreatedAt:       ct,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mt.Transform(context.TODO(), tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
