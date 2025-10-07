package transformer

import (
	"reflect"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_getEntityFromMetadata(t *testing.T) {
	type args struct {
		metadata models.WalletMetadata
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success get metadata entity",
			args: args{
				metadata: models.WalletMetadata{
					"entity": "test",
				},
			},
			want: "test",
		},
		{
			name: "failed get metadata entity",
			args: args{
				metadata: models.WalletMetadata{
					"entity": 1,
				},
			},
		},
		{
			name: "missing entity metadata",
			args: args{
				metadata: models.WalletMetadata{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEntityFromMetadata(tt.args.metadata); got != tt.want {
				t.Errorf("getEntityFromMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getPartnerPPOBMetadata(t *testing.T) {
	type args struct {
		metadata models.WalletMetadata
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success get metadata partner",
			args: args{
				metadata: models.WalletMetadata{
					"partner": "test",
				},
			},
			want: "test",
		},
		{
			name: "failed get metadata partner",
			args: args{
				metadata: models.WalletMetadata{
					"partner": 1,
				},
			},
		},
		{
			name: "missing partner metadata",
			args: args{
				metadata: models.WalletMetadata{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPartnerPPOBMetadata(tt.args.metadata); got != tt.want {
				t.Errorf("getPartnerPPOBMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getOrderTime(t *testing.T) {
	ct := time.Now()

	type args struct {
		parentWalletTransaction models.WalletTransaction
	}
	tests := []struct {
		name          string
		args          args
		wantOrderTime time.Time
	}{
		{
			name: "success get order time",
			args: args{
				parentWalletTransaction: models.WalletTransaction{
					CreatedAt: ct,
				},
			},
			wantOrderTime: ct,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOrderTime := getOrderTime(tt.args.parentWalletTransaction); !reflect.DeepEqual(gotOrderTime, tt.wantOrderTime) {
				t.Errorf("getOrderTime() = %v, want %v", gotOrderTime, tt.wantOrderTime)
			}
		})
	}
}

func Test_transformCurrency(t *testing.T) {
	type args struct {
		currency string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success transform currency",
			args: args{
				currency: "IDR",
			},
			want: "IDR",
		},
		{
			name: "use default currency if empty",
			want: models.IDRCurrency,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transformCurrency(tt.args.currency); got != tt.want {
				t.Errorf("transformCurrency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transformWalletTransactionStatus(t *testing.T) {
	type args struct {
		statusWallet models.WalletTransactionStatus
	}
	tests := []struct {
		name    string
		args    args
		want    models.TransactionStatus
		wantErr bool
	}{
		{
			name: "success transform wallet transaction status",
			args: args{
				statusWallet: models.WalletTransactionStatusSuccess,
			},
			want: models.TransactionStatusSuccess,
		},
		{
			name: "failed transform wallet transaction status",
			args: args{
				statusWallet: "invalid",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformWalletTransactionStatus(tt.args.statusWallet)
			if (err != nil) != tt.wantErr {
				t.Errorf("transformWalletTransactionStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transformWalletTransactionStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAccountNumberFromConfig(t *testing.T) {
	type args struct {
		ac  map[string]string
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "success get account number",
			args: args{
				ac: map[string]string{
					"amf": "000000000000",
					"afa": "111111111111",
				},
				key: "AMF",
			},
			want: "000000000000",
		},
		{
			name: "failed get account number - key not found",
			args: args{
				ac: map[string]string{
					"amf": "000000000000",
				},
				key: "afa",
			},
			wantErr: true,
		},
		{
			name: "failed get account number - account number is empty",
			args: args{
				ac: map[string]string{
					"amf": "",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAccountNumberFromConfig(tt.args.ac, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAccountNumberFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getAccountNumberFromConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func decimalComparer() cmp.Option {
	return cmp.Comparer(func(x, y decimal.Decimal) bool {
		return x.Equal(y)
	})
}

func Test_calculateVAT(t *testing.T) {
	currentTime := time.Now()

	type args struct {
		amount          decimal.Decimal
		transactionTime time.Time
		opts            []VATOpts
	}
	tests := []struct {
		name    string
		args    args
		want    decimal.Decimal
		wantErr bool
	}{
		{
			name: "success calculate VAT (use default value)",
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want: decimal.NewFromFloat(10),
		},
		{
			name: "success calculate VAT using config manager",
			args: args{
				amount:          decimal.NewFromFloat(100),
				transactionTime: currentTime,
				opts: []VATOpts{
					WithVATRevenueConfig([]models.ConfigVatRevenue{
						{
							// active config
							Percentage: decimal.NewFromFloat(0.12),
							StartTime:  currentTime.Add(-1 * time.Hour),
							EndTime:    currentTime.Add(1 * time.Hour),
						},
						{
							Percentage: decimal.NewFromFloat(0.15),
							StartTime:  currentTime.Add(1 * time.Hour),
							EndTime:    currentTime.Add(2 * time.Hour),
						},
					}),
				},
			},
			want: decimal.NewFromFloat(11),
		},
		{
			name: "failed no active config",
			args: args{
				amount:          decimal.NewFromFloat(100),
				transactionTime: currentTime.Add(365 * 24 * time.Hour),
				opts: []VATOpts{
					WithVATRevenueConfig([]models.ConfigVatRevenue{
						{
							Percentage: decimal.NewFromFloat(0.12),
							StartTime:  currentTime.Add(-2 * time.Hour),
							EndTime:    currentTime.Add(-1 * time.Hour),
						},
						{
							Percentage: decimal.NewFromFloat(0.15),
							StartTime:  currentTime.Add(1 * time.Hour),
							EndTime:    currentTime.Add(2 * time.Hour),
						},
					}),
				},
			},
			wantErr: true,
			want:    decimal.Zero,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateVAT(tt.args.amount, tt.args.transactionTime, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateVAT() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(tt.want, got, decimalComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.want, got, decimalComparer()))
			}
		})
	}
}

func Test_getLoanAccountNumber(t *testing.T) {
	type args struct {
		metadata models.WalletMetadata
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success get metadata loanAccountNumber",
			args: args{
				metadata: models.WalletMetadata{
					"loanAccountNumber": "test",
				},
			},
			want: "test",
		},
		{
			name: "failed get metadata loanAccountNumber",
			args: args{
				metadata: models.WalletMetadata{
					"loanAccountNumber": 1,
				},
			},
		},
		{
			name: "missing entity loanAccountNumber",
			args: args{
				metadata: models.WalletMetadata{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLoanAccountNumber(tt.args.metadata); got != tt.want {
				t.Errorf("getLoanAccountNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getLoanIds(t *testing.T) {
	type args struct {
		metadata models.WalletMetadata
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "success get metadata loanIds",
			args: args{
				metadata: models.WalletMetadata{
					"loanIds": []any{"1", "2", "3"},
				},
			},
			want: []string{"1", "2", "3"},
		},
		{
			name: "failed get metadata loanIds - invalid key",
			args: args{
				metadata: models.WalletMetadata{
					"invalid_key_here": []any{"1", "2", "3"},
				},
			},
			wantErr: true,
		},
		{
			name: "failed get metadata loanIds - invalid values",
			args: args{
				metadata: models.WalletMetadata{
					"loanIds": []any{"1", 2, "3"},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLoanIds(tt.args.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLoanIds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLoanIds() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutateMetadataByAccountEntity(t *testing.T) {
	cfg := config.Config{
		AccountConfig: config.AccountConfig{
			MapAccountEntity: map[string]string{
				"001": "AMF",
				"003": "AFA",
				"005": "AWF",
			},
		},
	}

	type args struct {
		EntityCode string
		Meta       models.WalletMetadata
	}

	tests := []struct {
		name string
		args args
		want models.WalletMetadata
	}{
		{
			name: "WHEN mutate from account entity 001 THEN should have AMF entity",
			args: args{
				EntityCode: "001",
				Meta:       models.WalletMetadata{},
			},
			want: models.WalletMetadata{
				"entity": "AMF",
			},
		},
		{
			name: "WHEN mutate to account entity 001 THEN should have AMF entity",
			args: args{
				EntityCode: "001",
				Meta:       models.WalletMetadata{},
			},
			want: models.WalletMetadata{
				"entity": "AMF",
			},
		},
		{
			name: "WHEN mutate to account entity 005 THEN should have AWF entity",
			args: args{
				EntityCode: "005",
				Meta:       models.WalletMetadata{},
			},
			want: models.WalletMetadata{
				"entity": "AWF",
			},
		},
		{
			name: "WHEN mutate from account entity 005 THEN should have AWF entity",
			args: args{
				EntityCode: "005",
				Meta:       models.WalletMetadata{},
			},
			want: models.WalletMetadata{
				"entity": "AWF",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			baseTransformer := baseWalletTransactionTransformer{config: cfg}
			got := baseTransformer.MutateMetadataByAccountEntity(tc.args.EntityCode, tc.args.Meta)
			assert.Equal(t, tc.want, got, tc.name)
		})
	}
}
