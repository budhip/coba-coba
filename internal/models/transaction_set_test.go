package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestTransactionSet_Calculate(t *testing.T) {
	type fields struct {
		FromAccount string
		ToAccount   string
		Amount      decimal.Decimal
	}
	type args struct {
		accountBalance map[string]Balance
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]Balance
		wantErr bool
	}{
		{
			name: "success calculate balance",
			fields: fields{
				FromAccount: "SOURCE_ACCOUNT",
				ToAccount:   "DESTINATION_ACCOUNT",
				Amount:      decimal.NewFromFloat(420.69),
			},
			args: args{
				accountBalance: map[string]Balance{
					"SOURCE_ACCOUNT":      NewBalance(decimal.NewFromFloat(1000), decimal.Zero),
					"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
				},
			},
			want: map[string]Balance{
				"SOURCE_ACCOUNT":      NewBalance(decimal.NewFromFloat(579.31), decimal.Zero),
				"DESTINATION_ACCOUNT": NewBalance(decimal.NewFromFloat(420.69), decimal.Zero),
			},
		},
		{
			name: "success using same account, balance not increased",
			fields: fields{
				FromAccount: "SOURCE_ACCOUNT",
				ToAccount:   "SOURCE_ACCOUNT",
				Amount:      decimal.NewFromFloat(420.69),
			},
			args: args{
				accountBalance: map[string]Balance{
					"SOURCE_ACCOUNT": NewBalance(decimal.NewFromFloat(1000), decimal.Zero),
				},
			},
			want: map[string]Balance{
				"SOURCE_ACCOUNT": NewBalance(decimal.NewFromFloat(1000), decimal.Zero),
			},
		},
		{
			name: "failed - source account not found",
			fields: fields{
				FromAccount: "SOURCE_ACCOUNT_THAT_DOES_NOT_EXIST",
				ToAccount:   "DESTINATION_ACCOUNT",
				Amount:      decimal.NewFromFloat(420.69),
			},
			args: args{
				accountBalance: map[string]Balance{
					"SOURCE_ACCOUNT":      NewBalance(decimal.NewFromFloat(1000), decimal.Zero),
					"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
				},
			},
			want: map[string]Balance{
				"SOURCE_ACCOUNT":      NewBalance(decimal.NewFromFloat(1000), decimal.Zero),
				"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
			},
			wantErr: true,
		},
		{
			name: "failed - destination account not found",
			fields: fields{
				FromAccount: "SOURCE_ACCOUNT",
				ToAccount:   "DESTINATION_ACCOUNT_THAT_DOES_NOT_EXIST",
				Amount:      decimal.NewFromFloat(420.69),
			},
			args: args{
				accountBalance: map[string]Balance{
					"SOURCE_ACCOUNT":      NewBalance(decimal.NewFromFloat(1000), decimal.Zero),
					"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
				},
			},
			want: map[string]Balance{
				"SOURCE_ACCOUNT":      NewBalance(decimal.NewFromFloat(579.31), decimal.Zero),
				"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
			},
			wantErr: true,
		},
		{
			name: "failed - insufficient funds",
			fields: fields{
				FromAccount: "SOURCE_ACCOUNT",
				ToAccount:   "DESTINATION_ACCOUNT",
				Amount:      decimal.NewFromFloat(420.69),
			},
			args: args{
				accountBalance: map[string]Balance{
					"SOURCE_ACCOUNT":      NewBalance(decimal.Zero, decimal.Zero),
					"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
				},
			},
			want: map[string]Balance{
				"SOURCE_ACCOUNT":      NewBalance(decimal.Zero, decimal.Zero),
				"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
			},
			wantErr: true,
		},
		{
			name: "failed - negative amount",
			fields: fields{
				FromAccount: "SOURCE_ACCOUNT",
				ToAccount:   "DESTINATION_ACCOUNT",
				Amount:      decimal.NewFromFloat(-420.69),
			},
			args: args{
				accountBalance: map[string]Balance{
					"SOURCE_ACCOUNT":      NewBalance(decimal.Zero, decimal.Zero, WithIgnoreBalanceSufficiency()),
					"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
				},
			},
			want: map[string]Balance{
				"SOURCE_ACCOUNT":      NewBalance(decimal.Zero, decimal.Zero, WithIgnoreBalanceSufficiency()),
				"DESTINATION_ACCOUNT": NewBalance(decimal.Zero, decimal.Zero),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trx := TransactionSet{
				FromAccount: tt.fields.FromAccount,
				ToAccount:   tt.fields.ToAccount,
				Amount:      tt.fields.Amount,
			}
			got, err := trx.Calculate(tt.args.accountBalance)
			if (err != nil) != tt.wantErr {
				t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(tt.want, got, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.want, got, balanceComparer()))
			}
		})
	}
}
